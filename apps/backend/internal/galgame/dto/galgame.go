package dto

import (
	"time"

	"backend/internal/galgame/model"
)

type CreateGalgameRequest struct {
	Title          string   `json:"title" binding:"required,max=255" example:"千恋＊万花"`
	OriginalTitle  string   `json:"original_title" binding:"max=255" example:"千恋＊万花"`
	RomajiTitle    string   `json:"romaji_title" binding:"max=255" example:"Senren Banka"`
	Slug           string   `json:"slug" binding:"required,max=255" example:"senren-banka"`
	Description    string   `json:"description" example:"作品简介"`
	CoverURL       string   `json:"cover_url" example:"https://example.com/cover.jpg"`
	BannerURL      string   `json:"banner_url" example:"https://example.com/banner.jpg"`
	DeveloperID    *uint    `json:"developer_id" binding:"omitempty,gt=0" example:"1"`
	ReleaseDate    string   `json:"release_date" example:"2016-07-29"`
	AgeRating      int16    `json:"age_rating" binding:"oneof=0 1 2 3 4 5" example:"3"`
	CoverSensitive bool     `json:"cover_sensitive" example:"false"`
	Status         int16    `json:"status" binding:"oneof=0 1 2 3" example:"1"`
	Aliases        []string `json:"aliases" binding:"max=100,dive,max=255" example:"千恋万花,Senren Banka"`
	TagIDs         []uint   `json:"tag_ids" binding:"max=100,dive,gt=0" example:"1,2"`
}

type UpdateGalgameRequest struct {
	Title          string   `json:"title" binding:"required,max=255" example:"千恋＊万花"`
	OriginalTitle  string   `json:"original_title" binding:"max=255" example:"千恋＊万花"`
	RomajiTitle    string   `json:"romaji_title" binding:"max=255" example:"Senren Banka"`
	Slug           string   `json:"slug" binding:"required,max=255" example:"senren-banka"`
	Description    string   `json:"description" example:"作品简介"`
	CoverURL       string   `json:"cover_url" example:"https://example.com/cover.jpg"`
	BannerURL      string   `json:"banner_url" example:"https://example.com/banner.jpg"`
	DeveloperID    *uint    `json:"developer_id" binding:"omitempty,gt=0" example:"1"`
	ReleaseDate    string   `json:"release_date" example:"2016-07-29"`
	AgeRating      *int16   `json:"age_rating" binding:"required,oneof=0 1 2 3 4 5" example:"3"`
	CoverSensitive *bool    `json:"cover_sensitive" binding:"required" example:"false"`
	Status         *int16   `json:"status" binding:"required,oneof=0 1 2 3" example:"1"`
	Aliases        []string `json:"aliases" binding:"max=100,dive,max=255" example:"千恋万花,Senren Banka"`
	TagIDs         []uint   `json:"tag_ids" binding:"max=100,dive,gt=0" example:"1,2"`
}

// BatchUpdateGalgameRequest only allows the whitelisted fields below.
type BatchUpdateGalgameRequest struct {
	IDs            []uint `json:"ids" binding:"required,min=1,max=500,dive,gt=0" example:"1,2,3"`
	AgeRating      *int16 `json:"age_rating" binding:"omitempty,oneof=0 1 2 3 4 5" example:"3"`
	CoverSensitive *bool  `json:"cover_sensitive" example:"true"`
}

// BatchDeleteGalgameRequest hard-deletes the matched galgames.
type BatchDeleteGalgameRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1,max=500,dive,gt=0" example:"1,2,3"`
}

type BatchDeleteGalgameData struct {
	Deleted int64 `json:"deleted" example:"5"`
}

type BatchDeleteGalgameResponse struct {
	Code int                    `json:"code" example:"0"`
	Data BatchDeleteGalgameData `json:"data"`
	Msg  string                 `json:"msg" example:"success"`
}

type BatchUpdateGalgameData struct {
	Updated int64 `json:"updated" example:"5"`
}

type BatchUpdateGalgameResponse struct {
	Code int                    `json:"code" example:"0"`
	Data BatchUpdateGalgameData `json:"data"`
	Msg  string                 `json:"msg" example:"success"`
}

type ReviewGalgameRequest struct {
	Status int16 `json:"status" binding:"required,oneof=1 2" example:"1"`
}

