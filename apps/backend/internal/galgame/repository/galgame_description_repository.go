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

var descriptionContentColumns = []string{
	"content", "source_type", "source_name", "source_url", "is_official", "updated_at",
}

var descriptionProvenanceColumns = []string{
	"source_external_id", "upstream_source",
}

// Upsert inserts or updates the row for (galgame_id, language) in a single
// statement so concurrent writers cannot create duplicates. Import provenance
// survives only when content and source type are unchanged, so re-saving a
// form keeps it while any real edit clears it.
func (r *GalgameDescriptionRepository) Upsert(
	ctx context.Context,
	description *model.GalgameDescription,
) error {
	updates := clause.AssignmentColumns(descriptionContentColumns)
	for _, column := range descriptionProvenanceColumns {
		updates = append(updates, clause.Assignment{
			Column: clause.Column{Name: column},
			Value: gorm.Expr(fmt.Sprintf(
				"CASE WHEN galgame_descriptions.content = EXCLUDED.content "+
					"AND galgame_descriptions.source_type = EXCLUDED.source_type "+
					"THEN galgame_descriptions.%[1]s ELSE EXCLUDED.%[1]s END",
				column,
			)),
		})
	}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "galgame_id"}, {Name: "language"}},
			DoUpdates: updates,
		}).
		Create(description).Error
	if err != nil {
		return fmt.Errorf("upsert galgame description: %w", err)
	}
	return nil
}

// UpsertWithProvenance writes every column including import provenance;
// used by automatic importers that own the row's provenance.
func (r *GalgameDescriptionRepository) UpsertWithProvenance(
	ctx context.Context,
	description *model.GalgameDescription,
) error {
	columns := append(append([]string{}, descriptionContentColumns...), descriptionProvenanceColumns...)
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "galgame_id"}, {Name: "language"}},
			DoUpdates: clause.AssignmentColumns(columns),
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
