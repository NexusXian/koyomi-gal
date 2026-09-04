package dto

import (
	"encoding/json"
	"time"

	"backend/internal/importer/model"
	"backend/internal/importer/provider"
	"backend/internal/importer/service"
)

type ImportSearchQuery struct {
	Provider string `form:"provider" binding:"required,max=32" example:"vndb"`
	Q        string `form:"q" binding:"required,max=255" example:"Summer Pockets"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
}

type ImportDuplicateCandidate struct {
	ID            uint       `json:"id" example:"123"`
	Title         string     `json:"title" example:"夏日口袋"`
	OriginalTitle string     `json:"original_title" example:"サマーポケッツ"`
	ReleaseDate   *time.Time `json:"release_date" example:"2018-06-29T00:00:00Z"`
}

type ExternalGameSummary struct {
	Source        string   `json:"source" example:"vndb"`
	ExternalID    string   `json:"external_id" example:"v20424"`
	URL           string   `json:"url" example:"https://vndb.org/v20424"`
	Title         string   `json:"title" example:"Summer Pockets"`
	OriginalTitle string   `json:"original_title" example:"サマーポケッツ"`
	RomajiTitle   string   `json:"romaji_title" example:"Summer Pockets"`
	CoverURL      string   `json:"cover_url" example:"https://t.vndb.org/cv/12/20424.jpg"`
	ReleaseDate   *string  `json:"release_date" example:"2018-06-29"`
	Developer     string   `json:"developer" example:"Key"`
	Rating        *float64 `json:"rating" example:"8.02"`
	RatingCount   int      `json:"rating_count" example:"5698"`
}

type ExternalGameDetail struct {
	ExternalGameSummary
	Description      string   `json:"description" example:"作品简介"`
	Aliases          []string `json:"aliases"`
	Tags             []string `json:"tags"`
	OriginalLanguage string   `json:"original_language" example:"ja"`
	LengthMinutes    *int     `json:"length_minutes" example:"1860"`
}

type ImportSearchItem struct {
	Game            ExternalGameSummary        `json:"game"`
	DuplicateStatus string                     `json:"duplicate_status" example:"none"`
	Candidates      []ImportDuplicateCandidate `json:"candidates"`
}

type ImportSearchData struct {
	Items []ImportSearchItem `json:"items"`
	Total int                `json:"total" example:"20"`
}

type ImportSearchResponse struct {
	Code int              `json:"code" example:"0"`
	Data ImportSearchData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

type ImportPreviewData struct {
	Game            ExternalGameDetail         `json:"game"`
	DuplicateStatus string                     `json:"duplicate_status" example:"none"`
	Candidates      []ImportDuplicateCandidate `json:"candidates"`
}

type ImportPreviewResponse struct {
	Code int               `json:"code" example:"0"`
	Data ImportPreviewData `json:"data"`
	Msg  string            `json:"msg" example:"success"`
}

type ImportProvidersData struct {
	Providers []string `json:"providers"`
}

type ImportProvidersResponse struct {
	Code int                 `json:"code" example:"0"`
	Data ImportProvidersData `json:"data"`
	Msg  string              `json:"msg" example:"success"`
}

type ImportGameRequest struct {
	Provider            string `json:"provider" binding:"required,max=32" example:"vndb"`
	ExternalID          string `json:"external_id" binding:"required,max=128" example:"v17"`
	DuplicateAction     string `json:"duplicate_action" binding:"omitempty,oneof=error create_new link_existing" example:"error"`
	ExistingGalgameID   *uint  `json:"existing_galgame_id" binding:"omitempty,gt=0" example:"123"`
	ForceMetadataUpdate bool   `json:"force_metadata_update" example:"false"`
}

type ImportResultData struct {
	DuplicateStatus   string                     `json:"duplicate_status" example:"none"`
	GalgameID         *uint                      `json:"galgame_id,omitempty" example:"456"`
	ExistingGalgameID *uint                      `json:"existing_galgame_id,omitempty" example:"123"`
	Candidates        []ImportDuplicateCandidate `json:"candidates"`
}

type ImportGameResponse struct {
	Code int              `json:"code" example:"0"`
	Data ImportResultData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

type CreateImportBatchRequest struct {
	Provider         string   `json:"provider" binding:"required,max=32" example:"vndb"`
	MinRating        *float64 `json:"min_rating" binding:"omitempty,min=0,max=10" example:"7.5"`
	MinVoteCount     *int     `json:"min_vote_count" binding:"omitempty,min=0" example:"100"`
	FromYear         *int     `json:"from_year" binding:"omitempty,min=1950,max=9999" example:"2010"`
	ToYear           *int     `json:"to_year" binding:"omitempty,min=1950,max=9999" example:"2024"`
	OriginalLanguage string   `json:"original_language" binding:"omitempty,max=16" example:"ja"`
	Limit            int      `json:"limit" binding:"required,min=1,max=5000" example:"500"`
}

type ImportJobParams struct {
	MinRating        *float64 `json:"min_rating,omitempty" example:"7.5"`
	MinVoteCount     *int     `json:"min_vote_count,omitempty" example:"100"`
	FromYear         *int     `json:"from_year,omitempty" example:"2010"`
	ToYear           *int     `json:"to_year,omitempty" example:"2024"`
	OriginalLanguage string   `json:"original_language,omitempty" example:"ja"`
	Limit            int      `json:"limit" example:"500"`
}

type ImportJobData struct {
	ID             int64           `json:"id" example:"1"`
	Provider       string          `json:"provider" example:"vndb"`
	JobType        string          `json:"job_type" example:"batch"`
	Status         int16           `json:"status" example:"1"`
	StatusLabel    string          `json:"status_label" example:"running"`
	TotalCount     int             `json:"total_count" example:"500"`
	ProcessedCount int             `json:"processed_count" example:"120"`
	CreatedCount   int             `json:"created_count" example:"100"`
	SkippedCount   int             `json:"skipped_count" example:"15"`
	FailedCount    int             `json:"failed_count" example:"5"`
	Params         ImportJobParams `json:"params"`
	ErrorMessage   string          `json:"error_message" example:""`
	CreatedBy      *uint           `json:"created_by,omitempty" example:"1"`
	CreatedAt      time.Time       `json:"created_at"`
	StartedAt      *time.Time      `json:"started_at"`
	FinishedAt     *time.Time      `json:"finished_at"`
}

type ImportJobDataResponse struct {
	Code int           `json:"code" example:"0"`
	Data ImportJobData `json:"data"`
	Msg  string        `json:"msg" example:"success"`
}

type ImportJobListQuery struct {
	Status *int16 `form:"status" binding:"omitempty,min=0,max=4" example:"1"`
	Page   int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

type ImportJobListData struct {
	Items []ImportJobData `json:"items"`
	Total int64           `json:"total" example:"10"`
	Page  int             `json:"page" example:"1"`
	Limit int             `json:"limit" example:"20"`
}

type ImportJobListResponse struct {
	Code int               `json:"code" example:"0"`
	Data ImportJobListData `json:"data"`
	Msg  string            `json:"msg" example:"success"`
}

func NewProvidersData(providers []string) ImportProvidersData {
	items := make([]string, 0, len(providers))
	items = append(items, providers...)
	return ImportProvidersData{Providers: items}
}

var importJobStatusLabels = map[int16]string{
	0: "pending",
	1: "running",
	2: "succeeded",
	3: "failed",
	4: "cancelled",
}

func NewImportJobData(job *model.ImportJob) ImportJobData {
	var params ImportJobParams
	if len(job.Params) > 0 {
		_ = json.Unmarshal(job.Params, &params)
	}
	return ImportJobData{
		ID:             int64(job.ID),
		Provider:       job.Provider,
		JobType:        job.JobType,
		Status:         job.Status,
		StatusLabel:    importJobStatusLabels[job.Status],
		TotalCount:     job.TotalCount,
		ProcessedCount: job.ProcessedCount,
		CreatedCount:   job.CreatedCount,
		SkippedCount:   job.SkippedCount,
		FailedCount:    job.FailedCount,
		Params:         params,
		ErrorMessage:   job.ErrorMessage,
		CreatedBy:      job.CreatedBy,
		CreatedAt:      job.CreatedAt,
		StartedAt:      job.StartedAt,
		FinishedAt:     job.FinishedAt,
	}
}

func NewImportJobListItems(jobs []model.ImportJob) []ImportJobData {
	items := make([]ImportJobData, 0, len(jobs))
	for i := range jobs {
		items = append(items, NewImportJobData(&jobs[i]))
	}
	return items
}

func NewImportSearchItems(previews []service.PreviewResult) []ImportSearchItem {
	items := make([]ImportSearchItem, 0, len(previews))
	for i := range previews {
		items = append(items, ImportSearchItem{
			Game:            newExternalGameSummary(previews[i].Game),
			DuplicateStatus: previews[i].DuplicateStatus,
			Candidates:      newImportDuplicateCandidates(previews[i].Candidates),
		})
	}
	return items
}

func NewImportPreviewData(preview *service.PreviewResult) ImportPreviewData {
	game := preview.Game
	return ImportPreviewData{
		Game: ExternalGameDetail{
			ExternalGameSummary: newExternalGameSummary(game),
			Description:         game.Description,
			Aliases:             nonEmpty(game.Aliases),
			Tags:                tagNames(game.Tags),
			OriginalLanguage:    game.OriginalLanguage,
			LengthMinutes:       game.LengthMinutes,
		},
		DuplicateStatus: preview.DuplicateStatus,
		Candidates:      newImportDuplicateCandidates(preview.Candidates),
	}
}

func NewImportResultData(result *service.ImportResult) ImportResultData {
	return ImportResultData{
		DuplicateStatus:   result.DuplicateStatus,
		GalgameID:         result.GalgameID,
		ExistingGalgameID: result.ExistingGalgameID,
		Candidates:        newImportDuplicateCandidates(result.Candidates),
	}
}

func newExternalGameSummary(game *provider.ExternalGame) ExternalGameSummary {
	developer := ""
	if game.Developer != nil {
		developer = game.Developer.Name
	}
	ratingCount := 0
	if game.RatingCount != nil {
		ratingCount = *game.RatingCount
	}
	return ExternalGameSummary{
		Source:        game.Source,
		ExternalID:    game.ExternalID,
		URL:           externalGameURL(game.Source, game.ExternalID),
		Title:         game.Title,
		OriginalTitle: game.OriginalTitle,
		RomajiTitle:   game.RomajiTitle,
		CoverURL:      game.CoverURL,
		ReleaseDate:   formatDate(game.ReleaseDate),
		Developer:     developer,
		Rating:        game.Rating,
		RatingCount:   ratingCount,
	}
}

func newImportDuplicateCandidates(candidates []service.DuplicateCandidate) []ImportDuplicateCandidate {
	items := make([]ImportDuplicateCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, ImportDuplicateCandidate{
			ID:            candidate.ID,
			Title:         candidate.Title,
			OriginalTitle: candidate.OriginalTitle,
			ReleaseDate:   candidate.ReleaseDate,
		})
	}
	return items
}

func formatDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02")
	return &formatted
}

func nonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func tagNames(tags []provider.ExternalTag) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag.Name != "" {
			result = append(result, tag.Name)
		}
	}
	return result
}

func externalGameURL(source, externalID string) string {
	if source == "vndb" {
		return "https://vndb.org/" + externalID
	}
	return ""
}
