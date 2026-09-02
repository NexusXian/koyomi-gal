package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/background/model"

	"gorm.io/gorm"
)

type BackgroundPresetRepository struct {
	db *gorm.DB
}

func NewBackgroundPresetRepository(db *gorm.DB) *BackgroundPresetRepository {
	return &BackgroundPresetRepository{db: db}
}

func (r *BackgroundPresetRepository) Create(ctx context.Context, preset *model.BackgroundPreset) error {
	if err := r.db.WithContext(ctx).Create(preset).Error; err != nil {
		return fmt.Errorf("create background preset: %w", err)
	}
	return nil
}

func (r *BackgroundPresetRepository) Update(ctx context.Context, preset *model.BackgroundPreset) error {
	preset.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Model(&model.BackgroundPreset{}).Where("id = ?", preset.ID).Updates(map[string]any{
		"name": preset.Name, "image_url": preset.ImageURL,
		"sort_order": preset.SortOrder, "is_active": preset.IsActive,
		"updated_at": preset.UpdatedAt,
	}).Error
	if err != nil {
		return fmt.Errorf("update background preset: %w", err)
	}
	return nil
}

func (r *BackgroundPresetRepository) Delete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&model.BackgroundPreset{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete background preset: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *BackgroundPresetRepository) FindByID(ctx context.Context, id uint) (*model.BackgroundPreset, error) {
	var preset model.BackgroundPreset
	err := r.db.WithContext(ctx).First(&preset, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find background preset by id: %w", err)
	}
	return &preset, nil
}

func (r *BackgroundPresetRepository) ListPublic(ctx context.Context) ([]model.BackgroundPreset, error) {
	presets := make([]model.BackgroundPreset, 0)
	err := r.db.WithContext(ctx).
		Where("is_active = TRUE").
		Order("sort_order DESC").Order("id ASC").
		Find(&presets).Error
	if err != nil {
		return nil, fmt.Errorf("list public background presets: %w", err)
	}
	return presets, nil
}

func (r *BackgroundPresetRepository) ListAdmin(ctx context.Context, page, limit int) ([]model.BackgroundPreset, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.BackgroundPreset{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count background presets: %w", err)
	}
	presets := make([]model.BackgroundPreset, 0)
	err := r.db.WithContext(ctx).Order("sort_order DESC").Order("id ASC").
		Offset((page - 1) * limit).Limit(limit).Find(&presets).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list admin background presets: %w", err)
	}
	return presets, total, nil
}
