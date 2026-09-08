package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/changelog/model"

	"gorm.io/gorm"
)

type ChangelogRepository struct {
	db *gorm.DB
}

func NewChangelogRepository(db *gorm.DB) *ChangelogRepository {
	return &ChangelogRepository{db: db}
}

func (r *ChangelogRepository) Create(ctx context.Context, value *model.SiteChangelog) error {
	if err := r.db.WithContext(ctx).Create(value).Error; err != nil {
		return fmt.Errorf("create site changelog: %w", err)
	}
	return nil
}

func (r *ChangelogRepository) Update(ctx context.Context, value *model.SiteChangelog) error {
	value.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Model(&model.SiteChangelog{}).Where("id = ?", value.ID).Updates(map[string]any{
		"version": value.Version, "title": value.Title, "items": value.Items,
		"published_at": value.PublishedAt, "updated_at": value.UpdatedAt,
	}).Error
	if err != nil {
		return fmt.Errorf("update site changelog: %w", err)
	}
	return nil
}

func (r *ChangelogRepository) Delete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&model.SiteChangelog{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete site changelog: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *ChangelogRepository) FindByID(ctx context.Context, id uint) (*model.SiteChangelog, error) {
	var value model.SiteChangelog
	err := r.db.WithContext(ctx).First(&value, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find site changelog: %w", err)
	}
	return &value, nil
}

func (r *ChangelogRepository) FindByVersion(ctx context.Context, version string) (*model.SiteChangelog, error) {
	var value model.SiteChangelog
	err := r.db.WithContext(ctx).Where("version = ?", version).First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find site changelog by version: %w", err)
	}
	return &value, nil
}

func (r *ChangelogRepository) List(ctx context.Context, page, limit int) ([]model.SiteChangelog, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.SiteChangelog{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count site changelogs: %w", err)
	}
	values := make([]model.SiteChangelog, 0)
	err := r.db.WithContext(ctx).
		Order("published_at DESC").Order("id DESC").
		Offset((page - 1) * limit).Limit(limit).Find(&values).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list site changelogs: %w", err)
	}
	return values, total, nil
}
