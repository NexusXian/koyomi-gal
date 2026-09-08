package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/apprelease/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AppReleaseRepository struct {
	db *gorm.DB
}

func NewAppReleaseRepository(db *gorm.DB) *AppReleaseRepository {
	return &AppReleaseRepository{db: db}
}

func (r *AppReleaseRepository) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *AppReleaseRepository) Create(ctx context.Context, value *model.AppRelease) error {
	if err := r.db.WithContext(ctx).Create(value).Error; err != nil {
		return fmt.Errorf("create app release: %w", err)
	}
	return nil
}

func (r *AppReleaseRepository) Update(ctx context.Context, value *model.AppRelease) error {
	value.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Model(&model.AppRelease{}).Where("id = ?", value.ID).Updates(map[string]any{
		"platform": value.Platform, "version_name": value.VersionName, "version_code": value.VersionCode,
		"title": value.Title, "changelog": value.Changelog, "download_url": value.DownloadURL,
		"file_size": value.FileSize, "file_sha256": value.FileSHA256,
		"minimum_version_code": value.MinimumVersionCode, "force_update": value.ForceUpdate,
		"status": value.Status, "published_at": value.PublishedAt,
		"announcement_id": value.AnnouncementID, "announcement_managed": value.AnnouncementManaged,
		"updated_at": value.UpdatedAt,
	}).Error
	if err != nil {
		return fmt.Errorf("update app release: %w", err)
	}
	return nil
}

func (r *AppReleaseRepository) Delete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&model.AppRelease{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete app release: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *AppReleaseRepository) FindByID(ctx context.Context, id uint) (*model.AppRelease, error) {
	return r.findByID(ctx, id, false)
}

func (r *AppReleaseRepository) FindByIDForUpdate(ctx context.Context, id uint) (*model.AppRelease, error) {
	return r.findByID(ctx, id, true)
}

func (r *AppReleaseRepository) findByID(ctx context.Context, id uint, lock bool) (*model.AppRelease, error) {
	var value model.AppRelease
	query := r.db.WithContext(ctx)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.First(&value, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find app release: %w", err)
	}
	return &value, nil
}

func (r *AppReleaseRepository) FindLatestPublished(ctx context.Context, platform string, clientVersionCode int64) (*model.AppRelease, error) {
	var value model.AppRelease
	err := r.db.WithContext(ctx).
		Where("platform = ? AND status = ?", platform, model.StatusPublished).
		Where("published_at <= NOW()").
		Where("version_code > ?", clientVersionCode).
		Order("version_code DESC").First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find latest app release: %w", err)
	}
	return &value, nil
}

func (r *AppReleaseRepository) ListAdmin(ctx context.Context, page, limit int) ([]model.AppRelease, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.AppRelease{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count app releases: %w", err)
	}
	values := make([]model.AppRelease, 0)
	err := r.db.WithContext(ctx).Order("updated_at DESC").Order("id DESC").
		Offset((page - 1) * limit).Limit(limit).Find(&values).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list app releases: %w", err)
	}
	return values, total, nil
}
