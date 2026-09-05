package dto

import (
	"time"

	contributionModel "backend/internal/contribution/model"
	galgameModel "backend/internal/galgame/model"
	relationModel "backend/internal/relation/model"
)

type TagSummary struct {
	ID   uint   `json:"id" example:"1"`
	Name string `json:"name" example:"恋爱"`
}

type ContributorData struct {
	UserID             uint      `json:"user_id" example:"1001"`
	Username           string    `json:"username" example:"NexusXian"`
	AvatarURL          string    `json:"avatar_url" example:"https://example.com/avatar.jpg"`
	ContributionCount  int64     `json:"contribution_count" example:"12"`
	FirstContributedAt time.Time `json:"first_contributed_at"`
	LastContributedAt  time.Time `json:"last_contributed_at"`
}

type RelatedWorkData struct {
	RelationID     uint   `json:"relation_id" example:"1"`
	RelationType   string `json:"relation_type" example:"adaptation"`
	WorkID         uint   `json:"work_id" example:"3"`
	Title          string `json:"title" example:"千恋＊万花"`
	OriginalTitle  string `json:"original_title" example:"千恋＊万花"`
	Slug           string `json:"slug" example:"senren-banka"`
	CoverURL       string `json:"cover_url" example:"https://example.com/cover.jpg"`
	CoverSensitive bool   `json:"cover_sensitive" example:"false"`
	AgeRating      int16  `json:"age_rating" example:"3"`
}

func newTagSummaries(tags []galgameModel.Tag) []TagSummary {
	responses := make([]TagSummary, 0, len(tags))
	for _, tag := range tags {
		responses = append(responses, TagSummary{ID: tag.ID, Name: tag.Name})
	}
	return responses
}

func NewContributorData(contributors []contributionModel.WorkContributor) []ContributorData {
	items := make([]ContributorData, 0, len(contributors))
	for _, contributor := range contributors {
		items = append(items, ContributorData{
			UserID:             contributor.UserID,
			Username:           contributor.Username,
			AvatarURL:          contributor.AvatarURL,
			ContributionCount:  contributor.ContributionCount,
			FirstContributedAt: contributor.FirstContributedAt,
			LastContributedAt:  contributor.LastContributedAt,
		})
	}
	return items
}

func NewRelatedWorkData(works []relationModel.RelatedWork) []RelatedWorkData {
	items := make([]RelatedWorkData, 0, len(works))
	for _, work := range works {
		items = append(items, RelatedWorkData{
			RelationID:     work.RelationID,
			RelationType:   work.RelationType,
			WorkID:         work.WorkID,
			Title:          work.Title,
			OriginalTitle:  work.OriginalTitle,
			Slug:           work.Slug,
			CoverURL:       work.CoverURL,
			CoverSensitive: work.CoverSensitive,
			AgeRating:      work.AgeRating,
		})
	}
	return items
}
