package dto

import (
	"time"

	"backend/internal/message/model"
	userdto "backend/internal/user/dto"
)

type CreateConversationRequest struct {
	UserID uint `json:"user_id" binding:"required,min=1" example:"1002"`
}

type ConversationListQuery struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

type MessageListQuery struct {
	BeforeID uint `form:"before_id" binding:"omitempty,min=1"`
	Limit    int  `form:"limit" binding:"omitempty,min=1,max=100"`
}

type SendMessageRequest struct {
	Type    string `json:"type" example:"text"`
	Content string `json:"content" example:"Hello"`
}

type MarkReadRequest struct {
	MessageID uint `json:"message_id" binding:"required,min=1" example:"42"`
}

type UpdateMessageSettingsRequest struct {
	Permission string `json:"permission" example:"everyone"`
}

type MessageSettingsData struct {
	Permission string `json:"permission" example:"everyone"`
}

type MessageData struct {
	ID             uint                      `json:"id" example:"42"`
	ConversationID uint                      `json:"conversation_id" example:"8"`
	SenderID       uint                      `json:"sender_id" example:"1001"`
	Sender         userdto.PublicUserSummary `json:"sender"`
	ReceiverID     uint                      `json:"receiver_id" example:"1002"`
	Type           string                    `json:"type" example:"text"`
	Content        string                    `json:"content" example:"Hello"`
	IsDeleted      bool                      `json:"is_deleted"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
}

type MessagePreview struct {
	ID        uint      `json:"id"`
	SenderID  uint      `json:"sender_id"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	IsDeleted bool      `json:"is_deleted"`
	CreatedAt time.Time `json:"created_at"`
}

type ConversationData struct {
	ID            uint                      `json:"id" example:"8"`
	Type          string                    `json:"type" example:"direct"`
	User          userdto.PublicUserSummary `json:"user"`
	LastMessage   *MessagePreview           `json:"last_message"`
	LastMessageAt *time.Time                `json:"last_message_at"`
	UnreadCount   int64                     `json:"unread_count"`
	IsBlocked     bool                      `json:"is_blocked"`
	CanSend       bool                      `json:"can_send"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
}

type ConversationListData struct {
	List       []ConversationData `json:"list"`
	NextCursor string             `json:"next_cursor"`
	HasMore    bool               `json:"has_more"`
}

type MessageListData struct {
	List       []MessageData `json:"list"`
	NextCursor uint          `json:"next_cursor"`
	HasMore    bool          `json:"has_more"`
}

type UnreadCountData struct {
	Count int64 `json:"count" example:"3"`
}

type ConversationResponse struct {
	Code int              `json:"code" example:"0"`
	Data ConversationData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

type ConversationListResponse struct {
	Code int                  `json:"code" example:"0"`
	Data ConversationListData `json:"data"`
	Msg  string               `json:"msg" example:"success"`
}

type MessageResponse struct {
	Code int         `json:"code" example:"0"`
	Data MessageData `json:"data"`
	Msg  string      `json:"msg" example:"success"`
}

type MessageListResponse struct {
	Code int             `json:"code" example:"0"`
	Data MessageListData `json:"data"`
	Msg  string          `json:"msg" example:"success"`
}

type MessageUnreadCountResponse struct {
	Code int             `json:"code" example:"0"`
	Data UnreadCountData `json:"data"`
	Msg  string          `json:"msg" example:"success"`
}

type MessageSettingsResponse struct {
	Code int                 `json:"code" example:"0"`
	Data MessageSettingsData `json:"data"`
	Msg  string              `json:"msg" example:"success"`
}

func NewMessageData(message *model.Message) MessageData {
	content := message.Content
	if message.DeletedAt != nil {
		content = ""
	}
	return MessageData{
		ID: message.ID, ConversationID: message.ConversationID, SenderID: message.SenderID,
		Sender: userdto.PublicUserSummary{
			ID: message.SenderID, Username: message.SenderUsername,
			DisplayName: message.SenderDisplayName, AvatarURL: message.SenderAvatarURL,
		},
		ReceiverID: message.ReceiverID, Type: message.MessageType, Content: content,
		IsDeleted: message.DeletedAt != nil, CreatedAt: message.CreatedAt, UpdatedAt: message.UpdatedAt,
	}
}

func NewConversationData(record *model.ConversationRecord) ConversationData {
	data := ConversationData{
		ID: record.ID, Type: record.ConversationType,
		User: userdto.PublicUserSummary{
			ID: record.PeerID, Username: record.PeerUsername,
			DisplayName: record.PeerDisplayName, AvatarURL: record.PeerAvatarURL,
		},
		LastMessageAt: record.LastMessageAt, UnreadCount: record.UnreadCount,
		IsBlocked: record.IsBlocked, CanSend: record.CanSend,
		CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
	if record.LastMessageID != nil {
		content := ""
		if record.LastMessageContent != nil && record.LastMessageDeletedAt == nil {
			content = *record.LastMessageContent
		}
		data.LastMessage = &MessagePreview{
			ID: *record.LastMessageID, Content: content,
			IsDeleted: record.LastMessageDeletedAt != nil,
			CreatedAt: *record.LastMessageCreatedAt,
		}
		if record.LastSenderID != nil {
			data.LastMessage.SenderID = *record.LastSenderID
		}
		if record.LastMessageType != nil {
			data.LastMessage.Type = *record.LastMessageType
		}
	}
	return data
}
