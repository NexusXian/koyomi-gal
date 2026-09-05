package dto

import (
	"time"

	"backend/internal/novel/model"
)

type CreateNovelRequest struct {
	Title            string `json:"title" binding:"required,max=255" example:"青春猪头少年不会梦到兔女郎学姐"`
	OriginalTitle    string `json:"original_title" binding:"max=255" example:"青春ブタ野郎はバニーガール先輩の夢を見ない"`
	Slug             string `json:"slug" binding:"required,max=255" example:"seishun-buta-yarou"`
	Summary          string `json:"summary" example:"小说简介（Markdown）"`
	CoverURL         string `json:"cover_url" binding:"max=2048" example:"https://example.com/cover.jpg"`
	Author           string `json:"author" binding:"max=255" example:"鸭志田一"`
	Illustrator      string `json:"illustrator" binding:"max=255" example:"沟口凯吉"`
	Publisher        string `json:"publisher" binding:"max=255" example:"KADOKAWA"`
	Label            string `json:"label" binding:"max=255" example:"电击文库"`
	Language         string `json:"language" binding:"max=16" example:"ja"`
	Region           string `json:"region" binding:"max=16" example:"JP"`
	ReleaseStatus    string `json:"release_status" binding:"omitempty,oneof=ongoing completed hiatus cancelled unknown" example:"ongoing"`
	FirstReleaseDate string `json:"first_release_date" example:"2014-04-10"`
	AgeRating        int16  `json:"age_rating" binding:"oneof=0 1 2 3 4 5" example:"1"`
	IsCoverSensitive bool   `json:"is_cover_sensitive" example:"false"`
	OfficialWebsite  string `json:"official_website" binding:"max=2048" example:"https://dengeki.com/"`
	Status           int16  `json:"status" binding:"oneof=0 1" example:"0"`
	TagIDs           []uint `json:"tag_ids" binding:"max=100,dive,gt=0" example:"1,2"`
}

type UpdateNovelRequest struct {
	Title            string `json:"title" binding:"required,max=255" example:"青春猪头少年不会梦到兔女郎学姐"`
	OriginalTitle    string `json:"original_title" binding:"max=255" example:"青春ブタ野郎はバニーガール先輩の夢を見ない"`
	Slug             string `json:"slug" binding:"required,max=255" example:"seishun-buta-yarou"`
	Summary          string `json:"summary" example:"小说简介（Markdown）"`
	CoverURL         string `json:"cover_url" binding:"max=2048" example:"https://example.com/cover.jpg"`
	Author           string `json:"author" binding:"max=255" example:"鸭志田一"`
	Illustrator      string `json:"illustrator" binding:"max=255" example:"沟口凯吉"`
	Publisher        string `json:"publisher" binding:"max=255" example:"KADOKAWA"`
	Label            string `json:"label" binding:"max=255" example:"电击文库"`
	Language         string `json:"language" binding:"max=16" example:"ja"`
	Region           string `json:"region" binding:"max=16" example:"JP"`
	ReleaseStatus    string `json:"release_status" binding:"required,oneof=ongoing completed hiatus cancelled unknown" example:"ongoing"`
	FirstReleaseDate string `json:"first_release_date" example:"2014-04-10"`
	AgeRating        *int16 `json:"age_rating" binding:"required,oneof=0 1 2 3 4 5" example:"1"`
	IsCoverSensitive *bool  `json:"is_cover_sensitive" binding:"required" example:"false"`
	OfficialWebsite  string `json:"official_website" binding:"max=2048" example:"https://dengeki.com/"`
	Status           *int16 `json:"status" binding:"required,oneof=0 1 2 3" example:"1"`
	TagIDs           []uint `json:"tag_ids" binding:"max=100,dive,gt=0" example:"1,2"`
}

