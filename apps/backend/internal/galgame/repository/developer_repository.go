package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/galgame/model"

	"gorm.io/gorm"
)

type DeveloperRepository struct {
	db *gorm.DB
}

func NewDeveloperRepository(db *gorm.DB) *DeveloperRepository {
	return &DeveloperRepository{db: db}
}

func (r *DeveloperRepository) Create(ctx context.Context, developer *model.Developer) error {
	if err := r.db.WithContext(ctx).Create(developer).Error; err != nil {
		return fmt.Errorf("create developer: %w", err)
	}
	return nil
}

func (r *DeveloperRepository) Update(ctx context.Context, developer *model.Developer) error {
	developer.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).
		Model(&model.Developer{}).
		Where("id = ?", developer.ID).
		Updates(map[string]any{
			"name":          developer.Name,
			"original_name": developer.OriginalName,
			"slug":          developer.Slug,
			"description":   developer.Description,
			"logo_url":      developer.LogoURL,
			"website":       developer.Website,
			"updated_at":    developer.UpdatedAt,
		}).Error
	if err != nil {
		return fmt.Errorf("update developer: %w", err)
	}
	return nil
}

func (r *DeveloperRepository) FindByID(ctx context.Context, id uint) (*model.Developer, error) {
	var developer model.Developer
	err := r.db.WithContext(ctx).First(&developer, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find developer by id: %w", err)
	}
	return &developer, nil
}

func (r *DeveloperRepository) FindBySlug(ctx context.Context, slug string) (*model.Developer, error) {
	var developer model.Developer
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&developer).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find developer by slug: %w", err)
	}
	return &developer, nil
}

func (r *DeveloperRepository) List(ctx context.Context) ([]model.Developer, error) {
	var developers []model.Developer
	if err := r.db.WithContext(ctx).Order("name, id").Find(&developers).Error; err != nil {
		return nil, fmt.Errorf("list developers: %w", err)
	}
	return developers, nil
}
