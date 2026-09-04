package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/user/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserPreferenceRepository struct {
	db *gorm.DB
}

func NewUserPreferenceRepository(db *gorm.DB) *UserPreferenceRepository {
	return &UserPreferenceRepository{db: db}
}

func (r *UserPreferenceRepository) FindByUserID(ctx context.Context, userID uint) (*model.UserPreference, error) {
	var preference model.UserPreference
	err := r.db.WithContext(ctx).First(&preference, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user preference: %w", err)
	}
	return &preference, nil
}

func (r *UserPreferenceRepository) Upsert(ctx context.Context, preference *model.UserPreference) error {
	preference.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"background_source", "background_asset_id", "background_preset",
			"background_opacity", "background_blur", "background_position",
			"background_size", "sensitive_cover_mode", "updated_at",
		}),
	}).Create(preference).Error
	if err != nil {
		return fmt.Errorf("upsert user preference: %w", err)
	}
	return nil
}
