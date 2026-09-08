package model

import "time"

const (
	TypeNormal      = "normal"
	TypeUpdate      = "update"
	TypeMaintenance = "maintenance"
	TypeWarning     = "warning"
	TypeEvent       = "event"
	TypeSystem      = "system"

	DisplayModeNormal       = "normal"
	DisplayModeBanner       = "banner"
	DisplayModeModal        = "modal"
	DisplayModeStartupModal = "startup_modal"

	TargetAll     = "all"
	TargetWeb     = "web"
	TargetAndroid = "android"
	TargetIOS     = "ios"
	TargetDesktop = "desktop"
)

type Announcement struct {
	ID          uint   `gorm:"primaryKey"`
	Title       string `gorm:"size:255;not null"`
	Content     string `gorm:"not null"`
	Type        string `gorm:"size:32;not null"`
	DisplayMode string `gorm:"size:32;not null"`
	Target      string `gorm:"size:32;not null"`
	Priority    int    `gorm:"not null"`
	StartsAt    *time.Time
	EndsAt      *time.Time
	Dismissible bool `gorm:"not null"`
	Published   bool `gorm:"not null"`
	CreatedBy   *uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
