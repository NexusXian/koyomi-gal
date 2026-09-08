package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	imageModel "backend/internal/image/model"
	"backend/internal/message/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrConversationNotFound     = errors.New("conversation not found")
	ErrConversationAccessDenied = errors.New("conversation access denied")
	ErrMessageNotFound          = errors.New("message not found")
	ErrMessageAccessDenied      = errors.New("message access denied")
	ErrUserNotFound             = errors.New("user not found")
)

type ConversationCursor struct {
	SortAt time.Time
	ID     uint
}

type MessageRepository struct {
	db        *gorm.DB
	publicURL string
}

func NewMessageRepository(db *gorm.DB, publicURL string) *MessageRepository {
	return &MessageRepository{db: db, publicURL: strings.TrimRight(publicURL, "/")}
}

func (r *MessageRepository) Transaction(ctx context.Context, fn func(*MessageRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&MessageRepository{db: tx, publicURL: r.publicURL})
	})
}

func (r *MessageRepository) LockUsers(ctx context.Context, firstID, secondID uint) error {
	ids := make([]uint, 0, 2)
	err := r.db.WithContext(ctx).Raw(
		"SELECT id FROM users WHERE id IN (?, ?) ORDER BY id FOR UPDATE",
		firstID, secondID,
	).Scan(&ids).Error
	if err != nil {
		return fmt.Errorf("lock message users: %w", err)
	}
	if len(ids) != 2 {
		return ErrUserNotFound
	}
	return nil
}

func (r *MessageRepository) UserExists(ctx context.Context, userID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check message user: %w", err)
	}
	return count > 0, nil
}

func (r *MessageRepository) MessagePermission(ctx context.Context, userID uint) (string, error) {
	var permission string
	err := r.db.WithContext(ctx).Raw(`
SELECT COALESCE(settings.message_permission, 'everyone')
FROM users
LEFT JOIN user_privacy_settings AS settings ON settings.user_id = users.id
WHERE users.id = ?
`, userID).Scan(&permission).Error
	if err != nil {
		return "", fmt.Errorf("find message permission: %w", err)
	}
	if permission == "" {
		return "", ErrUserNotFound
	}
	return permission, nil
}

func (r *MessageRepository) IsBlocked(ctx context.Context, firstID, secondID uint) (bool, error) {
	var blocked bool
	err := r.db.WithContext(ctx).Raw(`
SELECT EXISTS (
    SELECT 1 FROM user_blocks
    WHERE (blocker_id = ? AND blocked_id = ?)
       OR (blocker_id = ? AND blocked_id = ?)
)
`, firstID, secondID, secondID, firstID).Scan(&blocked).Error
	if err != nil {
		return false, fmt.Errorf("check message block: %w", err)
	}
	return blocked, nil
}

func (r *MessageRepository) FindDirectConversationID(ctx context.Context, lowID, highID uint) (uint, error) {
	var direct model.DirectConversation
	err := r.db.WithContext(ctx).
		Where("user_low_id = ? AND user_high_id = ?", lowID, highID).
		Take(&direct).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("find direct conversation: %w", err)
	}
	return direct.ConversationID, nil
}

func (r *MessageRepository) CreateDirectConversation(
	ctx context.Context,
	conversation *model.Conversation,
	lowID, highID uint,
) error {
	if err := r.db.WithContext(ctx).Create(conversation).Error; err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	now := time.Now()
	members := []model.ConversationMember{
		{ConversationID: conversation.ID, UserID: lowID, JoinedAt: now, UpdatedAt: now},
		{ConversationID: conversation.ID, UserID: highID, JoinedAt: now, UpdatedAt: now},
	}
	if err := r.db.WithContext(ctx).Create(&members).Error; err != nil {
		return fmt.Errorf("create conversation members: %w", err)
	}
	direct := model.DirectConversation{
		ConversationID: conversation.ID,
		UserLowID:      lowID,
		UserHighID:     highID,
	}
	if err := r.db.WithContext(ctx).Create(&direct).Error; err != nil {
		return fmt.Errorf("create direct conversation: %w", err)
	}
	return nil
}

func (r *MessageRepository) ReviveMember(ctx context.Context, conversationID, userID uint) error {
	result := r.db.WithContext(ctx).Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]any{"deleted_at": nil, "updated_at": gorm.Expr("NOW()")})
	if result.Error != nil {
		return fmt.Errorf("revive conversation member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrConversationAccessDenied
	}
	return nil
}

func (r *MessageRepository) Conversation(ctx context.Context, actorID, conversationID uint) (*model.ConversationRecord, error) {
	query := r.conversationQuery(ctx, actorID).
		Where("conversations.id = ?", conversationID)
	var record model.ConversationRecord
	result := query.Scan(&record)
	if result.Error != nil {
		return nil, fmt.Errorf("find conversation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		exists, err := r.conversationExists(ctx, conversationID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrConversationNotFound
		}
		return nil, ErrConversationAccessDenied
	}
	return &record, nil
}

