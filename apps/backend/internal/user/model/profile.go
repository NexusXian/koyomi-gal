package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	ProfileVisibilityPublic     = "public"
	ProfileVisibilityRegistered = "registered"
	ProfileVisibilityPrivate    = "private"
)

const (
	ActivityPostCreated       = "post_created"
	ActivityCommentCreated    = "comment_created"
	ActivityRatingCreated     = "rating_created"
	ActivityFavoriteCreated   = "favorite_created"
	ActivityResourceSubmitted = "resource_submitted"
	ActivityReviewApproved    = "review_approved"
)

type UserProfile struct {
	UserID        uint   `gorm:"primaryKey"`
	DisplayName   string `gorm:"size:100;not null"`
	AvatarAssetID *uint
	BannerAssetID *uint
	Bio           string     `gorm:"size:1000;not null"`
	Gender        string     `gorm:"size:20;not null"`
	Location      string     `gorm:"size:100;not null"`
	Birthday      *time.Time `gorm:"type:date"`
	WebsiteURL    string     `gorm:"size:2048;not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type UserPrivacySettings struct {
	UserID            uint   `gorm:"primaryKey"`
	ProfileVisibility string `gorm:"size:20;not null"`
	ShowLocation      bool
	ShowBirthday      bool
	ShowPosts         bool
	ShowComments      bool
	ShowRatings       bool
	ShowFavorites     bool
	ShowActivity      bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func DefaultUserPrivacySettings(userID uint) UserPrivacySettings {
	return UserPrivacySettings{
		UserID: userID, ProfileVisibility: ProfileVisibilityPublic,
		ShowPosts: true, ShowComments: true, ShowRatings: true, ShowActivity: true,
	}
}

type UserActivity struct {
	ID           uint   `gorm:"primaryKey"`
	UserID       uint   `gorm:"not null"`
	ActivityType string `gorm:"size:50;not null"`
	GalgameID    *uint
	PostID       *uint
	CommentID    *uint
	ResourceID   *uint
	Metadata     ActivityMetadata `gorm:"type:jsonb;not null"`
	CreatedAt    time.Time
}

type ActivityMetadata map[string]any

func (m ActivityMetadata) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode activity metadata: %w", err)
	}
	return string(encoded), nil
}

func (m *ActivityMetadata) Scan(value any) error {
	if m == nil {
		return errors.New("scan activity metadata into nil receiver")
	}
	if value == nil {
		*m = ActivityMetadata{}
		return nil
	}
	var encoded []byte
	switch value := value.(type) {
	case []byte:
		encoded = value
	case string:
		encoded = []byte(value)
	default:
		return fmt.Errorf("scan activity metadata from %T", value)
	}
	if err := json.Unmarshal(encoded, m); err != nil {
		return fmt.Errorf("decode activity metadata: %w", err)
	}
	if *m == nil {
		*m = ActivityMetadata{}
	}
	return nil
}

type PublicProfileRecord struct {
	ID                uint
	Username          string
	DisplayName       string
	AvatarAssetID     *uint
	BannerAssetID     *uint
	AvatarURL         string
	BannerURL         string
	Bio               string
	Gender            string
	Location          string
	Birthday          *time.Time
	WebsiteURL        string
	RegisteredAt      time.Time
	ProfileVisibility string
	ShowLocation      bool
	ShowBirthday      bool
	ShowPosts         bool
	ShowComments      bool
	ShowRatings       bool
	ShowFavorites     bool
	ShowActivity      bool
}

type ProfileCounts struct {
	Posts     int64
	Comments  int64
	Ratings   int64
	Favorites int64
}

type ProfilePost struct {
	ID            uint
	GalgameID     *uint
	GalgameTitle  string
	Title         string
	Content       string
	EditorMode    string
	LikeCount     int64
	CommentCount  int64
	FavoriteCount int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ProfileComment struct {
	ID        uint
	PostID    uint
	PostTitle string
	ParentID  *uint
	Content   string
	LikeCount int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ProfileGalgameItem struct {
	ID             uint
	Title          string
	Slug           string
	CoverURL       string
	CoverSensitive bool
	Score          *int16
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
