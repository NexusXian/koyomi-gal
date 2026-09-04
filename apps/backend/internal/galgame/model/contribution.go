package model

import "time"

type ContributionAction string

const (
	ContributionActionCreate   ContributionAction = "create"
	ContributionActionEdit     ContributionAction = "edit"
	ContributionActionCover    ContributionAction = "cover"
	ContributionActionGallery  ContributionAction = "gallery"
	ContributionActionResource ContributionAction = "resource"
)

const (
	ContributionSourceGalgameCreate = "galgame_create"
	ContributionSourceGalleryImage  = "gallery_image"
	ContributionSourceResource      = "resource"
)

type GalgameContribution struct {
	ID         uint               `gorm:"primaryKey" json:"id"`
	GalgameID  uint               `gorm:"not null" json:"galgame_id"`
	UserID     uint               `gorm:"not null" json:"user_id"`
	Action     ContributionAction `gorm:"size:32;not null" json:"action"`
	SourceType *string            `gorm:"size:32" json:"source_type,omitempty"`
	SourceID   *uint              `json:"source_id,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
}

func (GalgameContribution) TableName() string {
	return "galgame_contributions"
}

type GalgameContributor struct {
	UserID             uint      `json:"user_id"`
	Username           string    `json:"username"`
	AvatarURL          string    `json:"avatar_url"`
	ContributionCount  int64     `json:"contribution_count"`
	FirstContributedAt time.Time `json:"first_contributed_at"`
	LastContributedAt  time.Time `json:"last_contributed_at"`
}
