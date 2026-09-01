package model

import "time"

const (
	TypeAnnouncement = "announcement"
	TypeNews         = "news"
	TypeEvent        = "event"
	TypeUpdate       = "update"
)

type Article struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Title       string     `gorm:"size:255;not null" json:"title"`
	Summary     string     `gorm:"not null" json:"summary"`
	Content     string     `gorm:"not null" json:"content"`
	CoverURL    string     `gorm:"not null" json:"cover_url"`
	Type        string     `gorm:"size:32;not null" json:"type"`
	IsPinned    bool       `gorm:"not null" json:"is_pinned"`
	IsPublished bool       `gorm:"not null" json:"is_published"`
	PublishedAt *time.Time `json:"published_at"`
	ViewCount   int64      `gorm:"not null" json:"view_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
