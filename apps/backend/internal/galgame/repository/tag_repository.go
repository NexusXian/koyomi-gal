package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/galgame/model"

	"gorm.io/gorm"
)

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) Create(ctx context.Context, tag *model.Tag) error {
	if err := r.db.WithContext(ctx).Create(tag).Error; err != nil {
		return fmt.Errorf("create tag: %w", err)
	}
	return nil
}

func (r *TagRepository) Update(ctx context.Context, tag *model.Tag) error {
	tag.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.Tag{}).
		Where("id = ?", tag.ID).
		Updates(map[string]any{
			"name":        tag.Name,
			"slug":        tag.Slug,
			"description": tag.Description,
			"updated_at":  tag.UpdatedAt,
		}).Error
	if err != nil {
		return fmt.Errorf("update tag: %w", err)
	}
	return nil
}

func (r *TagRepository) FindByID(ctx context.Context, id uint) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.WithContext(ctx).First(&tag, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find tag by id: %w", err)
	}
	return &tag, nil
}

func (r *TagRepository) FindBySlug(ctx context.Context, slug string) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&tag).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find tag by slug: %w", err)
	}
	return &tag, nil
}

func (r *TagRepository) FindByName(ctx context.Context, name string) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find tag by name: %w", err)
	}
	return &tag, nil
}

func (r *TagRepository) CountByIDs(ctx context.Context, ids []uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Tag{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count tags by ids: %w", err)
	}
	return count, nil
}

func (r *TagRepository) List(ctx context.Context) ([]model.Tag, error) {
	var tags []model.Tag
	if err := r.db.WithContext(ctx).Order("name, id").Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	return tags, nil
}
