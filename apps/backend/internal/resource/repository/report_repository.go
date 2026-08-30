package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/resource/model"

	"gorm.io/gorm"
)

type ReportListOptions struct {
	Status *int16
	Page   int
	Limit  int
}

type ReportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(ctx context.Context, report *model.ResourceReport) error {
	if err := r.db.WithContext(ctx).Create(report).Error; err != nil {
		return fmt.Errorf("create resource report: %w", err)
	}
	return nil
}

func (r *ReportRepository) FindByID(ctx context.Context, id uint) (*model.ResourceReport, error) {
	var report model.ResourceReport
	err := r.db.WithContext(ctx).Preload("Resource").First(&report, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find resource report by id: %w", err)
	}
	return &report, nil
}

func (r *ReportRepository) List(
	ctx context.Context,
	options ReportListOptions,
) ([]model.ResourceReport, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&model.ResourceReport{})
		if options.Status != nil {
			query = query.Where("status = ?", *options.Status)
		}
		return query
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count resource reports: %w", err)
	}

	var reports []model.ResourceReport
	err := base().
		Preload("Resource").
		Order("resource_reports.id DESC").
		Offset((options.Page - 1) * options.Limit).
		Limit(options.Limit).
		Find(&reports).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list resource reports: %w", err)
	}
	return reports, total, nil
}

// Update persists handle results: status, handled_by, and handled_at.
func (r *ReportRepository) Update(ctx context.Context, report *model.ResourceReport) error {
	report.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.ResourceReport{}).
		Where("id = ?", report.ID).
		Updates(map[string]any{
			"status":     report.Status,
			"handled_by": report.HandledBy,
			"handled_at": report.HandledAt,
			"updated_at": report.UpdatedAt,
		}).Error
	if err != nil {
		return fmt.Errorf("update resource report: %w", err)
	}
	return nil
}
