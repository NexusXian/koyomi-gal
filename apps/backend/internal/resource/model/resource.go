package model

import "time"

const (
	ResourceTypeOther int16 = iota
	ResourceTypeGame
	ResourceTypePatch
	ResourceTypeSave
	ResourceTypeSoundtrack
	ResourceTypeCG
	ResourceTypeGuide
)

const (
	ResourceStatusPending int16 = iota
	ResourceStatusPublished
	ResourceStatusRejected
	ResourceStatusHidden
)

type Resource struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	GalgameID   uint           `gorm:"not null" json:"galgame_id"`
	UploaderID  *uint          `json:"uploader_id"`
	Title       string         `gorm:"size:255;not null" json:"title"`
	Type        int16          `gorm:"column:resource_type;not null" json:"type"`
	Description string         `gorm:"not null" json:"description"`
	Status      int16          `gorm:"not null" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Links       []ResourceLink `gorm:"foreignKey:ResourceID" json:"links,omitempty"`
}

type ResourceLink struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ResourceID uint      `gorm:"not null" json:"resource_id"`
	URL        string    `gorm:"not null" json:"url"`
	CreatedAt  time.Time `json:"created_at"`
}