// ReviewNovelRequest approves (1) or rejects (2); a reject reason is stored
// and shown to the submitter.
type ReviewNovelRequest struct {
	Status int16  `json:"status" binding:"required,oneof=1 2" example:"1"`
	Reason string `json:"reason" binding:"omitempty,max=1000" example:"资料不完整"`
}

type NovelQuery struct {
	Keyword       string `form:"keyword" binding:"max=255"`
	TagIDs        []uint `form:"-"`
	Author        string `form:"author" binding:"max=255"`
	Publisher     string `form:"publisher" binding:"max=255"`
	Label         string `form:"label" binding:"max=255"`
	ReleaseStatus string `form:"release_status" binding:"omitempty,oneof=ongoing completed hiatus cancelled unknown"`
	Language      string `form:"language" binding:"max=16"`
	Sort          string `form:"sort"`
	Page          int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit         int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// AdminNovelQuery filters the novel:review admin listing, which returns every
// status instead of only published entries.
type AdminNovelQuery struct {
	Status        *int16 `form:"status" binding:"omitempty,oneof=0 1 2 3" example:"0"`
	Keyword       string `form:"keyword" binding:"max=255"`
	ReleaseStatus string `form:"release_status" binding:"omitempty,oneof=ongoing completed hiatus cancelled unknown"`
	Sort          string `form:"sort"`
	Page          int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit         int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

type NovelStatistics struct {
	VolumeCount   int64 `json:"volume_count" example:"13"`
	ResourceCount int64 `json:"resource_count" example:"5"`
}

type NovelListItem struct {
	ID               uint            `json:"id" example:"1"`
	Title            string          `json:"title" example:"青春猪头少年不会梦到兔女郎学姐"`
	OriginalTitle    string          `json:"original_title" example:"青春ブタ野郎はバニーガール先輩の夢を見ない"`
	Slug             string          `json:"slug" example:"seishun-buta-yarou"`
	CoverURL         string          `json:"cover_url" example:"https://example.com/cover.jpg"`
	Author           string          `json:"author" example:"鸭志田一"`
	Publisher        string          `json:"publisher" example:"KADOKAWA"`
	Label            string          `json:"label" example:"电击文库"`
	Language         string          `json:"language" example:"ja"`
	ReleaseStatus    string          `json:"release_status" example:"ongoing"`
	FirstReleaseDate *string         `json:"first_release_date" example:"2014-04-10"`
	AgeRating        int16           `json:"age_rating" example:"1"`
	IsCoverSensitive bool            `json:"is_cover_sensitive" example:"false"`
	Status           int16           `json:"status" example:"1"`
	Tags             []TagSummary    `json:"tags"`
	Statistics       NovelStatistics `json:"statistics"`
	UpdatedAt        time.Time       `json:"updated_at"`
	CreatedAt        time.Time       `json:"created_at"`
}

type NovelResponse struct {
	ID               uint              `json:"id" example:"1"`
	Title            string            `json:"title" example:"青春猪头少年不会梦到兔女郎学姐"`
	OriginalTitle    string            `json:"original_title" example:"青春ブタ野郎はバニーガール先輩の夢を見ない"`
	Slug             string            `json:"slug" example:"seishun-buta-yarou"`
	Summary          string            `json:"summary" example:"小说简介（Markdown）"`
	CoverURL         string            `json:"cover_url" example:"https://example.com/cover.jpg"`
	Author           string            `json:"author" example:"鸭志田一"`
	Illustrator      string            `json:"illustrator" example:"沟口凯吉"`
	Publisher        string            `json:"publisher" example:"KADOKAWA"`
	Label            string            `json:"label" example:"电击文库"`
	Language         string            `json:"language" example:"ja"`
	Region           string            `json:"region" example:"JP"`
	ReleaseStatus    string            `json:"release_status" example:"ongoing"`
	FirstReleaseDate *string           `json:"first_release_date" example:"2014-04-10"`
	AgeRating        int16             `json:"age_rating" example:"1"`
	IsCoverSensitive bool              `json:"is_cover_sensitive" example:"false"`
	OfficialWebsite  string            `json:"official_website" example:"https://dengeki.com/"`
	Status           int16             `json:"status" example:"1"`
	ReviewedAt       *time.Time        `json:"reviewed_at"`
	RejectReason     string            `json:"reject_reason" example:"资料不完整"`
	Tags             []TagSummary      `json:"tags"`
	Volumes          []VolumeSummary   `json:"volumes"`
	RelatedGalgames  []RelatedWorkData `json:"related_galgames"`
	Statistics       NovelStatistics   `json:"statistics"`
	Contributors     []ContributorData `json:"contributors"`
	ContributorCount int64             `json:"contributor_count" example:"12"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type NovelListData struct {
	Items []NovelListItem `json:"items"`
	Total int64           `json:"total" example:"100"`
	Page  int             `json:"page" example:"1"`
	Limit int             `json:"limit" example:"20"`
}

type NovelListResponse struct {
	Code int           `json:"code" example:"0"`
	Data NovelListData `json:"data"`
	Msg  string        `json:"msg" example:"success"`
}

type NovelDataResponse struct {
	Code int           `json:"code" example:"0"`
	Data NovelResponse `json:"data"`
	Msg  string        `json:"msg" example:"success"`
}

func NewNovelListItems(novels []model.Novel) []NovelListItem {
	items := make([]NovelListItem, 0, len(novels))
	for i := range novels {
		novel := &novels[i]
		items = append(items, NovelListItem{
			ID:               novel.ID,
			Title:            novel.Title,
			OriginalTitle:    novel.OriginalTitle,
			Slug:             novel.Slug,
			CoverURL:         novel.CoverURL,
			Author:           novel.Author,
			Publisher:        novel.Publisher,
			Label:            novel.Label,
			Language:         novel.Language,
			ReleaseStatus:    novel.ReleaseStatus,
			FirstReleaseDate: formatDate(novel.FirstReleaseDate),
			AgeRating:        novel.AgeRating,
			IsCoverSensitive: novel.IsCoverSensitive,
			Status:           novel.Status,
			Tags:             newTagSummaries(novel.Tags),
			Statistics:       NovelStatistics{VolumeCount: novel.VolumeCount, ResourceCount: novel.ResourceCount},
			UpdatedAt:        novel.UpdatedAt,
			CreatedAt:        novel.CreatedAt,
		})
	}
	return items
}

func NewNovelResponse(novel *model.Novel) NovelResponse {
	return NovelResponse{
		ID:               novel.ID,
		Title:            novel.Title,
		OriginalTitle:    novel.OriginalTitle,
		Slug:             novel.Slug,
		Summary:          novel.Summary,
		CoverURL:         novel.CoverURL,
		Author:           novel.Author,
		Illustrator:      novel.Illustrator,
		Publisher:        novel.Publisher,
		Label:            novel.Label,
		Language:         novel.Language,
		Region:           novel.Region,
		ReleaseStatus:    novel.ReleaseStatus,
		FirstReleaseDate: formatDate(novel.FirstReleaseDate),
		AgeRating:        novel.AgeRating,
		IsCoverSensitive: novel.IsCoverSensitive,
		OfficialWebsite:  novel.OfficialWebsite,
		Status:           novel.Status,
		ReviewedAt:       novel.ReviewedAt,
		RejectReason:     novel.RejectReason,
		Tags:             newTagSummaries(novel.Tags),
		Volumes:          NewVolumeSummaries(novel.Volumes),
		RelatedGalgames:  NewRelatedWorkData(novel.RelatedGalgames),
		Statistics:       NovelStatistics{VolumeCount: novel.VolumeCount, ResourceCount: novel.ResourceCount},
		Contributors:     NewContributorData(novel.Contributors),
		ContributorCount: novel.ContributorCount,
		CreatedAt:        novel.CreatedAt,
		UpdatedAt:        novel.UpdatedAt,
	}
}

func formatDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02")
	return &formatted
}
