package realtime

import (
	"context"
	"time"
)

const (
	EventMessageCreated   = "message.created"
	EventConversationRead = "conversation.read"
	EventMessageDeleted   = "message.deleted"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type MessageCreatedData struct {
	ConversationID uint                  `json:"conversation_id"`
	Message        MessageCreatedMessage `json:"message"`
}

type MessageCreatedMessage struct {
	ID             uint      `json:"id"`
	ConversationID uint      `json:"conversation_id"`
	SenderID       uint      `json:"sender_id"`
	Type           string    `json:"type"`
	Content        string    `json:"content"`
	IsDeleted      bool      `json:"is_deleted"`
	CreatedAt      time.Time `json:"created_at"`
}

type ConversationReadData struct {
	ConversationID uint `json:"conversation_id"`
	UserID         uint `json:"user_id"`
	MessageID      uint `json:"message_id"`
}

type MessageDeletedData struct {
	ConversationID uint `json:"conversation_id"`
	MessageID      uint `json:"message_id"`
}

type Publisher interface {
	Publish(ctx context.Context, userIDs []uint, event Event) error
}
