package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	importerModel "backend/internal/importer/model"

	"gorm.io/gorm"
)

func (r *Repository) CreateJob(ctx context.Context, job *importerModel.ImportJob) error {
	if err := r.db.WithContext(ctx).Create(job).Error; err != nil {
		return fmt.Errorf("create import job: %w", err)
	}
	return nil
}

func (r *Repository) FindJob(ctx context.Context, id int64) (*importerModel.ImportJob, error) {
	var job importerModel.ImportJob
	err := r.db.WithContext(ctx).First(&job, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find import job: %w", err)
	}
	return &job, nil
}

// StartJob atomically moves a pending job into running state. It returns
// false when the job is missing or already started, which makes handler
// retries no-ops.
func (r *Repository) StartJob(ctx context.Context, id int64) (bool, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&importerModel.ImportJob{}).
		Where("id = ? AND status = ?", id, importerModel.ImportJobStatusPending).
		Updates(map[string]any{"status": importerModel.ImportJobStatusRunning, "started_at": now})
	if result.Error != nil {
		return false, fmt.Errorf("start import job: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *Repository) UpdateJobProgress(
	ctx context.Context,
	id int64,
	processed, created, skipped, failed int,
) error {
	err := r.db.WithContext(ctx).
		Model(&importerModel.ImportJob{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"processed_count": processed,
			"created_count":   created,
			"skipped_count":   skipped,
			"failed_count":    failed,
		}).Error
	if err != nil {
		return fmt.Errorf("update import job progress: %w", err)
	}
	return nil
}

func (r *Repository) SetJobTotal(ctx context.Context, id int64, total int) error {
	err := r.db.WithContext(ctx).
		Model(&importerModel.ImportJob{}).
		Where("id = ?", id).
		Update("total_count", total).Error
	if err != nil {
		return fmt.Errorf("set import job total: %w", err)
	}
	return nil
}

func (r *Repository) UpdateJobStats(ctx context.Context, id int64, stats any) error {
	encoded, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("encode import job stats: %w", err)
	}
	err = r.db.WithContext(ctx).
		Model(&importerModel.ImportJob{}).
		Where("id = ?", id).
		Update("stats", encoded).Error
	if err != nil {
		return fmt.Errorf("update import job stats: %w", err)
	}
	return nil
}

func (r *Repository) FinishJob(
	ctx context.Context,
	id int64,
	status int16,
	errorMessage string,
) error {
	now := time.Now()
	err := r.db.WithContext(ctx).
		Model(&importerModel.ImportJob{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        status,
			"error_message": errorMessage,
			"finished_at":   now,
		}).Error
	if err != nil {
		return fmt.Errorf("finish import job: %w", err)
	}
	return nil
}

func (r *Repository) ListJobs(
	ctx context.Context,
	status *int16,
	page, limit int,
) ([]importerModel.ImportJob, int64, error) {
	query := r.db.WithContext(ctx).Model(&importerModel.ImportJob{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count import jobs: %w", err)
	}
	var jobs []importerModel.ImportJob
	err := query.
		Order("id DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&jobs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list import jobs: %w", err)
	}
	return jobs, total, nil
}
