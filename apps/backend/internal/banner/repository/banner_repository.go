package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/banner/model"

	"gorm.io/gorm"
)

type BannerRepository struct {
	db *gorm.DB
}

func NewBannerRepository(db *gorm.DB) *BannerRepository {
	return &BannerRepository{db: db}
}

func (r *BannerRepository) Create(ctx context.Context, banner *model.Banner) error {
	if err := r.db.WithContext(ctx).Create(banner).Error; err != nil {
		return fmt.Errorf("create banner: %w", err)
	}
	return nil
}

func (r *BannerRepository) Update(ctx context.Context, banner *model.Banner) error {
	banner.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Model(&model.Banner{}).Where("id = ?", banner.ID).Updates(map[string]any{
		"title": banner.Title, "subtitle": banner.Subtitle, "image_url": banner.ImageURL,
		"link_type":  banner.LinkType,
		"link_value": banner.LinkValue, "sort_order": banner.SortOrder,
		"is_active": banner.IsActive, "start_at": banner.StartAt,
		"end_at": banner.EndAt, "updated_at": banner.UpdatedAt,
	}).Error
	if err != nil {
		return fmt.Errorf("update banner: %w", err)
	}
	return nil
}

func (r *BannerRepository) Delete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&model.Banner{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete banner: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *BannerRepository) FindByID(ctx context.Context, id uint) (*model.Banner, error) {
	var banner model.Banner
	err := r.db.WithContext(ctx).First(&banner, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find banner by id: %w", err)
	}
	return &banner, nil
}

func (r *BannerRepository) ListPublic(ctx context.Context, limit int) ([]model.Banner, error) {
	banners := make([]model.Banner, 0)
	query := r.db.WithContext(ctx).
		Where("is_active = TRUE").
		Where("start_at IS NULL OR start_at <= NOW()").
		Where("end_at IS NULL OR end_at >= NOW()").
		Order("sort_order DESC").Order("id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&banners).Error
	if err != nil {
		return nil, fmt.Errorf("list public banners: %w", err)
	}
	return banners, nil
}

func (r *BannerRepository) ListAdmin(ctx context.Context, page, limit int) ([]model.Banner, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.Banner{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count banners: %w", err)
	}
	banners := make([]model.Banner, 0)
	err := r.db.WithContext(ctx).Order("sort_order DESC").Order("id DESC").
		Offset((page - 1) * limit).Limit(limit).Find(&banners).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list admin banners: %w", err)
	}
	return banners, total, nil
}
