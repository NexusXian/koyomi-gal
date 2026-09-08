package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"backend/internal/message/dto"
	"backend/internal/message/model"
	"backend/internal/message/repository"
	"backend/internal/realtime"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

var (
	ErrConversationNotFound     = repository.ErrConversationNotFound
	ErrConversationAccessDenied = repository.ErrConversationAccessDenied
	ErrMessageNotFound          = repository.ErrMessageNotFound
	ErrMessageAccessDenied      = repository.ErrMessageAccessDenied
	ErrUserNotFound             = repository.ErrUserNotFound
	ErrMessageRateLimited       = repository.ErrMessageRateLimit
	ErrMessageRateLimit         = ErrMessageRateLimited
	ErrConversationRateLimit    = repository.ErrConversationRateLimit
	ErrRateLimitUnavailable     = repository.ErrRateLimitUnavailable
	ErrInvalidMessage           = errors.New("invalid message")
	ErrMessageEmpty             = errors.New("message is empty")
	ErrMessageTooLong           = errors.New("message is too long")
	ErrInvalidMessageType       = errors.New("invalid message type")
	ErrInvalidMessagePermission = errors.New("invalid message permission")
	ErrInvalidCursor            = errors.New("invalid conversation cursor")
	ErrCannotMessageSelf        = errors.New("cannot message self")
	ErrMessagePermissionDenied  = errors.New("message permission denied")
	ErrUserBlocked              = errors.New("UserBlocked")
	ErrMessageNotOwner          = ErrMessageAccessDenied
	ErrInvalidUser              = errors.New("invalid message user")
)

const (
	defaultConversationLimit = 20
	defaultMessageLimit      = 30
	maxListLimit             = 100
	maxMessageCodePoints     = 2000
)

type MessageService struct {
	messages  *repository.MessageRepository
	limiter   *repository.RateLimiter
	publisher realtime.Publisher
}

func NewMessageService(
	messages *repository.MessageRepository,
	limiter *repository.RateLimiter,
	publisher realtime.Publisher,
) *MessageService {
	return &MessageService{messages: messages, limiter: limiter, publisher: publisher}
}

