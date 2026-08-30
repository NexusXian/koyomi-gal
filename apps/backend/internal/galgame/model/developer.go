package model

import "time"

type Developer struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:255;not null" json:"name"`
	OriginalName string    `gorm:"size:255;not null" json:"original_name"`
	Slug         string    `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Description  string    `gorm:"not null" json:"description"`
	LogoURL      string    `gorm:"not null" json:"logo_url"`
	Website      string    `gorm:"not null" json:"website"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