func (r *MessageRepository) ListConversations(
	ctx context.Context,
	actorID uint,
	cursor *ConversationCursor,
	limit int,
) ([]model.ConversationRecord, error) {
	query := r.conversationQuery(ctx, actorID).
		Where("conversation_members.deleted_at IS NULL")
	if cursor != nil {
		query = query.Where(`
(COALESCE(conversations.last_message_at, conversations.updated_at, conversations.created_at), conversations.id) < (?, ?)
`, cursor.SortAt, cursor.ID)
	}
	records := make([]model.ConversationRecord, 0)
	err := query.Order("COALESCE(conversations.last_message_at, conversations.updated_at, conversations.created_at) DESC").
		Order("conversations.id DESC").Limit(limit).Scan(&records).Error
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	return records, nil
}

func (r *MessageRepository) conversationQuery(ctx context.Context, actorID uint) *gorm.DB {
	return r.db.WithContext(ctx).Table("conversations").
		Select(`conversations.id, conversations.conversation_type, conversation_members.unread_count,
conversations.last_message_at,
COALESCE(conversations.last_message_at, conversations.updated_at, conversations.created_at) AS sort_at,
conversations.created_at, conversations.updated_at,
peer.id AS peer_id, peer.username AS peer_username,
COALESCE(NULLIF(peer_profile.display_name, ''), peer.username) AS peer_display_name,
COALESCE(CASE WHEN peer_avatar.object_key IS NOT NULL
    THEN CAST(? AS text) || '/' || peer_avatar.object_key ELSE peer.avatar END, '') AS peer_avatar_url,
EXISTS (
    SELECT 1 FROM user_blocks
    WHERE (blocker_id = ? AND blocked_id = peer.id)
       OR (blocker_id = peer.id AND blocked_id = ?)
) AS is_blocked,
(
    COALESCE(peer_privacy.message_permission, 'everyone') = 'everyone'
    AND NOT EXISTS (
        SELECT 1 FROM user_blocks
        WHERE (blocker_id = ? AND blocked_id = peer.id)
           OR (blocker_id = peer.id AND blocked_id = ?)
    )
) AS can_send,
last_message.id AS last_message_id, last_message.sender_id AS last_sender_id,
last_message.message_type AS last_message_type, last_message.content AS last_message_content,
last_message.deleted_at AS last_message_deleted_at,
last_message.created_at AS last_message_created_at`,
			r.publicURL, actorID, actorID, actorID, actorID).
		Joins("JOIN conversation_members ON conversation_members.conversation_id = conversations.id AND conversation_members.user_id = ?", actorID).
		Joins("JOIN direct_conversations ON direct_conversations.conversation_id = conversations.id").
		Joins(`JOIN users AS peer ON peer.id = CASE
WHEN direct_conversations.user_low_id = ? THEN direct_conversations.user_high_id
ELSE direct_conversations.user_low_id END`, actorID).
		Joins("LEFT JOIN user_profiles AS peer_profile ON peer_profile.user_id = peer.id").
		Joins("LEFT JOIN user_privacy_settings AS peer_privacy ON peer_privacy.user_id = peer.id").
		Joins(fmt.Sprintf(`LEFT JOIN image_assets AS peer_avatar
ON peer_avatar.id = peer_profile.avatar_asset_id
AND peer_avatar.user_id = peer.id AND peer_avatar.status = %d`, imageModel.ImageStatusActive)).
		Joins("LEFT JOIN messages AS last_message ON last_message.id = conversations.last_message_id")
}

func (r *MessageRepository) LockConversationPeer(ctx context.Context, actorID, conversationID uint) (uint, error) {
	var lockedID uint
	lock := r.db.WithContext(ctx).Raw(
		"SELECT id FROM conversations WHERE id = ? FOR UPDATE",
		conversationID,
	).Scan(&lockedID)
	if lock.Error != nil {
		return 0, fmt.Errorf("lock conversation: %w", lock.Error)
	}
	if lock.RowsAffected == 0 {
		return 0, ErrConversationNotFound
	}

	var peerID uint
	result := r.db.WithContext(ctx).Raw(`
SELECT peer_member.user_id
FROM conversation_members AS me
JOIN conversation_members AS peer_member
	ON peer_member.conversation_id = me.conversation_id AND peer_member.user_id <> ?
JOIN direct_conversations AS direct ON direct.conversation_id = me.conversation_id
WHERE me.conversation_id = ? AND me.user_id = ?
FOR UPDATE OF me, peer_member
`, actorID, conversationID, actorID).Scan(&peerID)
	if result.Error != nil {
		return 0, fmt.Errorf("lock conversation: %w", result.Error)
	}
	if result.RowsAffected == 0 || peerID == 0 {
		return 0, ErrConversationAccessDenied
	}
	return peerID, nil
}

