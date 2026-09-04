package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	importerModel "backend/internal/importer/model"
	"backend/internal/importer/provider"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

const (
	batchPageSize = 100
	maxBatchLimit = 5000
)

var (
	ErrBatchUnsupported  = errors.New("provider does not support batch import")
	ErrImportJobNotFound = errors.New("import job not found")
)

// BatchParams is the persisted parameter set of a batch import job.
type BatchParams struct {
	MinRating        *float64 `json:"min_rating,omitempty"`
	MinVoteCount     *int     `json:"min_vote_count,omitempty"`
	FromYear         *int     `json:"from_year,omitempty"`
	ToYear           *int     `json:"to_year,omitempty"`
	OriginalLanguage string   `json:"original_language,omitempty"`
	Limit            int      `json:"limit"`
}

// SetBatchEnqueuer wires the Asynq enqueue callback used to dispatch batch
// jobs. The queue layer owns the callback to avoid an import cycle.
func (s *Service) SetBatchEnqueuer(fn func(ctx context.Context, jobID int64) error) {
	s.batchEnqueuer = fn
}

func (s *Service) CreateBatchJob(
	ctx context.Context,
	providerName string,
	params BatchParams,
	createdBy *uint,
) (*importerModel.ImportJob, error) {
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	selected, err := s.provider(providerName)
	if err != nil {
		return nil, err
	}
	if _, ok := selected.(provider.BatchLister); !ok {
		return nil, ErrBatchUnsupported
	}
	if params.Limit <= 0 || params.Limit > maxBatchLimit {
		params.Limit = maxBatchLimit
	}
	if params.MinRating != nil && (*params.MinRating < 0 || *params.MinRating > 10) {
		return nil, errors.New("min_rating must be between 0 and 10")
	}

	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encode batch params: %w", err)
	}
	job := &importerModel.ImportJob{
		Provider:  providerName,
		JobType:   importerModel.ImportJobTypeBatch,
		Status:    importerModel.ImportJobStatusPending,
		Params:    encoded,
		CreatedBy: createdBy,
	}
	if err := s.repository.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	if s.batchEnqueuer == nil {
		_ = s.repository.FinishJob(ctx, int64(job.ID), importerModel.ImportJobStatusFailed, "batch queue is not configured")
		return nil, errors.New("batch queue is not configured")
	}
	if err := s.batchEnqueuer(ctx, int64(job.ID)); err != nil {
		_ = s.repository.FinishJob(ctx, int64(job.ID), importerModel.ImportJobStatusFailed, "enqueue batch task failed")
		return nil, fmt.Errorf("enqueue batch import: %w", err)
	}
	created, err := s.repository.FindJob(ctx, int64(job.ID))
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) GetImportJob(ctx context.Context, id int64) (*importerModel.ImportJob, error) {
	job, err := s.repository.FindJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrImportJobNotFound
	}
	return job, nil
}

func (s *Service) ListImportJobs(
	ctx context.Context,
	status *int16,
	page, limit int,
) ([]importerModel.ImportJob, int64, error) {
	return s.repository.ListJobs(ctx, status, page, limit)
}

// RunImportJob executes one Asynq batch task. Item-level failures are counted
// and never abort the job; only job-level failures mark it failed.
func (s *Service) RunImportJob(ctx context.Context, jobID int64) error {
	job, err := s.repository.FindJob(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return ErrImportJobNotFound
	}
	if job.Status != importerModel.ImportJobStatusPending {
		return nil
	}
	started, err := s.repository.StartJob(ctx, jobID)
	if err != nil {
		return err
	}
	if !started {
		return nil
	}

	if err := s.executeBatchJob(ctx, job); err != nil {
		message := err.Error()
		if len(message) > 2000 {
			message = message[:2000]
		}
		_ = s.repository.FinishJob(ctx, jobID, importerModel.ImportJobStatusFailed, message)
		return err
	}
	return s.repository.FinishJob(ctx, jobID, importerModel.ImportJobStatusSucceeded, "")
}

func (s *Service) executeBatchJob(ctx context.Context, job *importerModel.ImportJob) error {
	var params BatchParams
	if len(job.Params) > 0 {
		if err := json.Unmarshal(job.Params, &params); err != nil {
			return fmt.Errorf("decode batch params: %w", err)
		}
	}
	selected, err := s.provider(job.Provider)
	if err != nil {
		return err
	}
	lister, ok := selected.(provider.BatchLister)
	if !ok {
		return ErrBatchUnsupported
	}

	filters := provider.BatchFilters{OriginalLanguage: params.OriginalLanguage}
	if params.MinRating != nil {
		filters.MinRating = *params.MinRating
	}
	if params.MinVoteCount != nil {
		filters.MinVoteCount = *params.MinVoteCount
	}
	if params.FromYear != nil {
		filters.FromYear = *params.FromYear
	}
	if params.ToYear != nil {
		filters.ToYear = *params.ToYear
	}
	limit := params.Limit
	if limit <= 0 {
		limit = maxBatchLimit
	}

	processed, created, skipped, failed := 0, 0, 0, 0
	page := 1
	for processed < limit {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("batch cancelled: %w", err)
		}
		result, err := lister.ListBatch(ctx, filters, page, batchPageSize, page == 1)
		if err != nil {
			return fmt.Errorf("fetch batch page %d: %w", page, err)
		}
		if page == 1 && result.Count >= 0 {
			total := int(result.Count)
			if total > limit {
				total = limit
			}
			_ = s.repository.SetJobTotal(ctx, int64(job.ID), total)
		}
		if len(result.Games) == 0 {
			break
		}
		for i := range result.Games {
			if processed >= limit {
				break
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("batch cancelled: %w", err)
			}
			game := &result.Games[i]
			outcome := s.importBatchItem(ctx, game)
			processed++
			switch outcome {
			case batchOutcomeCreated:
				created++
			case batchOutcomeSkipped:
				skipped++
			default:
				failed++
			}
			if err := s.repository.UpdateJobProgress(
				ctx, int64(job.ID), processed, created, skipped, failed,
			); err != nil {
				logger.Error("update import job progress",
					zap.Int64("job_id", int64(job.ID)), zap.Error(err))
			}
		}
		if processed >= limit || !result.More {
			break
		}
		page++
	}
	logger.Info("batch import finished",
		zap.Int64("job_id", int64(job.ID)),
		zap.Int("created", created),
		zap.Int("skipped", skipped),
		zap.Int("failed", failed))
	return nil
}

const (
	batchOutcomeCreated = iota
	batchOutcomeSkipped
	batchOutcomeFailed
)

func (s *Service) importBatchItem(ctx context.Context, game *provider.ExternalGame) int {
	result, err := s.importFetched(ctx, game, ImportInput{
		DuplicateAction: DuplicateActionError,
	})
	if err != nil {
		logger.Warn("batch import item failed",
			zap.String("source", game.Source),
			zap.String("external_id", game.ExternalID),
			zap.Error(err))
		return batchOutcomeFailed
	}
	switch result.DuplicateStatus {
	case DuplicateStatusAlreadyImported, DuplicateStatusPossible:
		return batchOutcomeSkipped
	default:
		return batchOutcomeCreated
	}
}
