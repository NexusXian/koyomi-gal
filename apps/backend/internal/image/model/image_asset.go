package model

import "time"

const (
	ImageStatusPending = 0
	ImageStatusActive  = 1
	ImageStatusDeleted = 2
	ImageStatusFailed  = 3
)

const (
	CategoryAvatar     = "avatars"
	CategoryPost       = "posts"
	CategoryComment    = "comments"
	CategoryGalgame    = "galgames"
	CategoryBackground = "backgrounds"
	CategoryBanner     = "banners"
	CategoryAdmin      = "admin"
)

type ImageAsset struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       *uint      `gorm:"index" json:"user_id"`
	ObjectKey    string     `gorm:"size:512;uniqueIndex;not null" json:"object_key"`
	OriginalName string     `gorm:"size:255;not null" json:"original_name"`
	MimeType     string     `gorm:"size:100;not null" json:"mime_type"`
	Extension    string     `gorm:"size:20;not null" json:"extension"`
	Size         int64      `gorm:"not null" json:"size"`
	Width        *int       `json:"width"`
	Height       *int       `json:"height"`
	Category     string     `gorm:"size:50;not null" json:"category"`
	Status       int16      `gorm:"not null" json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
}
