package model

import "time"

const (
	LinkTypeNone    = "none"
	LinkTypeURL     = "url"
	LinkTypeGalgame = "galgame"
	LinkTypePost    = "post"
	LinkTypeNews    = "news"
)

type Banner struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Title     string     `gorm:"size:255;not null" json:"title"`
	Subtitle  string     `gorm:"size:500;not null" json:"subtitle"`
	ImageURL  string     `gorm:"not null" json:"image_url"`
	LinkType  string     `gorm:"size:32;not null" json:"link_type"`
	LinkValue string     `gorm:"not null" json:"link_value"`
	SortOrder int        `gorm:"not null" json:"sort_order"`
	IsActive  bool       `gorm:"not null" json:"is_active"`
	StartAt   *time.Time `json:"start_at"`
	EndAt     *time.Time `json:"end_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
