package dto

import (
	"time"

	"backend/internal/galgame/model"
)

type ContributorQuery struct {
	Page     int `form:"page" binding:"omitempty,min=1,max=1000000"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type ContributorData struct {
	UserID             uint      `json:"user_id" example:"1001"`
	Username           string    `json:"username" example:"NexusXian"`
	AvatarURL          string    `json:"avatar_url" example:"https://example.com/avatar.jpg"`
	ContributionCount  int64     `json:"contribution_count" example:"12"`
	FirstContributedAt time.Time `json:"first_contributed_at"`
	LastContributedAt  time.Time `json:"last_contributed_at"`
}

type ContributorListData struct {
	Items    []ContributorData `json:"items"`
	Total    int64             `json:"total" example:"12"`
	Page     int               `json:"page" example:"1"`
	PageSize int               `json:"page_size" example:"20"`
}

type ContributorListResponse struct {
	Code int                 `json:"code" example:"0"`
	Data ContributorListData `json:"data"`
	Msg  string              `json:"msg" example:"success"`
}

func NewContributorData(contributors []model.GalgameContributor) []ContributorData {
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
