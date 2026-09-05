package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/novel/model"

	"gorm.io/gorm"
)

type VolumeRepository struct {
	db *gorm.DB
}

func NewVolumeRepository(db *gorm.DB) *VolumeRepository {
	return &VolumeRepository{db: db}
}

func (r *VolumeRepository) Transaction(ctx context.Context, fn func(tx *VolumeRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&VolumeRepository{db: tx})
	})
}

func (r *VolumeRepository) Create(ctx context.Context, volume *model.NovelVolume) error {
	if err := r.db.WithContext(ctx).Create(volume).Error; err != nil {
		return fmt.Errorf("create novel volume: %w", err)
	}
	return nil
}

func (r *VolumeRepository) Update(ctx context.Context, volume *model.NovelVolume) error {
	volume.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.NovelVolume{}).
		Where("id = ?", volume.ID).
		Updates(map[string]any{
			"volume_number":  volume.VolumeNumber,
			"title":          volume.Title,
			"original_title": volume.OriginalTitle,
			"cover_url":      volume.CoverURL,
			"isbn":           volume.ISBN,
			"release_date":   volume.ReleaseDate,
			"summary":        volume.Summary,
			"status":         volume.Status,
			"updated_at":     volume.UpdatedAt,
		}).Error
	if err != nil {
		return fmt.Errorf("update novel volume: %w", err)
	}
	return nil
}

func (r *VolumeRepository) UpdateReview(
	ctx context.Context,
	id uint,
	status int16,
	reviewedBy *uint,
	reviewedAt *time.Time,
	rejectReason string,
) error {
	err := r.db.WithContext(ctx).
		Model(&model.NovelVolume{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        status,
			"reviewed_by":   reviewedBy,
			"reviewed_at":   reviewedAt,
			"reject_reason": rejectReason,
			"updated_at":    time.Now(),
		}).Error
	if err != nil {
		return fmt.Errorf("update novel volume review: %w", err)
	}
	return nil
}

func (r *VolumeRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.NovelVolume{}, id).Error; err != nil {
		return fmt.Errorf("delete novel volume: %w", err)
	}
	return nil
}

func (r *VolumeRepository) FindByID(ctx context.Context, id uint) (*model.NovelVolume, error) {
	var volume model.NovelVolume
	err := r.db.WithContext(ctx).First(&volume, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find novel volume by id: %w", err)
	}
	return &volume, nil
}

// FindByIDAndNovel verifies the volume belongs to the novel, preventing
// cross-novel edits through nested routes.
func (r *VolumeRepository) FindByIDAndNovel(
	ctx context.Context,
	novelID, id uint,
) (*model.NovelVolume, error) {
	var volume model.NovelVolume
	err := r.db.WithContext(ctx).
		Where("id = ? AND novel_id = ?", id, novelID).
		First(&volume).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find novel volume by id and novel: %w", err)
	}
	return &volume, nil
}

// ListByNovel returns the novel's volumes ordered by sort_order; pass
// publishedOnly for the public endpoints.
func (r *VolumeRepository) ListByNovel(
	ctx context.Context,
	novelID uint,
	publishedOnly bool,
) ([]model.NovelVolume, error) {
	query := r.db.WithContext(ctx).Model(&model.NovelVolume{})
	if publishedOnly {
		query = query.Where("status = ?", model.NovelStatusPublished)
	}
	volumes := make([]model.NovelVolume, 0)
	err := query.
		Where("novel_id = ?", novelID).
		Order("sort_order").
		Order("id").
		Find(&volumes).Error
	if err != nil {
		return nil, fmt.Errorf("list novel volumes: %w", err)
	}
	return volumes, nil
}

type VolumeListOptions struct {
	Status  *int16
	NovelID *uint
	Page    int
	Limit   int
}

// ListAdmin returns volumes across all statuses with the parent novel title,
// newest first.
func (r *VolumeRepository) ListAdmin(
	ctx context.Context,
	options VolumeListOptions,
) ([]model.NovelVolume, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).
			Table("novel_volumes AS v").
			Where("v.deleted_at IS NULL")
		if options.Status != nil {
			query = query.Where("v.status = ?", *options.Status)
		}
		if options.NovelID != nil {
			query = query.Where("v.novel_id = ?", *options.NovelID)
		}
		return query
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count admin novel volumes: %w", err)
	}

	volumes := make([]model.NovelVolume, 0)
	err := base().
		Select("v.*, n.title AS novel_title").
		Joins("JOIN novels AS n ON n.id = v.novel_id AND n.deleted_at IS NULL").
		Order("v.id DESC").
		Offset((options.Page - 1) * options.Limit).
		Limit(options.Limit).
		Scan(&volumes).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list admin novel volumes: %w", err)
	}
	return volumes, total, nil
}

// MaxSortOrder returns the novel's current highest sort_order.
func (r *VolumeRepository) MaxSortOrder(ctx context.Context, novelID uint) (int, error) {
	var maxOrder *int
	err := r.db.WithContext(ctx).
		Model(&model.NovelVolume{}).
		Where("novel_id = ?", novelID).
		Select("MAX(sort_order)").
		Scan(&maxOrder).Error
	if err != nil {
		return 0, fmt.Errorf("max novel volume sort order: %w", err)
	}
	if maxOrder == nil {
		return -1, nil
	}
	return *maxOrder, nil
}

// UpdateOrder rewrites sort_order to match the position of each id in ids.
func (r *VolumeRepository) UpdateOrder(ctx context.Context, novelID uint, ids []uint) error {
	for index, id := range ids {
		err := r.db.WithContext(ctx).
			Model(&model.NovelVolume{}).
			Where("id = ? AND novel_id = ?", id, novelID).
			Updates(map[string]any{"sort_order": index, "updated_at": time.Now()}).Error
		if err != nil {
			return fmt.Errorf("update novel volume order: %w", err)
		}
	}
	return nil
}
