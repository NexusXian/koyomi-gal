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

// GalleryReviewRow is a gallery image joined with its galgame title and the
// usernames of its submitter and reviewer for the admin review queue.
type GalleryReviewRow struct {
	model.GalleryImage
	GalgameTitle       string
	GalgameSlug        string
	CreatedByUsername  string
	ReviewedByUsername string
}

// GalleryReviewFilter scopes the review queue listing.
type GalleryReviewFilter struct {
	Status     *int16
	GalgameID  uint
	SourceType *int16
	Page       int
	Limit      int
}

// ListByGalgameID returns gallery images ordered by sort_order, id with
// their assets preloaded in a single extra query (no N+1). Passing statuses
// filters on review state; without arguments every status is returned.
func (r *GalleryRepository) ListByGalgameID(ctx context.Context, galgameID uint, statuses ...int16) ([]model.GalleryImage, error) {
	var images []model.GalleryImage
	query := r.db.WithContext(ctx).
		Preload("Asset").
		Where("galgame_id = ?", galgameID)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	err := query.
		Order("sort_order ASC, id ASC").
		Find(&images).Error
	if err != nil {
		return nil, fmt.Errorf("list gallery images: %w", err)
	}
	return images, nil
}

// ListForReview returns the paginated cross-galgame review queue, newest
// submissions first, plus the total row count for pagination.
func (r *GalleryRepository) ListForReview(ctx context.Context, filter GalleryReviewFilter) ([]GalleryReviewRow, int64, error) {
	applyFilters := func(query *gorm.DB) *gorm.DB {
		if filter.Status != nil {
			query = query.Where("galgame_gallery_images.status = ?", *filter.Status)
		}
		if filter.GalgameID != 0 {
			query = query.Where("galgame_gallery_images.galgame_id = ?", filter.GalgameID)
		}
		if filter.SourceType != nil {
			query = query.Where("galgame_gallery_images.source_type = ?", *filter.SourceType)
		}
		return query
	}

	var total int64
	err := applyFilters(r.db.WithContext(ctx).Model(&model.GalleryImage{})).
		Joins("LEFT JOIN galgames g ON g.id = galgame_gallery_images.galgame_id").
		Joins("LEFT JOIN users cu ON cu.id = galgame_gallery_images.created_by").
		Joins("LEFT JOIN users ru ON ru.id = galgame_gallery_images.reviewed_by").
		Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("count gallery review queue: %w", err)
	}

	page, limit := filter.Page, filter.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	var rows []GalleryReviewRow
	err = applyFilters(r.db.WithContext(ctx).Model(&model.GalleryImage{})).
		Select(`galgame_gallery_images.*, g.title AS galgame_title, g.slug AS galgame_slug, ` +
			`cu.username AS created_by_username, ru.username AS reviewed_by_username`).
		Joins("LEFT JOIN galgames g ON g.id = galgame_gallery_images.galgame_id").
		Joins("LEFT JOIN users cu ON cu.id = galgame_gallery_images.created_by").
		Joins("LEFT JOIN users ru ON ru.id = galgame_gallery_images.reviewed_by").
		Preload("Asset").
		Order("galgame_gallery_images.created_at DESC, galgame_gallery_images.id DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list gallery review queue: %w", err)
	}
	return rows, total, nil
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

func (r *GalleryRepository) FindByIDs(ctx context.Context, ids []uint) ([]model.GalleryImage, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var images []model.GalleryImage
	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&images).Error
	if err != nil {
		return nil, fmt.Errorf("find gallery images by ids: %w", err)
	}
	return images, nil
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

func (r *GalleryRepository) FindByGalgameIDAndExternalURL(
	ctx context.Context,
	galgameID uint,
	externalURL string,
) (*model.GalleryImage, error) {
	var image model.GalleryImage
	err := r.db.WithContext(ctx).
		Where("galgame_id = ? AND external_url = ?", galgameID, externalURL).
		First(&image).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find gallery image by external url: %w", err)
	}
	return &image, nil
}

// CountActiveByGalgameID counts pending + published images; rejected entries
// no longer consume gallery slots.
func (r *GalleryRepository) CountActiveByGalgameID(ctx context.Context, galgameID uint) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.GalleryImage{}).
		Where("galgame_id = ? AND status IN ?", galgameID,
			[]int16{model.GalleryImageStatusPending, model.GalleryImageStatusPublished}).
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

// ReviewByIDs rewrites the review state of the given images and stamps who
// reviewed them and when.
func (r *GalleryRepository) ReviewByIDs(
	ctx context.Context,
	ids []uint,
	status int16,
	reviewedBy uint,
	reviewedAt time.Time,
	rejectReason string,
) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&model.GalleryImage{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"status":        status,
			"reviewed_by":   reviewedBy,
			"reviewed_at":   reviewedAt,
			"reject_reason": rejectReason,
			"updated_at":    time.Now(),
		})
	if result.Error != nil {
		return 0, fmt.Errorf("review gallery images: %w", result.Error)
	}
	return result.RowsAffected, nil
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
