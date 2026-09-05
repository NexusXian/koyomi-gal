package model

import "time"

type ContributionAction string

const (
	ContributionActionCreate       ContributionAction = "create"
	ContributionActionEdit         ContributionAction = "edit"
	ContributionActionCover        ContributionAction = "cover"
	ContributionActionGallery      ContributionAction = "gallery"
	ContributionActionResource     ContributionAction = "resource"
	ContributionActionAddVolume    ContributionAction = "add_volume"
	ContributionActionUpdateVolume ContributionAction = "update_volume"
	ContributionActionAddRelation  ContributionAction = "add_relation"
)

const (
	ContributionSourceGalgameCreate = "galgame_create"
	ContributionSourceNovelCreate   = "novel_create"
	ContributionSourceGalleryImage  = "gallery_image"
	ContributionSourceNovelVolume   = "novel_volume"
	ContributionSourceResource      = "resource"
	ContributionSourceWorkRelation  = "work_relation"
)

func ValidContributionAction(action ContributionAction) bool {
	switch action {
	case ContributionActionCreate,
		ContributionActionEdit,
		ContributionActionCover,
		ContributionActionGallery,
		ContributionActionResource,
		ContributionActionAddVolume,
		ContributionActionUpdateVolume,
		ContributionActionAddRelation:
		return true
	default:
		return false
	}
}

type WorkContribution struct {
	ID         uint               `gorm:"primaryKey" json:"id"`
	TargetType string             `gorm:"size:32;not null" json:"target_type"`
	TargetID   uint               `gorm:"not null" json:"target_id"`
	UserID     uint               `gorm:"not null" json:"user_id"`
	Action     ContributionAction `gorm:"size:32;not null" json:"action"`
	SourceType *string            `gorm:"size:32" json:"source_type,omitempty"`
	SourceID   *uint              `json:"source_id,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
}

func (WorkContribution) TableName() string { return "work_contributions" }

type WorkContributor struct {
	UserID             uint      `json:"user_id"`
	Username           string    `json:"username"`
	AvatarURL          string    `json:"avatar_url"`
	ContributionCount  int64     `json:"contribution_count"`
	FirstContributedAt time.Time `json:"first_contributed_at"`
	LastContributedAt  time.Time `json:"last_contributed_at"`
}