func (s *MessageService) CreateConversation(
	ctx context.Context,
	actorID uint,
	req *dto.CreateConversationRequest,
) (*dto.ConversationData, error) {
	if actorID == 0 || req == nil || req.UserID == 0 {
		return nil, ErrInvalidUser
	}
	if actorID == req.UserID {
		return nil, ErrCannotMessageSelf
	}
	lowID, highID := canonicalPair(actorID, req.UserID)
	var conversationID uint
	err := s.messages.Transaction(ctx, func(tx *repository.MessageRepository) error {
		if err := tx.LockUsers(ctx, lowID, highID); err != nil {
			return err
		}
		if err := canContact(ctx, tx, actorID, req.UserID); err != nil {
			return err
		}
		existingID, err := tx.FindDirectConversationID(ctx, lowID, highID)
		if err != nil {
			return err
		}
		if existingID != 0 {
			conversationID = existingID
			return tx.ReviveMember(ctx, conversationID, actorID)
		}
		if err := s.limiter.AllowNewConversation(ctx, actorID); err != nil {
			return err
		}
		conversation := &model.Conversation{
			ConversationType: model.ConversationTypeDirect,
		}
		if err := tx.CreateDirectConversation(ctx, conversation, lowID, highID); err != nil {
			return err
		}
		conversationID = conversation.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	record, err := s.messages.Conversation(ctx, actorID, conversationID)
	if err != nil {
		return nil, err
	}
	data := dto.NewConversationData(record)
	return &data, nil
}

func (s *MessageService) ListConversations(
	ctx context.Context,
	actorID uint,
	cursorValue string,
	limit int,
) (*dto.ConversationListData, error) {
	limit = normalizeLimit(limit, defaultConversationLimit)
	cursor, err := decodeCursor(cursorValue)
	if err != nil {
		return nil, err
	}
	records, err := s.messages.ListConversations(ctx, actorID, cursor, limit+1)
	if err != nil {
		return nil, err
	}
	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	list := make([]dto.ConversationData, 0, len(records))
	for i := range records {
		list = append(list, dto.NewConversationData(&records[i]))
	}
	nextCursor := ""
	if hasMore && len(records) > 0 {
		nextCursor, err = encodeCursor(records[len(records)-1].SortAt, records[len(records)-1].ID)
		if err != nil {
			return nil, err
		}
	}
	return &dto.ConversationListData{List: list, NextCursor: nextCursor, HasMore: hasMore}, nil
}

func (s *MessageService) ListMessages(
	ctx context.Context,
	actorID, conversationID, beforeID uint,
	limit int,
) (*dto.MessageListData, error) {
	limit = normalizeLimit(limit, defaultMessageLimit)
	messages, err := s.messages.ListMessages(ctx, actorID, conversationID, beforeID, limit+1)
	if err != nil {
		return nil, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	nextCursor := uint(0)
	if hasMore && len(messages) > 0 {
		nextCursor = messages[len(messages)-1].ID
	}
	list := make([]dto.MessageData, len(messages))
	for i := range messages {
		list[len(messages)-1-i] = dto.NewMessageData(&messages[i])
	}
	return &dto.MessageListData{List: list, NextCursor: nextCursor, HasMore: hasMore}, nil
}

func (s *MessageService) SendMessage(
	ctx context.Context,
	actorID, conversationID uint,
	req *dto.SendMessageRequest,
) (*dto.MessageData, error) {
	messageType, content, err := validateMessage(req)
	if err != nil {
		return nil, err
	}
	message := &model.Message{
		ConversationID: conversationID,
		SenderID:       actorID,
		MessageType:    messageType,
		Content:        content,
	}
	err = s.messages.Transaction(ctx, func(tx *repository.MessageRepository) error {
		peerID, err := tx.LockConversationPeer(ctx, actorID, conversationID)
		if err != nil {
			return err
		}
		if err := canContact(ctx, tx, actorID, peerID); err != nil {
			return err
		}
		if err := s.limiter.AllowMessage(ctx, actorID); err != nil {
			return err
		}
		message.ReceiverID = peerID
		now := time.Now()
		message.CreatedAt = now
		message.UpdatedAt = now
		if err := tx.CreateMessage(ctx, message); err != nil {
			return err
		}
		return tx.ApplySentMessage(ctx, message)
	})
	if err != nil {
		return nil, err
	}
	s.publish(ctx, []uint{message.SenderID, message.ReceiverID}, realtime.Event{
		Type: realtime.EventMessageCreated,
		Data: realtime.MessageCreatedData{
			ConversationID: message.ConversationID,
			Message: realtime.MessageCreatedMessage{
				ID: message.ID, ConversationID: message.ConversationID, SenderID: message.SenderID,
				Type: message.MessageType, Content: message.Content, IsDeleted: false, CreatedAt: message.CreatedAt,
			},
		},
	})
	created, err := s.messages.Message(ctx, message.ID)
	if err != nil {
		return nil, err
	}
	data := dto.NewMessageData(created)
	return &data, nil
}

func (s *MessageService) MarkRead(ctx context.Context, actorID, conversationID, messageID uint) error {
	if messageID == 0 {
		return ErrInvalidMessage
	}
	var peerID uint
	err := s.messages.Transaction(ctx, func(tx *repository.MessageRepository) error {
		var err error
		peerID, err = tx.LockConversationPeer(ctx, actorID, conversationID)
		if err != nil {
			return err
		}
		if err := tx.ValidateMessageInConversation(ctx, messageID, conversationID); err != nil {
			return err
		}
		return tx.MarkRead(ctx, actorID, conversationID, messageID)
	})
	if err != nil {
		return err
	}
	s.publish(ctx, []uint{actorID, peerID}, realtime.Event{
		Type: realtime.EventConversationRead,
		Data: realtime.ConversationReadData{ConversationID: conversationID, UserID: actorID, MessageID: messageID},
	})
	return nil
}

func (s *MessageService) DeleteConversation(ctx context.Context, actorID, conversationID uint) error {
	return s.messages.Transaction(ctx, func(tx *repository.MessageRepository) error {
		if _, err := tx.LockConversationPeer(ctx, actorID, conversationID); err != nil {
			return err
		}
		return tx.HideConversation(ctx, actorID, conversationID)
	})
}

func (s *MessageService) DeleteMessage(ctx context.Context, actorID, messageID uint) error {
	var conversationID, peerID uint
	err := s.messages.Transaction(ctx, func(tx *repository.MessageRepository) error {
		message, err := tx.LockMessageForDelete(ctx, actorID, messageID)
		if err != nil {
			return err
		}
		if message.SenderID != actorID {
			return ErrMessageAccessDenied
		}
		conversationID = message.ConversationID
		peerID = message.ReceiverID
		if message.DeletedAt != nil {
			return nil
		}
		if err := tx.SoftDeleteMessage(ctx, message); err != nil {
			return err
		}
		var conversation *model.ConversationRecord
		conversation, err = tx.Conversation(ctx, actorID, message.ConversationID)
		if err != nil {
			return err
		}
		if conversation.LastMessageID != nil && *conversation.LastMessageID == message.ID {
			return tx.RecomputeLastMessageAllowEmpty(ctx, message.ConversationID)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.publish(ctx, []uint{actorID, peerID}, realtime.Event{
		Type: realtime.EventMessageDeleted,
		Data: realtime.MessageDeletedData{ConversationID: conversationID, MessageID: messageID},
	})
	return nil
}

func (s *MessageService) UnreadCount(ctx context.Context, actorID uint) (int64, error) {
	return s.messages.UnreadCount(ctx, actorID)
}

func (s *MessageService) GetSettings(ctx context.Context, actorID uint) (*dto.MessageSettingsData, error) {
	permission, err := s.messages.MessagePermission(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return &dto.MessageSettingsData{Permission: permission}, nil
}

func (s *MessageService) UpdateSettings(
	ctx context.Context,
	actorID uint,
	req *dto.UpdateMessageSettingsRequest,
) (*dto.MessageSettingsData, error) {
	if req == nil || !validPermission(req.Permission) {
		return nil, ErrInvalidMessagePermission
	}
	if err := s.messages.SetMessagePermission(ctx, actorID, req.Permission); err != nil {
		return nil, err
	}
	return &dto.MessageSettingsData{Permission: req.Permission}, nil
}

func (s *MessageService) BlockUser(ctx context.Context, actorID, blockedID uint) error {
	if actorID == 0 || blockedID == 0 {
		return ErrInvalidUser
	}
	if actorID == blockedID {
		return ErrCannotMessageSelf
	}
	exists, err := s.messages.UserExists(ctx, blockedID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}
	return s.messages.BlockUser(ctx, actorID, blockedID)
}

func (s *MessageService) UnblockUser(ctx context.Context, actorID, blockedID uint) error {
	if actorID == 0 || blockedID == 0 {
		return ErrInvalidUser
	}
	if actorID == blockedID {
		return ErrCannotMessageSelf
	}
	exists, err := s.messages.UserExists(ctx, blockedID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}
	return s.messages.UnblockUser(ctx, actorID, blockedID)
}

func (s *MessageService) publish(ctx context.Context, userIDs []uint, event realtime.Event) {
	if s.publisher == nil {
		return
	}
	publishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if err := s.publisher.Publish(publishCtx, userIDs, event); err != nil {
		logger.Error("publish message realtime event", zap.String("event_type", event.Type), zap.Error(err))
	}
}

func canContact(
	ctx context.Context,
	repository *repository.MessageRepository,
	actorID, targetID uint,
) error {
	blocked, err := repository.IsBlocked(ctx, actorID, targetID)
	if err != nil {
		return err
	}
	if blocked {
		return ErrUserBlocked
	}
	permission, err := repository.MessagePermission(ctx, targetID)
	if err != nil {
		return err
	}
	if permission != model.MessagePermissionEveryone {
		return ErrMessagePermissionDenied
	}
	return nil
}

func validateMessage(req *dto.SendMessageRequest) (string, string, error) {
	if req == nil {
		return "", "", ErrMessageEmpty
	}
	messageType := strings.TrimSpace(req.Type)
	if messageType == "" {
		messageType = model.MessageTypeText
	}
	if messageType != model.MessageTypeText {
		return "", "", ErrInvalidMessageType
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return "", "", ErrMessageEmpty
	}
	if !utf8.ValidString(content) {
		return "", "", ErrInvalidMessage
	}
	if utf8.RuneCountInString(content) > maxMessageCodePoints {
		return "", "", ErrMessageTooLong
	}
	return messageType, content, nil
}

func validPermission(permission string) bool {
	return permission == model.MessagePermissionEveryone || permission == model.MessagePermissionNone
}

func normalizeLimit(limit, fallback int) int {
	if limit < 1 {
		return fallback
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}

func canonicalPair(firstID, secondID uint) (uint, uint) {
	if firstID < secondID {
		return firstID, secondID
	}
	return secondID, firstID
}

type cursorPayload struct {
	SortAt time.Time `json:"sort_at"`
	ID     uint      `json:"id"`
}

func encodeCursor(sortAt time.Time, id uint) (string, error) {
	encoded, err := json.Marshal(cursorPayload{SortAt: sortAt, ID: id})
	if err != nil {
		return "", fmt.Errorf("encode conversation cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeCursor(value string) (*repository.ConversationCursor, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	var payload cursorPayload
	if err := json.Unmarshal(decoded, &payload); err != nil || payload.ID == 0 || payload.SortAt.IsZero() {
		return nil, ErrInvalidCursor
	}
	return &repository.ConversationCursor{SortAt: payload.SortAt, ID: payload.ID}, nil
}
