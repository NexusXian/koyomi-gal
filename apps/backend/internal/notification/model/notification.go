package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type NotificationType string

const (
	TypePostCommented     NotificationType = "post_commented"
	TypeCommentReplied    NotificationType = "comment_replied"
	TypePostLiked         NotificationType = "post_liked"
	TypeCommentLiked      NotificationType = "comment_liked"
	TypeGalgameSubmitted  NotificationType = "galgame_submitted"
	TypeGalgameApproved   NotificationType = "galgame_approved"
	TypeGalgameRejected   NotificationType = "galgame_rejected"
	TypeResourceSubmitted NotificationType = "resource_submitted"
	TypeResourceApproved  NotificationType = "resource_approved"
	TypeResourceRejected  NotificationType = "resource_rejected"
	TypeResourceHidden    NotificationType = "resource_hidden"
	TypeResourceReported  NotificationType = "resource_reported"
	TypeReportResolved    NotificationType = "report_resolved"
	TypeReportRejected    NotificationType = "report_rejected"
	TypePostModerated     NotificationType = "post_moderated"
	TypeCommentModerated  NotificationType = "comment_moderated"
	TypeSystem            NotificationType = "system"
)

type NotificationCategory string

const (
	CategoryInteraction NotificationCategory = "interaction"
	CategoryReview      NotificationCategory = "review"
	CategoryModeration  NotificationCategory = "moderation"
	CategorySystem      NotificationCategory = "system"
)

type Metadata map[string]any

func NewMetadata(value map[string]any) (Metadata, error) {
	if value == nil {
		return Metadata{}, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode notification metadata: %w", err)
	}
	var metadata Metadata
	if err := json.Unmarshal(encoded, &metadata); err != nil {
		return nil, fmt.Errorf("normalize notification metadata: %w", err)
	}
	return metadata, nil
}

func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode notification metadata: %w", err)
	}
	return string(encoded), nil
}

func (m *Metadata) Scan(value any) error {
	if m == nil {
		return errors.New("scan notification metadata into nil receiver")
	}
	if value == nil {
		*m = Metadata{}
		return nil
	}
	var encoded []byte
	switch value := value.(type) {
	case []byte:
		encoded = value
	case string:
		encoded = []byte(value)
	default:
		return fmt.Errorf("scan notification metadata from %T", value)
	}
	if len(encoded) == 0 || string(encoded) == "null" {
		*m = Metadata{}
		return nil
	}
	var metadata Metadata
	if err := json.Unmarshal(encoded, &metadata); err != nil {
		return fmt.Errorf("decode notification metadata: %w", err)
	}
	if metadata == nil {
		metadata = Metadata{}
	}
	*m = metadata
	return nil
}

func IsValidCategory(category NotificationCategory) bool {
	switch category {
	case CategoryInteraction, CategoryReview, CategoryModeration, CategorySystem:
		return true
	default:
		return false
	}
}

func IsValidType(notificationType NotificationType) bool {
	switch notificationType {
	case TypePostCommented,
		TypeCommentReplied,
		TypePostLiked,
		TypeCommentLiked,
		TypeGalgameSubmitted,
		TypeGalgameApproved,
		TypeGalgameRejected,
		TypeResourceSubmitted,
		TypeResourceApproved,
		TypeResourceRejected,
		TypeResourceHidden,
		TypeResourceReported,
		TypeReportResolved,
		TypeReportRejected,
		TypePostModerated,
		TypeCommentModerated,
		TypeSystem:
		return true
	default:
		return false
	}
}

type Notification struct {
	ID          uint                 `gorm:"primaryKey" json:"id"`
	RecipientID uint                 `gorm:"not null;index" json:"-"`
	ActorID     *uint                `json:"-"`
	Category    NotificationCategory `gorm:"type:varchar(32);not null" json:"category"`
	Type        NotificationType     `gorm:"type:varchar(64);not null" json:"type"`
	EntityType  *string              `gorm:"type:varchar(32)" json:"entity_type"`
	EntityID    *uint                `json:"entity_id"`
	Title       string               `gorm:"size:255;not null" json:"title"`
	Content     string               `gorm:"not null" json:"content"`
	TargetURL   *string              `gorm:"type:varchar(512)" json:"target_url"`
	Metadata    Metadata             `gorm:"type:jsonb;not null" json:"metadata"`
	IsRead      bool                 `gorm:"not null;default:false" json:"is_read"`
	ReadAt      *time.Time           `json:"read_at"`
	CreatedAt   time.Time            `json:"created_at"`
	ActorName   string               `gorm:"->" json:"-"`
	ActorAvatar string               `gorm:"->" json:"-"`
}
