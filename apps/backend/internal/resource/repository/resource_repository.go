package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/resource/model"

	"gorm.io/gorm"
)

type ResourceRepository struct {
	db        *gorm.DB
	publicURL string
}

func NewResourceRepository(db *gorm.DB, publicURLs ...string) *ResourceRepository {
	publicURL := ""
	if len(publicURLs) > 0 {
		publicURL = strings.TrimRight(publicURLs[0], "/")
	}
	return &ResourceRepository{db: db, publicURL: publicURL}
}

func (r *ResourceRepository) Transaction(ctx context.Context, fn func(tx *ResourceRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&ResourceRepository{db: tx, publicURL: r.publicURL})
	})
}

func (r *ResourceRepository) Create(ctx context.Context, resource *model.Resource) error {
	if err := r.db.WithContext(ctx).Create(resource).Error; err != nil {
		return fmt.Errorf("create resource: %w", err)
	}
	return nil
}

func (r *ResourceRepository) Update(ctx context.Context, resource *model.Resource) error {
	resource.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.Resource{}).
		Where("id = ?", resource.ID).
		Updates(map[string]any{
			"title":         resource.Title,
			"resource_type": resource.Type,
			"description":   resource.Description,
			"status":        resource.Status,
			"updated_at":    resource.UpdatedAt,
		}).Error
	if err != nil {
		return fmt.Errorf("update resource: %w", err)
	}
	return nil
}

// Delete removes the resource row and reports whether one was deleted;
// resource_links are removed by the foreign key ON DELETE CASCADE.
func (r *ResourceRepository) Delete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&model.Resource{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete resource: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *ResourceRepository) FindByID(ctx context.Context, id uint) (*model.Resource, error) {
	return r.findByID(ctx, id, false)
}

func (r *ResourceRepository) FindPublishedByID(ctx context.Context, id uint) (*model.Resource, error) {
	return r.findByID(ctx, id, true)
}

func (r *ResourceRepository) findByID(ctx context.Context, id uint, publishedOnly bool) (*model.Resource, error) {
	var resource model.Resource
	query := r.withLinks(r.withUploader(r.db.WithContext(ctx).Model(&model.Resource{})))
	if publishedOnly {
		query = query.Where("resources.status = ?", model.ResourceStatusPublished)
	}
	err := query.First(&resource, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find resource by id: %w", err)
	}
	return &resource, nil
}

func (r *ResourceRepository) ListPublishedByGalgame(
	ctx context.Context,
	galgameID uint,
	page, limit int,
) ([]model.Resource, int64, error) {
	base := func() *gorm.DB {
		return r.db.WithContext(ctx).
			Model(&model.Resource{}).
			Where("resources.galgame_id = ? AND resources.status = ?", galgameID, model.ResourceStatusPublished)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count published resources: %w", err)
	}

	var resources []model.Resource
	err := r.withLinks(r.withUploader(base())).
		Order("resources.id DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&resources).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list published resources: %w", err)
	}
	return resources, total, nil
}

type ResourceListOptions struct {
	Status *int16
	Page   int
	Limit  int
}

// ListAdmin returns resources across all statuses with an optional status
// filter, paginated and newest first.
func (r *ResourceRepository) ListAdmin(
	ctx context.Context,
	options ResourceListOptions,
) ([]model.Resource, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&model.Resource{})
		if options.Status != nil {
			query = query.Where("resources.status = ?", *options.Status)
		}
		return query
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count resources: %w", err)
	}

	var resources []model.Resource
	err := r.withLinks(r.withUploader(base())).
		Order("resources.id DESC").
		Offset((options.Page - 1) * options.Limit).
		Limit(options.Limit).
		Find(&resources).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list admin resources: %w", err)
	}
	return resources, total, nil
}

func (r *ResourceRepository) CreateLinks(ctx context.Context, resourceID uint, urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	links := make([]model.ResourceLink, 0, len(urls))
	for _, url := range urls {
		links = append(links, model.ResourceLink{ResourceID: resourceID, URL: url})
	}
	if err := r.db.WithContext(ctx).Create(&links).Error; err != nil {
		return fmt.Errorf("create resource links: %w", err)
	}
	return nil
}

func (r *ResourceRepository) ReplaceLinks(ctx context.Context, resourceID uint, urls []string) error {
	if err := r.db.WithContext(ctx).
		Where("resource_id = ?", resourceID).
		Delete(&model.ResourceLink{}).Error; err != nil {
		return fmt.Errorf("delete resource links: %w", err)
	}
	return r.CreateLinks(ctx, resourceID, urls)
}

func (r *ResourceRepository) IncrementResourceCount(ctx context.Context, galgameID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE galgames SET resource_count = resource_count + 1 WHERE id = ?",
		galgameID,
	).Error
	if err != nil {
		return fmt.Errorf("increment resource count: %w", err)
	}
	return nil
}

func (r *ResourceRepository) DecrementResourceCount(ctx context.Context, galgameID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE galgames SET resource_count = GREATEST(resource_count - 1, 0) WHERE id = ?",
		galgameID,
	).Error
	if err != nil {
		return fmt.Errorf("decrement resource count: %w", err)
	}
	return nil
}

func (r *ResourceRepository) withLinks(query *gorm.DB) *gorm.DB {
	return query.Preload("Links", func(db *gorm.DB) *gorm.DB { return db.Order("resource_links.id") })
}

func (r *ResourceRepository) withUploader(query *gorm.DB) *gorm.DB {
	return query.
		Select(`resources.*,
COALESCE(users.username, '') AS uploader_name,
COALESCE(NULLIF(user_profiles.display_name, ''), users.username, '') AS uploader_display_name,
CASE WHEN avatar_assets.object_key IS NOT NULL THEN CAST(? AS text) || '/' || avatar_assets.object_key ELSE COALESCE(users.avatar, '') END AS uploader_avatar`, r.publicURL).
		Joins("LEFT JOIN users ON users.id = resources.uploader_id").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
		Joins("LEFT JOIN image_assets AS avatar_assets ON avatar_assets.id = user_profiles.avatar_asset_id AND avatar_assets.user_id = users.id AND avatar_assets.category = 'avatars' AND avatar_assets.status = 1")
}
