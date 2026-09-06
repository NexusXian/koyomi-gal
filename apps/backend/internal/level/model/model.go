package model

import "time"

// EventType enumerates every experience event handled by the level system.
// Values are stored in experience_rules.event_type / experience_logs.event_type.
type EventType string

const (
	EventDailyCheckin             EventType = "daily_checkin"
	EventCommentCreated           EventType = "comment_created"
	EventLikeGiven                EventType = "like_given"
	EventCommentLiked             EventType = "comment_liked"
	EventShare                    EventType = "share"
	EventGameContributionApproved EventType = "game_contribution_approved"
	EventResourceApproved         EventType = "resource_approved"
	EventCGApproved               EventType = "cg_approved"
	EventCharacterApproved        EventType = "character_approved"
	EventArticleApproved          EventType = "article_approved"
	EventAdminAdjustment          EventType = "admin_adjustment"
)

// ValidEventType reports whether the event type is known to the system.
func ValidEventType(eventType EventType) bool {
	switch eventType {
	case EventDailyCheckin,
		EventCommentCreated,
		EventLikeGiven,
		EventCommentLiked,
		EventShare,
		EventGameContributionApproved,
		EventResourceApproved,
		EventCGApproved,
		EventCharacterApproved,
		EventArticleApproved,
		EventAdminAdjustment:
		return true
	default:
		return false
	}
}

// IsIdempotentEvent reports whether repeated reports of the event must be
// de-duplicated through an idempotency key instead of daily limits only.
func IsIdempotentEvent(eventType EventType) bool {
	switch eventType {
	case EventGameContributionApproved,
		EventResourceApproved,
		EventCGApproved,
		EventCharacterApproved,
		EventArticleApproved,
		EventCommentLiked:
		return true
	default:
		return false
	}
}

// Source types used for idempotency keys and experience_logs.source_type.
const (
	SourceGalgame      = "galgame"
	SourceResource     = "resource"
	SourceGalleryImage = "gallery_image"
	SourceCharacter    = "character"
	SourceArticle      = "article"
	SourceComment      = "comment"
	SourcePost         = "post"
	SourceCheckin      = "checkin"
	SourceAdmin        = "admin"
)

type LevelConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Level       int       `gorm:"not null;uniqueIndex" json:"level"`
	Name        string    `gorm:"size:50;not null" json:"name"`
	MinExp      int64     `gorm:"not null" json:"min_exp"`
	IconURL     string    `gorm:"column:icon_url;size:2048;not null" json:"icon_url"`
	Color       string    `gorm:"size:32;not null" json:"color"`
	Description string    `gorm:"size:255;not null" json:"description"`
	IsEnabled   bool      `gorm:"column:is_enabled;not null" json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (LevelConfig) TableName() string { return "level_configs" }

type UserExperience struct {
	UserID       uint      `gorm:"primaryKey" json:"user_id"`
	TotalExp     int64     `gorm:"not null" json:"total_exp"`
	CurrentLevel int       `gorm:"not null" json:"current_level"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (UserExperience) TableName() string { return "user_experience" }

type ExperienceRule struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	EventType       EventType `gorm:"size:64;not null;uniqueIndex" json:"event_type"`
	Name            string    `gorm:"size:100;not null" json:"name"`
	Exp             int       `gorm:"not null" json:"exp"`
	DailyLimit      int       `gorm:"not null" json:"daily_limit"`
	DailyExpLimit   int       `gorm:"not null" json:"daily_exp_limit"`
	CooldownSeconds int       `gorm:"not null" json:"cooldown_seconds"`
	Enabled         bool      `gorm:"not null" json:"enabled"`
	Description     string    `gorm:"size:255;not null" json:"description"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ExperienceRule) TableName() string { return "experience_rules" }

type ExperienceLog struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	EventType      EventType `gorm:"size:64;not null" json:"event_type"`
	ExpDelta       int64     `gorm:"not null" json:"exp_delta"`
	SourceType     string    `gorm:"size:32;not null" json:"source_type"`
	SourceID       *uint     `json:"source_id"`
	IdempotencyKey *string   `gorm:"size:128" json:"idempotency_key"`
	Description    string    `gorm:"size:255;not null" json:"description"`
	OperatorID     *uint     `json:"operator_id"`
	CreatedAt      time.Time `json:"created_at"`
}

func (ExperienceLog) TableName() string { return "experience_logs" }

type UserCheckin struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"not null" json:"user_id"`
	CheckinDate     time.Time `gorm:"type:date;not null" json:"checkin_date"`
	ConsecutiveDays int       `gorm:"not null" json:"consecutive_days"`
	CreatedAt       time.Time `json:"created_at"`
}

func (UserCheckin) TableName() string { return "user_checkins" }
