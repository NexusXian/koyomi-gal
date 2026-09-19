package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	db            *gorm.DB
	avatarBaseURL string
}

func NewUserRelationRepository(db *gorm.DB, avatarBaseURL ...string) *UserRelationRepository {
	baseURL := ""
	if len(avatarBaseURL) > 0 {
		baseURL = strings.TrimRight(avatarBaseURL[0], "/")
	}
	return &UserRelationRepository{db: db, avatarBaseURL: baseURL}
}

func (r *UserRelationRepository) Transaction(ctx context.Context, fn func(tx *UserRelationRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&UserRelationRepository{db: tx, avatarBaseURL: r.avatarBaseURL})
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

func (r *UserRelationRepository) UpsertDetailedRating(ctx context.Context, rating *model.Rating) error {
	now := time.Now()
	rating.UpdatedAt = now
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "galgame_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"score": rating.Score, "visual": rating.Visual, "story": rating.Story,
				"music": rating.Music, "character": rating.Character, "branch": rating.Branch,
				"system": rating.System, "voice": rating.Voice, "replay": rating.Replay,
				"recommendation": rating.Recommendation, "review_text": rating.ReviewText,
				"spoiler_level": rating.SpoilerLevel, "updated_at": now,
			}),
		}).Create(rating).Error
	if err != nil {
		return fmt.Errorf("upsert detailed galgame rating: %w", err)
	}
	return nil
}

type RatingListOptions struct {
	Page     int
	PageSize int
	Sort     string
	ViewerID uint
}

func (r *UserRelationRepository) ratingViewQuery(ctx context.Context, viewerID uint) *gorm.DB {
	return r.db.WithContext(ctx).Table("galgame_ratings AS ratings").
		Select(`ratings.*,
users.username,
COALESCE(NULLIF(user_profiles.display_name, ''), users.username) AS display_name,
CASE WHEN avatar_assets.object_key IS NOT NULL THEN CAST(? AS text) || '/' || avatar_assets.object_key ELSE users.avatar END AS avatar_url,
user_galgames.state AS play_status,
EXISTS (SELECT 1 FROM galgame_rating_likes viewer_likes WHERE viewer_likes.rating_id = ratings.id AND viewer_likes.user_id = ?) AS liked`, r.avatarBaseURL, viewerID).
		Joins("JOIN users ON users.id = ratings.user_id").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
		Joins("LEFT JOIN image_assets AS avatar_assets ON avatar_assets.id = users.avatar_asset_id AND avatar_assets.user_id = users.id AND avatar_assets.status = 1").
		Joins("LEFT JOIN user_galgames ON user_galgames.galgame_id = ratings.galgame_id AND user_galgames.user_id = ratings.user_id").
		Joins("LEFT JOIN user_privacy_settings ON user_privacy_settings.user_id = ratings.user_id")
}

func (r *UserRelationRepository) FindRatingView(ctx context.Context, galgameID, userID, viewerID uint) (*model.RatingView, error) {
	var rating model.RatingView
	err := r.ratingViewQuery(ctx, viewerID).
		Where("ratings.galgame_id = ? AND ratings.user_id = ?", galgameID, userID).
		First(&rating).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find galgame rating view: %w", err)
	}
	return &rating, nil
}

func (r *UserRelationRepository) FindPublishedRatingByID(ctx context.Context, ratingID, viewerID uint) (*model.RatingView, error) {
	var rating model.RatingView
	err := r.ratingViewQuery(ctx, viewerID).
		Joins("JOIN galgames ON galgames.id = ratings.galgame_id AND galgames.status = 1").
		Where("ratings.id = ?", ratingID).First(&rating).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find published galgame rating: %w", err)
	}
	return &rating, nil
}

func (r *UserRelationRepository) LockPublishedRating(ctx context.Context, ratingID uint) (bool, error) {
	var row struct{ ID uint }
	err := r.db.WithContext(ctx).Raw(`
SELECT ratings.id
FROM galgame_ratings AS ratings
JOIN galgames ON galgames.id = ratings.galgame_id AND galgames.status = 1
WHERE ratings.id = ?
FOR UPDATE OF ratings`, ratingID).Scan(&row).Error
	if err != nil {
		return false, fmt.Errorf("lock published galgame rating: %w", err)
	}
	return row.ID != 0, nil
}

