package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	galgameModel "backend/internal/galgame/model"
	importerModel "backend/internal/importer/model"
	"backend/internal/importer/provider"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

const (
	enrichPageSize    = 50
	maxEnrichLimit    = 5000
	enrichSearchLimit = 10
)

// enrichRequireSource is the discovery source a galgame must already have
// before enrichment runs; VNDB stays responsible for visual-novel identity.
const enrichRequireSource = "vndb"

// EnrichStats is the JSONB stats payload persisted on enrich import jobs.
type EnrichStats struct {
	Matched  int `json:"matched"`
	Review   int `json:"review"`
	NotFound int `json:"not_found"`
	Failed   int `json:"failed"`
}

// EnrichParams is the persisted parameter set of an enrich job.
type EnrichParams struct {
	Limit int `json:"limit"`
}

// SetEnrichEnqueuer wires the Asynq enqueue callback for enrich jobs.
func (s *Service) SetEnrichEnqueuer(fn func(ctx context.Context, jobID int64) error) {
	s.enrichEnqueuer = fn
}

// CreateEnrichJob creates and dispatches a batch enrichment job that links
// the target provider to every galgame discovered via the identity source.
func (s *Service) CreateEnrichJob(
	ctx context.Context,
	providerName string,
	limit int,
	createdBy *uint,
) (*importerModel.ImportJob, error) {
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	if _, err := s.provider(providerName); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > maxEnrichLimit {
		limit = maxEnrichLimit
	}

	stats, err := json.Marshal(EnrichStats{})
	if err != nil {
		return nil, fmt.Errorf("encode enrich stats: %w", err)
	}
	encodedParams, err := json.Marshal(EnrichParams{Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("encode enrich params: %w", err)
	}
	job := &importerModel.ImportJob{
		Provider:  providerName,
		JobType:   importerModel.ImportJobTypeEnrich,
		Status:    importerModel.ImportJobStatusPending,
		Params:    encodedParams,
		Stats:     stats,
		CreatedBy: createdBy,
	}
	if err := s.repository.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	if s.enrichEnqueuer == nil {
		_ = s.repository.FinishJob(ctx, int64(job.ID), importerModel.ImportJobStatusFailed, "enrich queue is not configured")
		return nil, errors.New("enrich queue is not configured")
	}
	if err := s.enrichEnqueuer(ctx, int64(job.ID)); err != nil {
		_ = s.repository.FinishJob(ctx, int64(job.ID), importerModel.ImportJobStatusFailed, "enqueue enrich task failed")
		return nil, fmt.Errorf("enqueue enrich job: %w", err)
	}
	created, err := s.repository.FindJob(ctx, int64(job.ID))
	if err != nil {
		return nil, err
	}
	return created, nil
}

// executeEnrichJob walks galgames that have the identity source but no
// target-provider source, matches them, and links or queues candidates.
func (s *Service) executeEnrichJob(ctx context.Context, job *importerModel.ImportJob) error {
	selected, err := s.provider(job.Provider)
	if err != nil {
		return err
	}
	var params EnrichParams
	if len(job.Params) > 0 {
		if err := json.Unmarshal(job.Params, &params); err != nil {
			return fmt.Errorf("decode enrich params: %w", err)
		}
	}
	limit := params.Limit
	if limit <= 0 {
		limit = maxEnrichLimit
	}

	total, err := s.repository.CountGalgamesForEnrichment(ctx, enrichRequireSource, job.Provider)
	if err != nil {
		return err
	}
	if total > int64(limit) {
		total = int64(limit)
	}
	if err := s.repository.SetJobTotal(ctx, int64(job.ID), int(total)); err != nil {
		return err
	}

	stats := EnrichStats{}
	if len(job.Stats) > 0 {
		_ = json.Unmarshal(job.Stats, &stats)
	}

	processed := 0
	afterID := uint(0)
	for processed < limit {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("enrich cancelled: %w", err)
		}
		games, err := s.repository.ListGalgamesForEnrichment(ctx, enrichRequireSource, job.Provider, afterID, enrichPageSize)
		if err != nil {
			return err
		}
		if len(games) == 0 {
			break
		}
		for i := range games {
			if processed >= limit {
				break
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("enrich cancelled: %w", err)
			}
			afterID = games[i].ID
			outcome := s.enrichOne(ctx, selected, &games[i], &stats)
			processed++
			if err := s.repository.UpdateJobProgress(
				ctx, int64(job.ID), processed, stats.Matched, stats.NotFound, stats.Failed,
			); err != nil {
				logger.Error("update enrich job progress",
					zap.Int64("job_id", int64(job.ID)), zap.Error(err))
			}
			if err := s.repository.UpdateJobStats(ctx, int64(job.ID), stats); err != nil {
				logger.Error("update enrich job stats",
					zap.Int64("job_id", int64(job.ID)), zap.Error(err))
			}
			if outcome != nil {
				logger.Warn("enrich item failed",
					zap.Int64("job_id", int64(job.ID)),
					zap.Uint("galgame_id", games[i].ID),
					zap.Error(outcome))
			}
		}
	}
	logger.Info("enrich job finished",
		zap.Int64("job_id", int64(job.ID)),
		zap.Int("matched", stats.Matched),
		zap.Int("review", stats.Review),
		zap.Int("not_found", stats.NotFound),
		zap.Int("failed", stats.Failed))
	return nil
}

// enrichOne matches a single galgame and applies the outcome. The returned
// error is logged but never aborts the job.
func (s *Service) enrichOne(
	ctx context.Context,
	selected provider.Provider,
	galgame *galgameModel.Galgame,
	stats *EnrichStats,
) error {
	input := matchInputFromGalgame(galgame)
	query := input.SearchQuery()
	if query == "" {
		stats.NotFound++
		return nil
	}
	results, err := selected.Search(ctx, query, enrichSearchLimit)
	if err != nil {
		stats.Failed++
		return err
	}
	matches := MatchBangumiCandidates(input, results)
	if len(matches) == 0 {
		stats.NotFound++
		return nil
	}
	best := matches[0]
	if best.Confidence >= autoMatchThreshold {
		if _, err := s.Enrich(ctx, galgame.ID, best.Game.Source, best.Game.ExternalID, DefaultEnrichOptions()); err != nil {
			stats.Failed++
			return err
		}
		stats.Matched++
		return nil
	}
	if err := s.SaveMatchCandidates(ctx, galgame.ID, matches); err != nil {
		stats.Failed++
		return err
	}
	stats.Review++
	return nil
}

func matchInputFromGalgame(galgame *galgameModel.Galgame) MatchInput {
	input := MatchInput{
		Title:         galgame.Title,
		OriginalTitle: galgame.OriginalTitle,
		RomajiTitle:   galgame.RomajiTitle,
		ReleaseDate:   galgame.ReleaseDate,
	}
	for _, alias := range galgame.Aliases {
		if strings.TrimSpace(alias.Alias) != "" {
			input.Aliases = append(input.Aliases, alias.Alias)
		}
	}
	if galgame.Developer != nil {
		input.Developer = galgame.Developer.Name
	}
	return input
}
