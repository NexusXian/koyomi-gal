package model

import "time"

const (
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
	PlatformWindows = "windows"
	PlatformMacOS   = "macos"
	PlatformLinux   = "linux"

	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusDisabled  = "disabled"
)

type AppRelease struct {
	ID                  uint   `gorm:"primaryKey"`
	Platform            string `gorm:"size:32;not null"`
	VersionName         string `gorm:"size:64;not null"`
	VersionCode         int64  `gorm:"not null"`
	Title               string `gorm:"size:255;not null"`
	Changelog           string `gorm:"not null"`
	DownloadURL         string `gorm:"size:2048;not null"`
	FileSize            *int64
	FileSHA256          *string `gorm:"size:64"`
	MinimumVersionCode  int64   `gorm:"not null"`
	ForceUpdate         bool    `gorm:"not null"`
	Status              string  `gorm:"size:32;not null"`
	PublishedAt         *time.Time
	AnnouncementID      *uint
	AnnouncementManaged bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
