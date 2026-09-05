package repository

import (
	"context"
	"errors"
	"fmt"

	relationModel "backend/internal/relation/model"

	"gorm.io/gorm"
)

type RelationRepository struct {
	db *gorm.DB
}

func NewRelationRepository(db *gorm.DB) *RelationRepository {
	return &RelationRepository{db: db}
}

func (r *RelationRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *RelationRepository) Create(ctx context.Context, relation *relationModel.WorkRelation) error {
	if err := r.db.WithContext(ctx).Create(relation).Error; err != nil {
		return fmt.Errorf("create work relation: %w", err)
	}
	return nil
}

func (r *RelationRepository) FindByID(ctx context.Context, id uint) (*relationModel.WorkRelation, error) {
	var relation relationModel.WorkRelation
	err := r.db.WithContext(ctx).First(&relation, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find work relation by id: %w", err)
	}
	return &relation, nil
}

func (r *RelationRepository) Delete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&relationModel.WorkRelation{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete work relation: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// DeleteByWork physically removes every relation attached to the work on
// either side; used when a work is deleted.
func (r *RelationRepository) DeleteByWork(ctx context.Context, workType string, workID uint) error {
	err := r.db.WithContext(ctx).
		Where(
			"(source_type = ? AND source_id = ?) OR (target_type = ? AND target_id = ?)",
			workType, workID, workType, workID,
		).
		Delete(&relationModel.WorkRelation{}).Error
	if err != nil {
		return fmt.Errorf("delete work relations by work: %w", err)
	}
	return nil
}

// ListForNovel returns both directions of a novel's relations with the
// opposite side's work type and id resolved.
func (r *RelationRepository) ListForNovel(ctx context.Context, novelID uint) ([]relationModel.WorkRelation, error) {
	var relations []relationModel.WorkRelation
	err := r.db.WithContext(ctx).
		Where(
			"(source_type = ? AND source_id = ?) OR (target_type = ? AND target_id = ?)",
			relationModel.WorkTypeNovel, novelID, relationModel.WorkTypeNovel, novelID,
		).
		Order("work_relations.id").
		Find(&relations).Error
	if err != nil {
		return nil, fmt.Errorf("list work relations for novel: %w", err)
	}
	return relations, nil
}

// ListRelatedGalgamesForNovel returns published galgames related to the novel
// in either relation direction.
func (r *RelationRepository) ListRelatedGalgamesForNovel(
	ctx context.Context,
	novelID uint,
) ([]relationModel.RelatedWork, error) {
	works := make([]relationModel.RelatedWork, 0)
	err := r.db.WithContext(ctx).
		Table("work_relations AS r").
		Select(`r.id AS relation_id,
r.relation_type,
g.id AS work_id,
g.title,
g.original_title,
g.slug,
g.cover_url,
g.cover_sensitive,
g.age_rating`).
		Joins("JOIN galgames AS g ON g.status = 1 AND (("+
			"r.source_type = 'novel' AND r.source_id = ? AND r.target_type = 'galgame' AND g.id = r.target_id) OR ("+
			"r.target_type = 'novel' AND r.target_id = ? AND r.source_type = 'galgame' AND g.id = r.source_id))", novelID, novelID).
		Order("r.id").
		Scan(&works).Error
	if err != nil {
		return nil, fmt.Errorf("list related galgames for novel: %w", err)
	}
	return works, nil
}

// ListRelatedNovelsForGalgame returns published novels related to the galgame
// in either relation direction.
func (r *RelationRepository) ListRelatedNovelsForGalgame(
	ctx context.Context,
	galgameID uint,
) ([]relationModel.RelatedWork, error) {
	works := make([]relationModel.RelatedWork, 0)
	err := r.db.WithContext(ctx).
		Table("work_relations AS r").
		Select(`r.id AS relation_id,
r.relation_type,
n.id AS work_id,
n.title,
n.original_title,
n.slug,
n.cover_url,
n.is_cover_sensitive AS cover_sensitive,
n.age_rating`).
		Joins("JOIN novels AS n ON n.status = 1 AND n.deleted_at IS NULL AND (("+
			"r.source_type = 'galgame' AND r.source_id = ? AND r.target_type = 'novel' AND n.id = r.target_id) OR ("+
			"r.target_type = 'galgame' AND r.target_id = ? AND r.source_type = 'novel' AND n.id = r.source_id))", galgameID, galgameID).
		Order("r.id").
		Scan(&works).Error
	if err != nil {
		return nil, fmt.Errorf("list related novels for galgame: %w", err)
	}
	return works, nil
}
