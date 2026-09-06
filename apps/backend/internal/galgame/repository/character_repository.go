package repository

import (
	"context"
	"strings"

	"backend/internal/galgame/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CharacterRepository struct {
	db *gorm.DB
}

func NewCharacterRepository(db *gorm.DB) *CharacterRepository {
	return &CharacterRepository{db: db}
}

// CreateOrReuse leaves existing external metadata untouched even for concurrent imports.
func (r *CharacterRepository) CreateOrReuse(ctx context.Context, character *model.Character) (*model.Character, error) {
	query := r.db.WithContext(ctx)
	if character.Source != "" && character.SourceID != "" {
		query = query.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "source"}, {Name: "source_id"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{
				clause.Expr{SQL: "source <> '' AND source_id <> ''"},
			}},
			DoNothing: true,
		})
	}
	result := query.Create(character)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 0 {
		return character, nil
	}
	// A separate statement sees the winning insert after ON CONFLICT waits for its commit.
	var existing model.Character
	err := r.db.WithContext(ctx).Where("source = ? AND source_id = ?", character.Source, character.SourceID).First(&existing).Error
	return &existing, err
}

func (r *CharacterRepository) FindByID(ctx context.Context, id uint) (*model.Character, error) {
	var character model.Character
	err := r.db.WithContext(ctx).First(&character, id).Error
	return &character, err
}

func (r *CharacterRepository) Search(ctx context.Context, q string, page, pageSize int) ([]model.Character, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Character{})
	if q != "" {
		pattern := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q) + "%"
		query = query.Where("name ILIKE ? OR original_name ILIKE ? OR source_id ILIKE ?", pattern, pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var characters []model.Character
	err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&characters).Error
	return characters, total, err
}

func (r *CharacterRepository) Update(ctx context.Context, character *model.Character) error {
	result := r.db.WithContext(ctx).Model(character).Clauses(clause.Returning{}).
		Select("name", "original_name", "description", "image_url", "gender", "birthday", "blood_type", "height", "source", "source_id", "updated_at").
		Updates(character)
	if result.Error == nil && result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *CharacterRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Character{}, id)
	if result.Error == nil && result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *CharacterRepository) ListByGalgameID(ctx context.Context, galgameID uint) ([]model.GalgameCharacter, error) {
	var relations []model.GalgameCharacter
	err := r.db.WithContext(ctx).Preload("Character").Where("galgame_id = ?", galgameID).
		Order("sort_order ASC, id ASC").Find(&relations).Error
	return relations, err
}

func (r *CharacterRepository) FindRelation(ctx context.Context, galgameID, characterID uint) (*model.GalgameCharacter, error) {
	var relation model.GalgameCharacter
	err := r.db.WithContext(ctx).Preload("Character").
		Where("galgame_id = ? AND character_id = ?", galgameID, characterID).First(&relation).Error
	return &relation, err
}

func (r *CharacterRepository) Bind(ctx context.Context, relation *model.GalgameCharacter) error {
	return r.db.WithContext(ctx).Omit(clause.Associations).Create(relation).Error
}

func (r *CharacterRepository) UpdateRelation(ctx context.Context, relation *model.GalgameCharacter) error {
	result := r.db.WithContext(ctx).Model(relation).Clauses(clause.Returning{}).
		Where("galgame_id = ? AND character_id = ?", relation.GalgameID, relation.CharacterID).
		Select("role", "spoiler_level", "appearance_spoiler", "description", "spoiler_description", "sort_order", "updated_at").
		Updates(relation)
	if result.Error == nil && result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *CharacterRepository) Unbind(ctx context.Context, galgameID, characterID uint) error {
	result := r.db.WithContext(ctx).Where("galgame_id = ? AND character_id = ?", galgameID, characterID).
		Delete(&model.GalgameCharacter{})
	if result.Error == nil && result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}
