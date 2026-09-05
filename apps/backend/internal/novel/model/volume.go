package model

import (
	"time"

	"gorm.io/gorm"
)

type NovelVolume struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	NovelID       uint           `gorm:"not null;index" json:"novel_id"`
	VolumeNumber  *int           `json:"volume_number"`
	Title         string         `gorm:"size:255;not null;default:''" json:"title"`
	OriginalTitle string         `gorm:"size:255;not null;default:''" json:"original_title"`
	CoverURL      string         `gorm:"not null" json:"cover_url"`
	ISBN          string         `gorm:"column:isbn;size:20;not null;default:''" json:"isbn"`
	ReleaseDate   *time.Time     `gorm:"type:date" json:"release_date"`
	Summary       string         `gorm:"not null" json:"summary"`
	SortOrder     int            `gorm:"not null;default:0" json:"sort_order"`
	CreatedBy     *uint          `json:"created_by"`
	Status        int16          `gorm:"not null" json:"status"`
	ReviewedBy    *uint          `json:"reviewed_by"`
	ReviewedAt    *time.Time     `json:"reviewed_at"`
	RejectReason  string         `gorm:"not null;default:''" json:"reject_reason"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Transient novel title, populated only by the admin listing query.
	NovelTitle string `gorm:"->;-:migration" json:"-"`
}

func (NovelVolume) TableName() string { return "novel_volumes" }
