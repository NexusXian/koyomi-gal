package model

import (
	"time"

	contributionModel "backend/internal/contribution/model"
	galgameModel "backend/internal/galgame/model"
	relationModel "backend/internal/relation/model"

	"gorm.io/gorm"
)

const (
	NovelStatusPending int16 = iota
	NovelStatusPublished
	NovelStatusRejected
	NovelStatusHidden
)

const (
	ReleaseStatusOngoing   = "ongoing"
	ReleaseStatusCompleted = "completed"
	ReleaseStatusHiatus    = "hiatus"
	ReleaseStatusCancelled = "cancelled"
	ReleaseStatusUnknown   = "unknown"
)

func ValidReleaseStatus(value string) bool {
	switch value {
	case ReleaseStatusOngoing,
		ReleaseStatusCompleted,
		ReleaseStatusHiatus,
		ReleaseStatusCancelled,
		ReleaseStatusUnknown:
		return true
	default:
		return false
	}
}

type Novel struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Title            string         `gorm:"size:255;not null" json:"title"`
	OriginalTitle    string         `gorm:"size:255;not null;default:''" json:"original_title"`
	Slug             string         `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Summary          string         `gorm:"not null" json:"summary"`
	CoverURL         string         `gorm:"not null" json:"cover_url"`
	Author           string         `gorm:"size:255;not null;default:''" json:"author"`
	Illustrator      string         `gorm:"size:255;not null;default:''" json:"illustrator"`
	Publisher        string         `gorm:"size:255;not null;default:''" json:"publisher"`
	Label            string         `gorm:"size:255;not null;default:''" json:"label"`
	Language         string         `gorm:"size:16;not null;default:''" json:"language"`
	Region           string         `gorm:"size:16;not null;default:''" json:"region"`
	ReleaseStatus    string         `gorm:"size:16;not null;default:'unknown'" json:"release_status"`
	FirstReleaseDate *time.Time     `gorm:"type:date" json:"first_release_date"`
	AgeRating        int16          `gorm:"not null" json:"age_rating"`
	IsCoverSensitive bool           `gorm:"not null;default:false;index" json:"is_cover_sensitive"`
	OfficialWebsite  string         `gorm:"not null" json:"official_website"`
	ResourceCount    int64          `gorm:"not null" json:"resource_count"`
	CreatedBy        *uint          `json:"created_by"`
	Status           int16          `gorm:"not null" json:"status"`
	ReviewedBy       *uint          `json:"reviewed_by"`
	ReviewedAt       *time.Time     `json:"reviewed_at"`
	RejectReason     string         `gorm:"not null;default:''" json:"reject_reason"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

	Tags        []galgameModel.Tag `gorm:"many2many:novel_tags" json:"tags,omitempty"`
	Volumes     []NovelVolume      `gorm:"-" json:"-"`
	VolumeCount int64              `gorm:"->;-:migration" json:"-"`

	Contributors     []contributionModel.WorkContributor `gorm:"-" json:"-"`
	ContributorCount int64                               `gorm:"-" json:"-"`
	RelatedGalgames  []relationModel.RelatedWork         `gorm:"-" json:"-"`
}

type NovelTag struct {
	NovelID   uint      `gorm:"not null" json:"novel_id"`
	TagID     uint      `gorm:"not null" json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (NovelTag) TableName() string { return "novel_tags" }
