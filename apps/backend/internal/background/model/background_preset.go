package model

import "time"

type BackgroundPreset struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"size:64;uniqueIndex;not null" json:"key"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	ImageURL  string    `gorm:"not null" json:"image_url"`
	SortOrder int       `gorm:"not null" json:"sort_order"`
	IsActive  bool      `gorm:"not null" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
