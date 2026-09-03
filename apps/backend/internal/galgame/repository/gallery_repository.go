package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/galgame/model"

	"gorm.io/gorm"
)

type GalleryRepository struct {
	db *gorm.DB
}

func NewGalleryRepository(db *gorm.DB) *GalleryRepository {
	return &GalleryRepository{db: db}
}

func (r *GalleryRepository) Transaction(ctx context.Context, fn func(tx *GalleryRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&GalleryRepository{db: tx})
	})
}

// ListByGalgameID returns all gallery images ordered by sort_order, id with
// their assets preloaded in a single extra query (no N+1).
func (r *GalleryRepository) ListByGalgameID(ctx context.Context, galgameID uint) ([]model.GalleryImage, error) {
	var images []model.GalleryImage
	err := r.db.WithContext(ctx).
		Preload("Asset").
		Where("galgame_id = ?", galgameID).
		Order("sort_order ASC, id ASC").
		Find(&images).Error
	if err != nil {
		return nil, fmt.Errorf("list gallery images: %w", err)
	}
	return images, nil
}

func (r *GalleryRepository) FindByID(ctx context.Context, id uint) (*model.GalleryImage, error) {
	var image model.GalleryImage
	err := r.db.WithContext(ctx).
		Preload("Asset").
		First(&image, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find gallery image by id: %w", err)
	}
	return &image, nil
}

func (r *GalleryRepository) FindByGalgameIDAndAssetID(
	ctx context.Context,
	galgameID, assetID uint,
) (*model.GalleryImage, error) {
	var image model.GalleryImage
	err := r.db.WithContext(ctx).
		Where("galgame_id = ? AND asset_id = ?", galgameID, assetID).
		First(&image).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find gallery image by asset: %w", err)
	}
	return &image, nil
}

func (r *GalleryRepository) CountByGalgameID(ctx context.Context, galgameID uint) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.GalleryImage{}).
		Where("galgame_id = ?", galgameID).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("count gallery images: %w", err)
	}
	return total, nil
}

// MaxSortOrder returns the current maximum sort_order for a galgame; it is 0
// when the gallery is empty.
func (r *GalleryRepository) MaxSortOrder(ctx context.Context, galgameID uint) (int, error) {
	var max int
	err := r.db.WithContext(ctx).
		Model(&model.GalleryImage{}).
		Select("COALESCE(MAX(sort_order), 0)").
		Where("galgame_id = ?", galgameID).
		Scan(&max).Error
	if err != nil {
		return 0, fmt.Errorf("max gallery sort order: %w", err)
	}
	return max, nil
}

func (r *GalleryRepository) Create(ctx context.Context, image *model.GalleryImage) error {
	if err := r.db.WithContext(ctx).Create(image).Error; err != nil {
		return fmt.Errorf("create gallery image: %w", err)
	}
	return nil
}

func (r *GalleryRepository) Update(ctx context.Context, image *model.GalleryImage) error {
	err := r.db.WithContext(ctx).
		Model(&model.GalleryImage{}).
		Where("id = ?", image.ID).
		Updates(map[string]any{
			"title":       image.Title,
			"description": image.Description,
			"image_type":  image.ImageType,
			"is_spoiler":  image.IsSpoiler,
			"updated_at":  time.Now(),
		}).Error
	if err != nil {
		return fmt.Errorf("update gallery image: %w", err)
	}
	return nil
}

func (r *GalleryRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.GalleryImage{}, id).Error; err != nil {
		return fmt.Errorf("delete gallery image: %w", err)
	}
	return nil
}

// UpdateOrder rewrites sort_order to match the position of each id in ids.
func (r *GalleryRepository) UpdateOrder(ctx context.Context, galgameID uint, ids []uint) error {
	for index, id := range ids {
		err := r.db.WithContext(ctx).
			Model(&model.GalleryImage{}).
			Where("id = ? AND galgame_id = ?", id, galgameID).
			Updates(map[string]any{"sort_order": index, "updated_at": time.Now()}).Error
		if err != nil {
			return fmt.Errorf("update gallery order: %w", err)
		}
	}
	return nil
}