type GalgameQuery struct {
	Keyword     string `form:"keyword" binding:"max=255"`
	DeveloperID *uint  `form:"developer_id" binding:"omitempty,gt=0"`
	TagIDs      []uint `form:"-"`
	ReleaseFrom *int   `form:"release_from" binding:"omitempty,min=1,max=9999"`
	ReleaseTo   *int   `form:"release_to" binding:"omitempty,min=1,max=9999"`
	AgeRating   *int16 `form:"age_rating" binding:"omitempty,oneof=0 1 2 3 4 5"`
	Sort        string `form:"sort"`
	Page        int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit       int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// AdminGalgameQuery filters the galgame:review admin listing, which returns
// every status instead of only published entries. The ai_* filters apply to
// each game's latest AI classification proposal.
type AdminGalgameQuery struct {
	Status         *int16 `form:"status" binding:"omitempty,oneof=0 1 2 3" example:"0"`
	AgeRating      *int16 `form:"age_rating" binding:"omitempty,oneof=0 1 2 3 4 5" example:"3"`
	CoverSensitive *bool  `form:"cover_sensitive" example:"true"`
	Keyword        string `form:"keyword" binding:"max=255"`
	Sort           string `form:"sort"`
	Page           int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit          int    `form:"limit" binding:"omitempty,min=1,max=100"`

	AIClassification string   `form:"ai_classification" binding:"omitempty,oneof=r18 non_r18 unknown" example:"r18"`
	AIStatus         string   `form:"ai_status" binding:"omitempty,oneof=queued processing pending_review approved rejected failed" example:"failed"`
	AIConflict       *bool    `form:"ai_conflict" example:"true"`
	AIMinConfidence  *float64 `form:"ai_min_confidence" binding:"omitempty,gte=0,lte=1" example:"0.95"`
	AIMaxConfidence  *float64 `form:"ai_max_confidence" binding:"omitempty,gte=0,lte=1" example:"0.7"`
}

type MyGalgameQuery struct {
	Type  string `form:"type" binding:"omitempty,oneof=uploaded favorite"`
	Page  int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

type DeveloperSummary struct {
	ID   uint   `json:"id" example:"1"`
	Name string `json:"name" example:"YUZUSOFT"`
}

type TagSummary struct {
	ID   uint   `json:"id" example:"1"`
	Name string `json:"name" example:"纯爱"`
}

type RatingSummary struct {
	Average float64 `json:"average" example:"8.72"`
	Count   int64   `json:"count" example:"120"`
}

type GalgameStatistics struct {
	FavoriteCount int64 `json:"favorite_count" example:"300"`
	ResourceCount int64 `json:"resource_count" example:"5"`
	PostCount     int64 `json:"post_count" example:"20"`
}

type GalgameListItem struct {
	ID             uint              `json:"id" example:"1"`
	Title          string            `json:"title" example:"千恋＊万花"`
	OriginalTitle  string            `json:"original_title" example:"千恋＊万花"`
	RomajiTitle    string            `json:"romaji_title" example:"Senren Banka"`
	Slug           string            `json:"slug" example:"senren-banka"`
	CoverURL       string            `json:"cover_url" example:"https://example.com/cover.jpg"`
	ReleaseDate    *string           `json:"release_date" example:"2016-07-29"`
	AgeRating      int16             `json:"age_rating" example:"3"`
	CoverSensitive bool              `json:"cover_sensitive" example:"false"`
	Status         int16             `json:"status" example:"1"`
	Developer      *DeveloperSummary `json:"developer"`
	Tags           []TagSummary      `json:"tags"`
	Rating         RatingSummary     `json:"rating"`
	Statistics     GalgameStatistics `json:"statistics"`

	// Latest AI age-rating proposal (admin listing only; absent when the game
	// was never classified).
	AIClassification string  `json:"ai_classification,omitempty" example:"r18"`
	AIConfidence     float64 `json:"ai_confidence,omitempty" example:"0.98"`
	AIStatus         string  `json:"ai_status,omitempty" example:"pending_review"`
	AIConflict       bool    `json:"ai_conflict,omitempty" example:"false"`
}

type GalgameResponse struct {
	ID                uint              `json:"id" example:"1"`
	Title             string            `json:"title" example:"千恋＊万花"`
	OriginalTitle     string            `json:"original_title" example:"千恋＊万花"`
	RomajiTitle       string            `json:"romaji_title" example:"Senren Banka"`
	Slug              string            `json:"slug" example:"senren-banka"`
	Description       string            `json:"description" example:"作品简介"`
	DescriptionSource string            `json:"description_source" example:"bangumi"`
	CoverURL          string            `json:"cover_url" example:"https://example.com/cover.jpg"`
	BannerURL         string            `json:"banner_url" example:"https://example.com/banner.jpg"`
	ReleaseDate       *string           `json:"release_date" example:"2016-07-29"`
	AgeRating         int16             `json:"age_rating" example:"3"`
	CoverSensitive    bool              `json:"cover_sensitive" example:"false"`
	Status            int16             `json:"status" example:"1"`
	Developer         *DeveloperSummary `json:"developer"`
	Aliases           []string          `json:"aliases"`
	Tags              []TagSummary      `json:"tags"`
	Rating            RatingSummary     `json:"rating"`
	Statistics        GalgameStatistics `json:"statistics"`
	Contributors      []ContributorData `json:"contributors"`
	ContributorCount  int64             `json:"contributor_count" example:"12"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type GalgameListData struct {
	Items []GalgameListItem `json:"items"`
	Total int64             `json:"total" example:"100"`
	Page  int               `json:"page" example:"1"`
	Limit int               `json:"limit" example:"20"`
}

type GalgameListResponse struct {
	Code int             `json:"code" example:"0"`
	Data GalgameListData `json:"data"`
	Msg  string          `json:"msg" example:"success"`
}

type GalgameDataResponse struct {
	Code int             `json:"code" example:"0"`
	Data GalgameResponse `json:"data"`
	Msg  string          `json:"msg" example:"success"`
}

func NewGalgameListItems(galgames []model.Galgame) []GalgameListItem {
	items := make([]GalgameListItem, 0, len(galgames))
	for i := range galgames {
		galgame := &galgames[i]
		items = append(items, GalgameListItem{
			ID:               galgame.ID,
			Title:            galgame.Title,
			OriginalTitle:    galgame.OriginalTitle,
			RomajiTitle:      galgame.RomajiTitle,
			Slug:             galgame.Slug,
			CoverURL:         galgame.CoverURL,
			ReleaseDate:      formatDate(galgame.ReleaseDate),
			AgeRating:        galgame.AgeRating,
			CoverSensitive:   galgame.CoverSensitive,
			Status:           galgame.Status,
			Developer:        newDeveloperSummary(galgame.Developer),
			Tags:             newTagSummaries(galgame.Tags),
			Rating:           newRatingSummary(galgame),
			Statistics:       newStatistics(galgame),
			AIClassification: galgame.AIClassification,
			AIConfidence:     galgame.AIConfidence,
			AIStatus:         galgame.AIStatus,
			AIConflict:       galgame.AIConflict,
		})
	}
	return items
}

func NewGalgameResponse(galgame *model.Galgame) GalgameResponse {
	aliases := make([]string, 0, len(galgame.Aliases))
	for _, alias := range galgame.Aliases {
		aliases = append(aliases, alias.Alias)
	}
	return GalgameResponse{
		ID:                galgame.ID,
		Title:             galgame.Title,
		OriginalTitle:     galgame.OriginalTitle,
		RomajiTitle:       galgame.RomajiTitle,
		Slug:              galgame.Slug,
		Description:       galgame.Description,
		DescriptionSource: galgame.DescriptionSource,
		CoverURL:          galgame.CoverURL,
		BannerURL:         galgame.BannerURL,
		ReleaseDate:       formatDate(galgame.ReleaseDate),
		AgeRating:         galgame.AgeRating,
		CoverSensitive:    galgame.CoverSensitive,
		Status:            galgame.Status,
		Developer:         newDeveloperSummary(galgame.Developer),
		Aliases:           aliases,
		Tags:              newTagSummaries(galgame.Tags),
		Rating:            newRatingSummary(galgame),
		Statistics:        newStatistics(galgame),
		Contributors:      NewContributorData(galgame.Contributors),
		ContributorCount:  galgame.ContributorCount,
		CreatedAt:         galgame.CreatedAt,
		UpdatedAt:         galgame.UpdatedAt,
	}
}

func formatDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02")
	return &formatted
}

func newDeveloperSummary(developer *model.Developer) *DeveloperSummary {
	if developer == nil {
		return nil
	}
	return &DeveloperSummary{ID: developer.ID, Name: developer.Name}
}

func newTagSummaries(tags []model.Tag) []TagSummary {
	responses := make([]TagSummary, 0, len(tags))
	for _, tag := range tags {
		responses = append(responses, TagSummary{ID: tag.ID, Name: tag.Name})
	}
	return responses
}

func newRatingSummary(galgame *model.Galgame) RatingSummary {
	return RatingSummary{Average: galgame.RatingAverage, Count: galgame.RatingCount}
}

func newStatistics(galgame *model.Galgame) GalgameStatistics {
	return GalgameStatistics{
		FavoriteCount: galgame.FavoriteCount,
		ResourceCount: galgame.ResourceCount,
		PostCount:     galgame.PostCount,
	}
}
