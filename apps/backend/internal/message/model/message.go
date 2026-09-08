package model

import "time"

const (
	ConversationTypeDirect = "direct"
	MessageTypeText        = "text"

	MessagePermissionEveryone = "everyone"
	MessagePermissionNone     = "none"
)

type Conversation struct {
	ID               uint       `gorm:"primaryKey"`
	ConversationType string     `gorm:"column:conversation_type;size:20;not null"`
	LastMessageID    *uint      `gorm:"column:last_message_id"`
	LastMessageAt    *time.Time `gorm:"column:last_message_at"`
	CreatedAt        time.Time  `gorm:"not null"`
	UpdatedAt        time.Time  `gorm:"not null"`
}

type ConversationMember struct {
	ConversationID    uint      `gorm:"primaryKey"`
	UserID            uint      `gorm:"primaryKey"`
	LastReadMessageID *uint     `gorm:"column:last_read_message_id"`
	UnreadCount       int64     `gorm:"not null"`
	JoinedAt          time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
	DeletedAt         *time.Time
}

type DirectConversation struct {
	ConversationID uint      `gorm:"primaryKey"`
	UserLowID      uint      `gorm:"column:user_low_id;not null"`
	UserHighID     uint      `gorm:"column:user_high_id;not null"`
	CreatedAt      time.Time `gorm:"not null"`
}

type Message struct {
	ID               uint      `gorm:"primaryKey"`
	ConversationID   uint      `gorm:"not null"`
	SenderID         uint      `gorm:"not null"`
	ReceiverID       uint      `gorm:"not null"`
	ReplyToMessageID *uint     `gorm:"column:reply_to_message_id"`
	MessageType      string    `gorm:"column:message_type;size:20;not null"`
	Content          string    `gorm:"not null"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
	DeletedAt        *time.Time

	SenderUsername    string `gorm:"->;column:sender_username"`
	SenderDisplayName string `gorm:"->;column:sender_display_name"`
	SenderAvatarURL   string `gorm:"->;column:sender_avatar_url"`
}

type UserBlock struct {
	BlockerID uint      `gorm:"primaryKey"`
	BlockedID uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"not null"`
}

type ConversationRecord struct {
	ID                   uint
	ConversationType     string
	UnreadCount          int64
	LastMessageAt        *time.Time
	SortAt               time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
	PeerID               uint
	PeerUsername         string
	PeerDisplayName      string
	PeerAvatarURL        string
	IsBlocked            bool
	CanSend              bool
	LastMessageID        *uint
	LastSenderID         *uint
	LastMessageType      *string
	LastMessageContent   *string
	LastMessageDeletedAt *time.Time
	LastMessageCreatedAt *time.Time
}
