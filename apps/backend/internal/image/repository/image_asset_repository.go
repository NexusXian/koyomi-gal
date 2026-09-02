package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/image/model"

	"gorm.io/gorm"
)

type AdminImageFilter struct {
	Page     int
	Limit    int
	Category string
	UserID   *uint
	Status   *int16
}

type ImageAssetRepository struct {
	db *gorm.DB
}

func NewImageAssetRepository(db *gorm.DB) *ImageAssetRepository {
	return &ImageAssetRepository{db: db}
}

func (r *ImageAssetRepository) Create(ctx context.Context, asset *model.ImageAsset) error {
	if err := r.db.WithContext(ctx).Create(asset).Error; err != nil {
		return fmt.Errorf("create image asset: %w", err)
	}
	return nil
}

func (r *ImageAssetRepository) FindByID(ctx context.Context, id uint) (*model.ImageAsset, error) {
	var asset model.ImageAsset
	err := r.db.WithContext(ctx).First(&asset, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find image asset by id: %w", err)
	}
	return &asset, nil
}

// MarkActive flips a pending asset to active and stores the verified object
// metadata. It reports whether the row was still pending.
func (r *ImageAssetRepository) MarkActive(
	ctx context.Context,
	id uint,
	size int64,
	width, height *int,
) (bool, error) {
	updates := map[string]any{
		"status":     model.ImageStatusActive,
		"size":       size,
		"width":      width,
		"height":     height,
		"updated_at": time.Now(),
	}
	result := r.db.WithContext(ctx).Model(&model.ImageAsset{}).
		Where("id = ? AND status = ?", id, model.ImageStatusPending).
		Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("mark image asset active: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// SoftDelete marks the asset deleted. It reports whether the row existed and
// was not already deleted.
func (r *ImageAssetRepository) SoftDelete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.ImageAsset{}).
		Where("id = ? AND status <> ?", id, model.ImageStatusDeleted).
		Updates(map[string]any{
			"status":     model.ImageStatusDeleted,
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return false, fmt.Errorf("soft delete image asset: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// HardDelete permanently removes the row; used by the pending-asset cleanup.
func (r *ImageAssetRepository) HardDelete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.ImageAsset{}, id).Error; err != nil {
		return fmt.Errorf("hard delete image asset: %w", err)
	}
	return nil
}

func (r *ImageAssetRepository) ListAdmin(
	ctx context.Context,
	filter AdminImageFilter,
) ([]model.ImageAsset, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.ImageAsset{})
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count image assets: %w", err)
	}
	assets := make([]model.ImageAsset, 0)
	err := query.Order("id DESC").
		Offset((filter.Page - 1) * filter.Limit).Limit(filter.Limit).
		Find(&assets).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list image assets: %w", err)
	}
	return assets, total, nil
}

// ListPendingBefore returns at most limit pending assets created before the
// given time, oldest first.
func (r *ImageAssetRepository) ListPendingBefore(
	ctx context.Context,
	before time.Time,
	limit int,
) ([]model.ImageAsset, error) {
	assets := make([]model.ImageAsset, 0)
	err := r.db.WithContext(ctx).
		Where("status = ? AND created_at < ?", model.ImageStatusPending, before).
		Order("created_at ASC").Limit(limit).
		Find(&assets).Error
	if err != nil {
		return nil, fmt.Errorf("list pending image assets: %w", err)
	}
	return assets, nil
}
