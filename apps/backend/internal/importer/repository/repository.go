package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	galgameModel "backend/internal/galgame/model"
	importerModel "backend/internal/importer/model"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *Repository) FindExternalSource(
	ctx context.Context,
	source, externalID string,
) (*importerModel.GalgameExternalSource, error) {
	var mapping importerModel.GalgameExternalSource
	err := r.db.WithContext(ctx).
		Where("source = ? AND external_id = ?", source, externalID).
		First(&mapping).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find external source: %w", err)
	}
	return &mapping, nil
}

func (r *Repository) FindGalgame(ctx context.Context, id uint) (*galgameModel.Galgame, error) {
	var game galgameModel.Galgame
	err := r.db.WithContext(ctx).
		Preload("Developer").
		Preload("Aliases").
		Preload("Tags").
		First(&game, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find galgame: %w", err)
	}
	return &game, nil
}

func (r *Repository) FindDuplicateCandidates(
	ctx context.Context,
	releaseDate *time.Time,
	limit int,
) ([]galgameModel.Galgame, error) {
	query := r.db.WithContext(ctx).Model(&galgameModel.Galgame{})
	if releaseDate == nil {
		query = query.Where("release_date IS NULL")
	} else {
		query = query.Where("release_date = ?", releaseDate.Format("2006-01-02"))
	}
	var games []galgameModel.Galgame
	if err := query.Order("id DESC").Limit(limit).Find(&games).Error; err != nil {
		return nil, fmt.Errorf("find duplicate candidates: %w", err)
	}
	return games, nil
}
