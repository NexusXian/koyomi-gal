package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/galgame/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserRelationSummary holds one user's rating, favorite, and play state for a
// single galgame. nil fields mean the relation does not exist.
type UserRelationSummary struct {
	Rating   *model.Rating
	Favorite *model.Favorite
	State    *model.UserState
}

type UserRelationRepository struct {
	db *gorm.DB
}

func NewUserRelationRepository(db *gorm.DB) *UserRelationRepository {
	return &UserRelationRepository{db: db}
}

func (r *UserRelationRepository) Transaction(ctx context.Context, fn func(tx *UserRelationRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&UserRelationRepository{db: tx})
	})
}

func (r *UserRelationRepository) FindRating(ctx context.Context, galgameID, userID uint) (*model.Rating, error) {
	var rating model.Rating
	err := r.db.WithContext(ctx).
		Where("galgame_id = ? AND user_id = ?", galgameID, userID).
		First(&rating).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find galgame rating: %w", err)
	}
	return &rating, nil
}

func (r *UserRelationRepository) UpsertRating(ctx context.Context, galgameID, userID uint, score int16) error {
	rating := model.Rating{GalgameID: galgameID, UserID: userID, Score: score}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "galgame_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"score":      score,
				"updated_at": time.Now(),
			}),
		}).
		Create(&rating).Error
	if err != nil {
		return fmt.Errorf("upsert galgame rating: %w", err)
	}
	return nil
}

// DeleteRating removes the rating row and reports whether one was deleted.
func (r *UserRelationRepository) DeleteRating(ctx context.Context, galgameID, userID uint) (bool, error) {
	result := r.db.WithContext(ctx).
		Where("galgame_id = ? AND user_id = ?", galgameID, userID).
		Delete(&model.Rating{})
	if result.Error != nil {
		return false, fmt.Errorf("delete galgame rating: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// RecalculateGalgameRating recomputes rating_average and rating_count from
// galgame_ratings; an empty rating set resets both to 0.
func (r *UserRelationRepository) RecalculateGalgameRating(ctx context.Context, galgameID uint) error {
	err := r.db.WithContext(ctx).Exec(`
UPDATE galgames AS g SET
    rating_average = COALESCE(agg.average, 0),
    rating_count = COALESCE(agg.count, 0)
FROM (
    SELECT AVG(score) AS average, COUNT(*) AS count
    FROM galgame_ratings
    WHERE galgame_id = ?
) AS agg
WHERE g.id = ?
`, galgameID, galgameID).Error
	if err != nil {
		return fmt.Errorf("recalculate galgame rating: %w", err)
	}
	return nil
}

func (r *UserRelationRepository) FindFavorite(ctx context.Context, galgameID, userID uint) (*model.Favorite, error) {
	var favorite model.Favorite
	err := r.db.WithContext(ctx).
		Where("galgame_id = ? AND user_id = ?", galgameID, userID).
		First(&favorite).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find galgame favorite: %w", err)
	}
	return &favorite, nil
}

// AddFavorite inserts the favorite and reports whether a new row was created;
// false means the relation already exists.
func (r *UserRelationRepository) AddFavorite(ctx context.Context, galgameID, userID uint) (bool, error) {
	favorite := model.Favorite{GalgameID: galgameID, UserID: userID}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&favorite).Error
	if err != nil {
		return false, fmt.Errorf("add galgame favorite: %w", err)
	}
	return favorite.ID != 0, nil
}

// RemoveFavorite deletes the favorite row and reports whether one was deleted.
func (r *UserRelationRepository) RemoveFavorite(ctx context.Context, galgameID, userID uint) (bool, error) {
	result := r.db.WithContext(ctx).
		Where("galgame_id = ? AND user_id = ?", galgameID, userID).
		Delete(&model.Favorite{})
	if result.Error != nil {
		return false, fmt.Errorf("remove galgame favorite: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *UserRelationRepository) IncrementFavoriteCount(ctx context.Context, galgameID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE galgames SET favorite_count = favorite_count + 1 WHERE id = ?",
		galgameID,
	).Error
	if err != nil {
		return fmt.Errorf("increment favorite count: %w", err)
	}
	return nil
}

func (r *UserRelationRepository) DecrementFavoriteCount(ctx context.Context, galgameID uint) error {
	err := r.db.WithContext(ctx).Exec(
		"UPDATE galgames SET favorite_count = GREATEST(favorite_count - 1, 0) WHERE id = ?",
		galgameID,
	).Error
	if err != nil {
		return fmt.Errorf("decrement favorite count: %w", err)
	}
	return nil
}

func (r *UserRelationRepository) FindUserState(ctx context.Context, galgameID, userID uint) (*model.UserState, error) {
	var state model.UserState
	err := r.db.WithContext(ctx).
		Where("galgame_id = ? AND user_id = ?", galgameID, userID).
		First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user galgame state: %w", err)
	}
	return &state, nil
}

func (r *UserRelationRepository) UpsertUserState(
	ctx context.Context,
	galgameID, userID uint,
	state int16,
	playTimeMinutes int64,
) error {
	userState := model.UserState{
		GalgameID:       galgameID,
		UserID:          userID,
		State:           state,
		PlayTimeMinutes: playTimeMinutes,
	}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "galgame_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"state":             state,
				"play_time_minutes": playTimeMinutes,
				"updated_at":        time.Now(),
			}),
		}).
		Create(&userState).Error
	if err != nil {
		return fmt.Errorf("upsert user galgame state: %w", err)
	}
	return nil
}

// DeleteUserState deletes the play state row and reports whether one existed.
func (r *UserRelationRepository) DeleteUserState(ctx context.Context, galgameID, userID uint) (bool, error) {
	result := r.db.WithContext(ctx).
		Where("galgame_id = ? AND user_id = ?", galgameID, userID).
		Delete(&model.UserState{})
	if result.Error != nil {
		return false, fmt.Errorf("delete user galgame state: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *UserRelationRepository) FindUserRelation(
	ctx context.Context,
	galgameID, userID uint,
) (*UserRelationSummary, error) {
	summary := &UserRelationSummary{}
	var err error
	if summary.Rating, err = r.FindRating(ctx, galgameID, userID); err != nil {
		return nil, err
	}
	if summary.Favorite, err = r.FindFavorite(ctx, galgameID, userID); err != nil {
		return nil, err
	}
	if summary.State, err = r.FindUserState(ctx, galgameID, userID); err != nil {
		return nil, err
	}
	return summary, nil
}
