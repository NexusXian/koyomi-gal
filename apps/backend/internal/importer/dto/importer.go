package dto

import (
	"encoding/json"
	"errors"
	"time"

	"backend/internal/importer/model"
	"backend/internal/importer/provider"
	"backend/internal/importer/repository"
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

type ImportJobStats struct {
	Matched  int `json:"matched" example:"2834"`
	Review   int `json:"review" example:"623"`
	NotFound int `json:"not_found" example:"917"`
	Failed   int `json:"failed" example:"12"`
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
	Stats          ImportJobStats  `json:"stats"`
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

type EnrichStatsQuery struct {
	Provider string `form:"provider" binding:"omitempty,max=32" example:"bangumi"`
}

type EnrichStatsData struct {
	Provider       string `json:"provider" example:"bangumi"`
	VndbCount      int64  `json:"vndb_count" example:"5000"`
	LinkedCount    int64  `json:"linked_count" example:"3200"`
	UnlinkedCount  int64  `json:"unlinked_count" example:"1800"`
	PendingMatches int64  `json:"pending_matches" example:"613"`
}

type EnrichStatsResponse struct {
	Code int             `json:"code" example:"0"`
	Data EnrichStatsData `json:"data"`
	Msg  string          `json:"msg" example:"success"`
}

type CreateEnrichBatchRequest struct {
	Provider string `json:"provider" binding:"required,max=32" example:"bangumi"`
	Limit    int    `json:"limit" binding:"required,min=1,max=5000" example:"1000"`
}

type ExternalCandidateQuery struct {
	Provider string `form:"provider" binding:"required,max=32" example:"bangumi"`
}

type ExternalCandidateItem struct {
	ExternalID    string   `json:"external_id" example:"200763"`
	Source        string   `json:"source" example:"bangumi"`
	URL           string   `json:"url" example:"https://bgm.tv/subject/200763"`
	Title         string   `json:"title" example:"夏日口袋"`
	OriginalTitle string   `json:"original_title" example:"Summer Pockets"`
	CoverURL      string   `json:"cover_url" example:"https://lain.bgm.tv/pic/cover/l/200763.jpg"`
	ReleaseDate   *string  `json:"release_date" example:"2018-06-29"`
	Rating        *float64 `json:"rating" example:"8.2"`
	RatingCount   *int     `json:"rating_count" example:"5819"`
	Confidence    float64  `json:"confidence" example:"0.94"`
	Reasons       []string `json:"reasons"`
	Linked        bool     `json:"linked" example:"false"`
}

type ExternalCandidateListData struct {
	Items []ExternalCandidateItem `json:"items"`
	Total int                     `json:"total" example:"2"`
}

type ExternalCandidateListResponse struct {
	Code int                       `json:"code" example:"0"`
	Data ExternalCandidateListData `json:"data"`
	Msg  string                    `json:"msg" example:"success"`
}

type EnrichGalgameRequest struct {
	Provider   string   `json:"provider" binding:"required,max=32" example:"bangumi"`
	ExternalID string   `json:"external_id" binding:"required,max=128" example:"200763"`
	Fields     []string `json:"fields" binding:"omitempty,dive,oneof=title description aliases cover tags" example:"title,description,aliases,tags"`
	Force      bool     `json:"force" example:"false"`
}

type EnrichResultData struct {
	GalgameID     uint     `json:"galgame_id" example:"123"`
	Provider      string   `json:"provider" example:"bangumi"`
	ExternalID    string   `json:"external_id" example:"200763"`
	UpdatedFields []string `json:"updated_fields"`
}

type EnrichGalgameResponse struct {
	Code int              `json:"code" example:"0"`
	Data EnrichResultData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

type MatchCandidateQuery struct {
	Status *int16 `form:"status" binding:"omitempty,min=0,max=2" example:"0"`
	Page   int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

type MatchCandidatePreview struct {
	ExternalID    string   `json:"external_id" example:"200763"`
	Title         string   `json:"title" example:"夏日口袋"`
	OriginalTitle string   `json:"original_title" example:"Summer Pockets"`
	CoverURL      string   `json:"cover_url" example:"https://lain.bgm.tv/pic/cover/l/200763.jpg"`
	Rating        *float64 `json:"rating" example:"8.2"`
	RatingCount   *int     `json:"rating_count" example:"5819"`
	URL           string   `json:"url" example:"https://bgm.tv/subject/200763"`
}

type MatchCandidateItem struct {
	ID                   uint64                 `json:"id" example:"1"`
	GalgameID            uint                   `json:"galgame_id" example:"123"`
	GalgameTitle         string                 `json:"galgame_title" example:"Summer Pockets"`
	GalgameOriginalTitle string                 `json:"galgame_original_title" example:"サマーポケッツ"`
	Provider             string                 `json:"provider" example:"bangumi"`
	ExternalID           string                 `json:"external_id" example:"200763"`
	Confidence           float64                `json:"confidence" example:"0.72"`
	Reasons              []string               `json:"reasons"`
	Preview              *MatchCandidatePreview `json:"preview,omitempty"`
	Status               int16                  `json:"status" example:"0"`
	StatusLabel          string                 `json:"status_label" example:"pending"`
	CreatedAt            time.Time              `json:"created_at"`
	ReviewedAt           *time.Time             `json:"reviewed_at"`
}

type MatchCandidateListData struct {
	Items []MatchCandidateItem `json:"items"`
	Total int64                `json:"total" example:"613"`
	Page  int                  `json:"page" example:"1"`
	Limit int                  `json:"limit" example:"20"`
}

type MatchCandidateListResponse struct {
	Code int                    `json:"code" example:"0"`
	Data MatchCandidateListData `json:"data"`
	Msg  string                 `json:"msg" example:"success"`
}

type MatchCandidateBatchRequest struct {
	IDs []uint64 `json:"ids" binding:"required,min=1,max=100,dive,gt=0" example:"1,2,3"`
}

type MatchCandidateBatchItem struct {
	ID      uint64 `json:"id" example:"1"`
	Status  string `json:"status" example:"approved"`
	Message string `json:"message,omitempty" example:"匹配候选已审核"`
}

type MatchCandidateBatchData struct {
	Items          []MatchCandidateBatchItem `json:"items"`
	ProcessedCount int                       `json:"processed_count" example:"3"`
	SucceededCount int                       `json:"succeeded_count" example:"2"`
	FailedCount    int                       `json:"failed_count" example:"1"`
}

type MatchCandidateBatchResponse struct {
	Code int                     `json:"code" example:"0"`
	Data MatchCandidateBatchData `json:"data"`
	Msg  string                  `json:"msg" example:"success"`
}

func NewEnrichResultData(result *service.EnrichResult) EnrichResultData {
	fields := result.UpdatedFields
	if fields == nil {
		fields = []string{}
	}
	return EnrichResultData{
		GalgameID:     result.GalgameID,
		Provider:      result.Provider,
		ExternalID:    result.ExternalID,
		UpdatedFields: fields,
	}
}

func NewEnrichStatsData(overview repository.EnrichOverview, providerName string) EnrichStatsData {
	return EnrichStatsData{
		Provider:       providerName,
		VndbCount:      overview.VndbCount,
		LinkedCount:    overview.LinkedCount,
		UnlinkedCount:  overview.UnlinkedCount,
		PendingMatches: overview.PendingCandidates,
	}
}

func NewExternalCandidateItems(candidates []service.ExternalCandidate) []ExternalCandidateItem {
	items := make([]ExternalCandidateItem, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, ExternalCandidateItem{
			ExternalID:    candidate.ExternalID,
			Source:        candidate.Source,
			URL:           externalGameURL(candidate.Source, candidate.ExternalID),
			Title:         candidate.Title,
			OriginalTitle: candidate.OriginalTitle,
			CoverURL:      candidate.CoverURL,
			ReleaseDate:   formatDate(candidate.ReleaseDate),
			Rating:        candidate.Rating,
			RatingCount:   candidate.RatingCount,
			Confidence:    candidate.Confidence,
			Reasons:       candidate.Reasons,
			Linked:        candidate.Linked,
		})
	}
	return items
}

func NewMatchCandidateItems(candidates []model.ExternalMatchCandidate) []MatchCandidateItem {
	items := make([]MatchCandidateItem, 0, len(candidates))
	statusLabels := map[int16]string{0: "pending", 1: "approved", 2: "rejected"}
	for _, candidate := range candidates {
		var reasons []string
		if len(candidate.Reasons) > 0 {
			_ = json.Unmarshal(candidate.Reasons, &reasons)
		}
		item := MatchCandidateItem{
			ID:                   candidate.ID,
			GalgameID:            candidate.GalgameID,
			GalgameTitle:         candidate.GalgameTitle,
			GalgameOriginalTitle: candidate.GalgameOriginalTitle,
			Provider:             candidate.Provider,
			ExternalID:           candidate.ExternalID,
			Confidence:           candidate.Confidence,
			Reasons:              reasons,
			Status:               candidate.Status,
			StatusLabel:          statusLabels[candidate.Status],
			CreatedAt:            candidate.CreatedAt,
			ReviewedAt:           candidate.ReviewedAt,
		}
		if len(candidate.Preview) > 0 {
			var preview MatchCandidatePreview
			if err := json.Unmarshal(candidate.Preview, &preview); err == nil {
				item.Preview = &preview
			}
		}
		items = append(items, item)
	}
	return items
}

func NewMatchCandidateBatchData(summary service.MatchCandidateBatchSummary, successStatus string) MatchCandidateBatchData {
	items := make([]MatchCandidateBatchItem, 0, len(summary.Results))
	for _, result := range summary.Results {
		item := MatchCandidateBatchItem{ID: result.ID}
		if result.Err == nil {
			item.Status = successStatus
		} else {
			item.Status = "failed"
			item.Message = matchCandidateBatchMessage(result.Err)
		}
		items = append(items, item)
	}
	return MatchCandidateBatchData{
		Items:          items,
		ProcessedCount: len(items),
		SucceededCount: summary.SucceededCount,
		FailedCount:    summary.FailedCount,
	}
}

func matchCandidateBatchMessage(err error) string {
	switch {
	case errors.Is(err, service.ErrMatchCandidateNotFound):
		return "匹配候选不存在"
	case errors.Is(err, service.ErrMatchCandidateReviewed):
		return "匹配候选已审核"
	case errors.Is(err, service.ErrExternalGameNotFound):
		return "外部条目不存在"
	default:
		return "处理失败"
	}
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
	var stats ImportJobStats
	if len(job.Stats) > 0 {
		_ = json.Unmarshal(job.Stats, &stats)
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
		Stats:          stats,
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
	switch source {
	case "vndb":
		return "https://vndb.org/" + externalID
	case "bangumi":
		return "https://bgm.tv/subject/" + externalID
	default:
		return ""
	}
}