func (r *UserRelationRepository) ListRatings(ctx context.Context, galgameID uint, options RatingListOptions) ([]model.RatingView, int64, error) {
	// Authors always see their own rating; other users' ratings require
	// show_ratings privacy to be enabled (missing privacy rows default to visible).
	base := r.db.WithContext(ctx).Table("galgame_ratings").
		Joins("LEFT JOIN user_privacy_settings ON user_privacy_settings.user_id = galgame_ratings.user_id").
		Where("galgame_id = ?", galgameID).
		Where("(galgame_ratings.user_id = ? OR COALESCE(user_privacy_settings.show_ratings, TRUE))", options.ViewerID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count galgame ratings: %w", err)
	}
	query := r.ratingViewQuery(ctx, options.ViewerID).
		Where("ratings.galgame_id = ?", galgameID).
		Where("(ratings.user_id = ? OR COALESCE(user_privacy_settings.show_ratings, TRUE))", options.ViewerID)
	switch options.Sort {
	case "highest":
		query = query.Order("ratings.score DESC").Order("ratings.updated_at DESC").Order("ratings.id DESC")
	case "lowest":
		query = query.Order("ratings.score ASC").Order("ratings.updated_at DESC").Order("ratings.id DESC")
	case "popular":
		query = query.Order("ratings.like_count DESC").Order("ratings.updated_at DESC").Order("ratings.id DESC")
	default:
		query = query.Order("ratings.updated_at DESC").Order("ratings.id DESC")
	}
	items := make([]model.RatingView, 0)
	if err := query.Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Scan(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list galgame ratings: %w", err)
	}
	return items, total, nil
}

func (r *UserRelationRepository) RatingSummary(ctx context.Context, galgameID uint) (*model.RatingSummary, error) {
	var row struct {
		Count            int64
		Overall          *float64
		VisualAverage    *float64
		VisualCount      int64
		StoryAverage     *float64
		StoryCount       int64
		MusicAverage     *float64
		MusicCount       int64
		CharacterAverage *float64
		CharacterCount   int64
		BranchAverage    *float64
		BranchCount      int64
		SystemAverage    *float64
		SystemCount      int64
		VoiceAverage     *float64
		VoiceCount       int64
		ReplayAverage    *float64
		ReplayCount      int64
	}
	err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(*) AS count, AVG(score)::float8 AS overall,
AVG(visual)::float8 AS visual_average, COUNT(visual) AS visual_count,
AVG(story)::float8 AS story_average, COUNT(story) AS story_count,
AVG(music)::float8 AS music_average, COUNT(music) AS music_count,
AVG(character)::float8 AS character_average, COUNT(character) AS character_count,
AVG(branch)::float8 AS branch_average, COUNT(branch) AS branch_count,
AVG(system)::float8 AS system_average, COUNT(system) AS system_count,
AVG(voice)::float8 AS voice_average, COUNT(voice) AS voice_count,
AVG(replay)::float8 AS replay_average, COUNT(replay) AS replay_count
FROM galgame_ratings WHERE galgame_id = ?`, galgameID).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("aggregate galgame rating summary: %w", err)
	}
	return &model.RatingSummary{
		Count: row.Count, Overall: row.Overall,
		Visual:    model.RatingDimensionSummary{Average: row.VisualAverage, Count: row.VisualCount},
		Story:     model.RatingDimensionSummary{Average: row.StoryAverage, Count: row.StoryCount},
		Music:     model.RatingDimensionSummary{Average: row.MusicAverage, Count: row.MusicCount},
		Character: model.RatingDimensionSummary{Average: row.CharacterAverage, Count: row.CharacterCount},
		Branch:    model.RatingDimensionSummary{Average: row.BranchAverage, Count: row.BranchCount},
		System:    model.RatingDimensionSummary{Average: row.SystemAverage, Count: row.SystemCount},
		Voice:     model.RatingDimensionSummary{Average: row.VoiceAverage, Count: row.VoiceCount},
		Replay:    model.RatingDimensionSummary{Average: row.ReplayAverage, Count: row.ReplayCount},
	}, nil
}

func (r *UserRelationRepository) AddRatingLike(ctx context.Context, ratingID, userID uint) (bool, error) {
	result := r.db.WithContext(ctx).Exec(`
INSERT INTO galgame_rating_likes (rating_id, user_id)
VALUES (?, ?) ON CONFLICT (rating_id, user_id) DO NOTHING`, ratingID, userID)
	if result.Error != nil {
		return false, fmt.Errorf("add galgame rating like: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *UserRelationRepository) RemoveRatingLike(ctx context.Context, ratingID, userID uint) (bool, error) {
	result := r.db.WithContext(ctx).Exec(
		"DELETE FROM galgame_rating_likes WHERE rating_id = ? AND user_id = ?", ratingID, userID,
	)
	if result.Error != nil {
		return false, fmt.Errorf("remove galgame rating like: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *UserRelationRepository) IncrementRatingLikeCount(ctx context.Context, ratingID uint) error {
	if err := r.db.WithContext(ctx).Exec(
		"UPDATE galgame_ratings SET like_count = like_count + 1 WHERE id = ?", ratingID,
	).Error; err != nil {
		return fmt.Errorf("increment galgame rating like count: %w", err)
	}
	return nil
}

func (r *UserRelationRepository) DecrementRatingLikeCount(ctx context.Context, ratingID uint) error {
	if err := r.db.WithContext(ctx).Exec(
		"UPDATE galgame_ratings SET like_count = GREATEST(like_count - 1, 0) WHERE id = ?", ratingID,
	).Error; err != nil {
		return fmt.Errorf("decrement galgame rating like count: %w", err)
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
