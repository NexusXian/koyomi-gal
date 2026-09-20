package repository

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/galgame/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GalgameDescriptionRepository struct {
	db *gorm.DB
}

func NewGalgameDescriptionRepository(db *gorm.DB) *GalgameDescriptionRepository {
	return &GalgameDescriptionRepository{db: db}
}

func (r *GalgameDescriptionRepository) Transaction(
	ctx context.Context,
	fn func(tx *gorm.DB) error,
) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *GalgameDescriptionRepository) FindByGalgameID(
	ctx context.Context,
	galgameID uint,
) ([]model.GalgameDescription, error) {
	var descriptions []model.GalgameDescription
	err := r.db.WithContext(ctx).
		Where("galgame_id = ?", galgameID).
		Order("language").
		Find(&descriptions).Error
	if err != nil {
		return nil, fmt.Errorf("find galgame descriptions: %w", err)
	}
	return descriptions, nil
}

func (r *GalgameDescriptionRepository) FindByGalgameIDAndLanguage(
	ctx context.Context,
	galgameID uint,
	language string,
) (*model.GalgameDescription, error) {
	var description model.GalgameDescription
	err := r.db.WithContext(ctx).
		Where("galgame_id = ? AND language = ?", galgameID, language).
		First(&description).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find galgame description: %w", err)
	}
	return &description, nil
}

// Upsert inserts or updates the row for (galgame_id, language) in a single
// statement so concurrent writers cannot create duplicates.
func (r *GalgameDescriptionRepository) Upsert(
	ctx context.Context,
	description *model.GalgameDescription,
) error {
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "galgame_id"}, {Name: "language"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"content", "source_type", "source_name", "source_url", "is_official", "updated_at",
			}),
		}).
		Create(description).Error
	if err != nil {
		return fmt.Errorf("upsert galgame description: %w", err)
	}
	return nil
}

func (r *GalgameDescriptionRepository) Delete(
	ctx context.Context,
	galgameID uint,
	language string,
) error {
	err := r.db.WithContext(ctx).
		Where("galgame_id = ? AND language = ?", galgameID, language).
		Delete(&model.GalgameDescription{}).Error
	if err != nil {
		return fmt.Errorf("delete galgame description: %w", err)
	}
	return nil
}