func (r *MessageRepository) CreateMessage(ctx context.Context, message *model.Message) error {
	if err := r.db.WithContext(ctx).Create(message).Error; err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

func (r *MessageRepository) ApplySentMessage(ctx context.Context, message *model.Message) error {
	result := r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("id = ?", message.ConversationID).
		Updates(map[string]any{
			"last_message_id": message.ID,
			"last_message_at": message.CreatedAt,
			"updated_at":      message.CreatedAt,
		})
	if result.Error != nil {
		return fmt.Errorf("update conversation last message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrConversationNotFound
	}
	if err := r.db.WithContext(ctx).Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", message.ConversationID, message.SenderID).
		Updates(map[string]any{"deleted_at": nil, "updated_at": message.CreatedAt}).Error; err != nil {
		return fmt.Errorf("revive sender conversation: %w", err)
	}
	receiver := r.db.WithContext(ctx).Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", message.ConversationID, message.ReceiverID).
		Updates(map[string]any{
			"unread_count": gorm.Expr("unread_count + 1"),
			"deleted_at":   nil,
			"updated_at":   message.CreatedAt,
		})
	if receiver.Error != nil {
		return fmt.Errorf("increment message unread count: %w", receiver.Error)
	}
	if receiver.RowsAffected == 0 {
		return ErrConversationNotFound
	}
	return nil
}

func (r *MessageRepository) Message(ctx context.Context, messageID uint) (*model.Message, error) {
	var message model.Message
	result := r.messageQuery(ctx).Where("messages.id = ?", messageID).Scan(&message)
	if result.Error != nil {
		return nil, fmt.Errorf("find message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrMessageNotFound
	}
	return &message, nil
}

func (r *MessageRepository) ListMessages(
	ctx context.Context,
	actorID, conversationID, beforeID uint,
	limit int,
) ([]model.Message, error) {
	var membership int64
	if err := r.db.WithContext(ctx).Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, actorID).
		Count(&membership).Error; err != nil {
		return nil, fmt.Errorf("check conversation membership: %w", err)
	}
	if membership == 0 {
		exists, err := r.conversationExists(ctx, conversationID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrConversationNotFound
		}
		return nil, ErrConversationAccessDenied
	}
	query := r.messageQuery(ctx).Where("messages.conversation_id = ?", conversationID)
	if beforeID > 0 {
		query = query.Where("messages.id < ?", beforeID)
	}
	messages := make([]model.Message, 0)
	if err := query.Order("messages.id DESC").Limit(limit).Scan(&messages).Error; err != nil {
		return nil, fmt.Errorf("list conversation messages: %w", err)
	}
	return messages, nil
}

func (r *MessageRepository) messageQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("messages").
		Select(`messages.*,
sender.username AS sender_username,
COALESCE(NULLIF(sender_profile.display_name, ''), sender.username) AS sender_display_name,
COALESCE(CASE WHEN sender_avatar.object_key IS NOT NULL
    THEN CAST(? AS text) || '/' || sender_avatar.object_key ELSE sender.avatar END, '') AS sender_avatar_url`, r.publicURL).
		Joins("JOIN users AS sender ON sender.id = messages.sender_id").
		Joins("LEFT JOIN user_profiles AS sender_profile ON sender_profile.user_id = sender.id").
		Joins(fmt.Sprintf(`LEFT JOIN image_assets AS sender_avatar
ON sender_avatar.id = sender_profile.avatar_asset_id
AND sender_avatar.user_id = sender.id AND sender_avatar.status = %d`, imageModel.ImageStatusActive))
}

func (r *MessageRepository) ValidateMessageInConversation(ctx context.Context, messageID, conversationID uint) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Message{}).
		Where("id = ? AND conversation_id = ?", messageID, conversationID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("validate read message: %w", err)
	}
	if count == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *MessageRepository) MarkRead(ctx context.Context, actorID, conversationID, messageID uint) error {
	result := r.db.WithContext(ctx).Exec(`
UPDATE conversation_members
SET last_read_message_id = CASE
        WHEN COALESCE(last_read_message_id, 0) < ? THEN ?
        ELSE last_read_message_id
    END,
    unread_count = CASE
        WHEN COALESCE(last_read_message_id, 0) < ? THEN (
            SELECT COUNT(*) FROM messages
            WHERE conversation_id = ? AND receiver_id = ? AND id > ?
        )
        ELSE unread_count
    END,
    updated_at = NOW()
WHERE conversation_id = ? AND user_id = ?
`, messageID, messageID, messageID, conversationID, actorID, messageID, conversationID, actorID)
	if result.Error != nil {
		return fmt.Errorf("mark conversation read: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrConversationAccessDenied
	}
	return nil
}

func (r *MessageRepository) HideConversation(ctx context.Context, actorID, conversationID uint) error {
	result := r.db.WithContext(ctx).Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, actorID).
		Updates(map[string]any{
			"deleted_at":   gorm.Expr("NOW()"),
			"unread_count": 0,
			"updated_at":   gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return fmt.Errorf("hide conversation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrConversationAccessDenied
	}
	return nil
}

func (r *MessageRepository) LockMessageForDelete(ctx context.Context, actorID, messageID uint) (*model.Message, error) {
	var message model.Message
	result := r.db.WithContext(ctx).Raw(`
SELECT messages.*
FROM messages
JOIN conversations AS conversation ON conversation.id = messages.conversation_id
WHERE messages.id = ?
FOR UPDATE OF conversation, messages
`, messageID).Scan(&message)
	if result.Error != nil {
		return nil, fmt.Errorf("lock message for delete: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrMessageNotFound
	}
	var membership int64
	if err := r.db.WithContext(ctx).Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", message.ConversationID, actorID).
		Count(&membership).Error; err != nil {
		return nil, fmt.Errorf("check message access: %w", err)
	}
	if membership == 0 {
		return nil, ErrMessageAccessDenied
	}
	return &message, nil
}

func (r *MessageRepository) conversationExists(ctx context.Context, conversationID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("id = ?", conversationID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check conversation existence: %w", err)
	}
	return count > 0, nil
}

func (r *MessageRepository) SoftDeleteMessage(ctx context.Context, message *model.Message) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.Message{}).
		Where("id = ? AND deleted_at IS NULL", message.ID).
		Updates(map[string]any{"content": "", "deleted_at": now, "updated_at": now})
	if result.Error != nil {
		return fmt.Errorf("soft delete message: %w", result.Error)
	}
	if result.RowsAffected == 0 && message.DeletedAt == nil {
		return ErrMessageNotFound
	}
	return nil
}

func (r *MessageRepository) RecomputeLastMessageAllowEmpty(ctx context.Context, conversationID uint) error {
	var latest struct {
		ID        uint
		CreatedAt time.Time
	}
	result := r.db.WithContext(ctx).Model(&model.Message{}).
		Select("id, created_at").
		Where("conversation_id = ? AND deleted_at IS NULL", conversationID).
		Order("id DESC").Limit(1).Scan(&latest)
	if result.Error != nil {
		return fmt.Errorf("find replacement last message: %w", result.Error)
	}
	values := map[string]any{"updated_at": gorm.Expr("NOW()")}
	if result.RowsAffected == 0 {
		values["last_message_id"] = nil
		values["last_message_at"] = nil
	} else {
		values["last_message_id"] = latest.ID
		values["last_message_at"] = latest.CreatedAt
	}
	if err := r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("id = ?", conversationID).Updates(values).Error; err != nil {
		return fmt.Errorf("replace conversation last message: %w", err)
	}
	return nil
}

func (r *MessageRepository) UnreadCount(ctx context.Context, actorID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ConversationMember{}).
		Select("COALESCE(SUM(unread_count), 0)").
		Where("user_id = ? AND deleted_at IS NULL", actorID).Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count unread messages: %w", err)
	}
	return count, nil
}

func (r *MessageRepository) SetMessagePermission(ctx context.Context, actorID uint, permission string) error {
	err := r.db.WithContext(ctx).Exec(`
INSERT INTO user_privacy_settings (user_id, message_permission)
VALUES (?, ?)
ON CONFLICT (user_id) DO UPDATE
SET message_permission = EXCLUDED.message_permission, updated_at = NOW()
`, actorID, permission).Error
	if err != nil {
		return fmt.Errorf("update message permission: %w", err)
	}
	return nil
}

func (r *MessageRepository) BlockUser(ctx context.Context, actorID, blockedID uint) error {
	block := model.UserBlock{BlockerID: actorID, BlockedID: blockedID}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&block).Error; err != nil {
		return fmt.Errorf("block message user: %w", err)
	}
	return nil
}

func (r *MessageRepository) UnblockUser(ctx context.Context, actorID, blockedID uint) error {
	if err := r.db.WithContext(ctx).
		Where("blocker_id = ? AND blocked_id = ?", actorID, blockedID).
		Delete(&model.UserBlock{}).Error; err != nil {
		return fmt.Errorf("unblock message user: %w", err)
	}
	return nil
}
