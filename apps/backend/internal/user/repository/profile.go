package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	imageModel "backend/internal/image/model"
	"backend/internal/user/model"

	"gorm.io/gorm"
)

type UserProfileRepository struct {
	db        *gorm.DB
	publicURL string
}

func NewUserProfileRepository(db *gorm.DB, publicURL string) *UserProfileRepository {
	return &UserProfileRepository{db: db, publicURL: strings.TrimRight(publicURL, "/")}
}

func (r *UserProfileRepository) FindByUsername(ctx context.Context, username string) (*model.PublicProfileRecord, error) {
	return r.find(ctx, "LOWER(users.username) = LOWER(?)", username)
}

func (r *UserProfileRepository) FindByUserID(ctx context.Context, userID uint) (*model.PublicProfileRecord, error) {
	return r.find(ctx, "users.id = ?", userID)
}

func (r *UserProfileRepository) find(ctx context.Context, where string, value any) (*model.PublicProfileRecord, error) {
	var record model.PublicProfileRecord
	err := r.db.WithContext(ctx).Table("users").
		Select(`users.id, users.username,
COALESCE(NULLIF(user_profiles.display_name, ''), users.username) AS display_name,
user_profiles.avatar_asset_id, user_profiles.banner_asset_id,
CASE WHEN avatar_assets.object_key IS NOT NULL THEN CAST(? AS text) || '/' || avatar_assets.object_key ELSE users.avatar END AS avatar_url,
CASE WHEN banner_assets.object_key IS NOT NULL THEN CAST(? AS text) || '/' || banner_assets.object_key ELSE '' END AS banner_url,
COALESCE(user_profiles.bio, '') AS bio, COALESCE(user_profiles.gender, '') AS gender,
COALESCE(user_profiles.location, '') AS location, user_profiles.birthday,
COALESCE(user_profiles.website_url, '') AS website_url,
COALESCE(users.created_at, user_profiles.created_at, NOW()) AS registered_at,
COALESCE(user_privacy_settings.profile_visibility, 'public') AS profile_visibility,
COALESCE(user_privacy_settings.show_location, FALSE) AS show_location,
COALESCE(user_privacy_settings.show_birthday, FALSE) AS show_birthday,
COALESCE(user_privacy_settings.show_posts, TRUE) AS show_posts,
COALESCE(user_privacy_settings.show_comments, TRUE) AS show_comments,
COALESCE(user_privacy_settings.show_ratings, TRUE) AS show_ratings,
COALESCE(user_privacy_settings.show_favorites, FALSE) AS show_favorites,
COALESCE(user_privacy_settings.show_activity, TRUE) AS show_activity`, r.publicURL, r.publicURL).
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
		Joins("LEFT JOIN user_privacy_settings ON user_privacy_settings.user_id = users.id").
		Joins(fmt.Sprintf("LEFT JOIN image_assets AS avatar_assets ON avatar_assets.id = user_profiles.avatar_asset_id AND avatar_assets.user_id = users.id AND avatar_assets.category = 'avatars' AND avatar_assets.status = %d", imageModel.ImageStatusActive)).
		Joins(fmt.Sprintf("LEFT JOIN image_assets AS banner_assets ON banner_assets.id = user_profiles.banner_asset_id AND banner_assets.user_id = users.id AND banner_assets.category = 'profile-banners' AND banner_assets.status = %d", imageModel.ImageStatusActive)).
		Where(where, value).Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find public profile: %w", err)
	}
	return &record, nil
}

func (r *UserProfileRepository) Counts(ctx context.Context, userID uint) (model.ProfileCounts, error) {
	var counts model.ProfileCounts
	err := r.db.WithContext(ctx).Raw(`SELECT
(SELECT COUNT(*) FROM posts WHERE author_id = ?) AS posts,
(SELECT COUNT(*) FROM comments WHERE author_id = ?) AS comments,
(SELECT COUNT(*) FROM galgame_ratings r JOIN galgames g ON g.id = r.galgame_id WHERE r.user_id = ? AND g.status = 1) AS ratings,
(SELECT COUNT(*) FROM galgame_favorites f JOIN galgames g ON g.id = f.galgame_id WHERE f.user_id = ? AND g.status = 1) AS favorites`, userID, userID, userID, userID).Scan(&counts).Error
	if err != nil {
		return counts, fmt.Errorf("count profile collections: %w", err)
	}
	return counts, nil
}

func (r *UserProfileRepository) UpdateProfile(ctx context.Context, userID uint, values map[string]any, avatarSet bool, avatarID *uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(values) > 0 {
			result := tx.Model(&model.UserProfile{}).Where("user_id = ?", userID).Updates(values)
			if result.Error != nil {
				return fmt.Errorf("update user profile: %w", result.Error)
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}
		if avatarSet {
			if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("avatar_asset_id", avatarID).Error; err != nil {
				return fmt.Errorf("synchronize user avatar: %w", err)
			}
		}
		return nil
	})
}

func (r *UserProfileRepository) Privacy(ctx context.Context, userID uint) (*model.UserPrivacySettings, error) {
	var settings model.UserPrivacySettings
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Take(&settings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		defaults := model.DefaultUserPrivacySettings(userID)
		return &defaults, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find privacy settings: %w", err)
	}
	return &settings, nil
}

func (r *UserProfileRepository) UpdatePrivacy(ctx context.Context, userID uint, values map[string]any) error {
	if len(values) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).Model(&model.UserPrivacySettings{}).Where("user_id = ?", userID).Updates(values)
	if result.Error != nil {
		return fmt.Errorf("update privacy settings: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *UserProfileRepository) ListPosts(ctx context.Context, userID uint, page, limit int) ([]model.ProfilePost, int64, error) {
	base := r.db.WithContext(ctx).Table("posts").Where("posts.author_id = ?", userID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count user posts: %w", err)
	}
	items := make([]model.ProfilePost, 0)
	err := base.Select("posts.*, COALESCE(galgames.title, '') AS galgame_title").
		Joins("LEFT JOIN galgames ON galgames.id = posts.galgame_id").
		Order("posts.created_at DESC").Order("posts.id DESC").Offset((page - 1) * limit).Limit(limit).Scan(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list user posts: %w", err)
	}
	return items, total, nil
}

func (r *UserProfileRepository) ListComments(ctx context.Context, userID uint, page, limit int) ([]model.ProfileComment, int64, error) {
	base := r.db.WithContext(ctx).Table("comments").Where("comments.author_id = ?", userID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count user comments: %w", err)
	}
	items := make([]model.ProfileComment, 0)
	err := base.Select("comments.*, posts.title AS post_title").Joins("JOIN posts ON posts.id = comments.post_id").
		Order("comments.created_at DESC").Order("comments.id DESC").Offset((page - 1) * limit).Limit(limit).Scan(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list user comments: %w", err)
	}
	return items, total, nil
}

func (r *UserProfileRepository) ListRatings(ctx context.Context, userID uint, page, limit int) ([]model.ProfileGalgameItem, int64, error) {
	base := r.db.WithContext(ctx).Table("galgame_ratings AS relations").Joins("JOIN galgames ON galgames.id = relations.galgame_id AND galgames.status = 1").Where("relations.user_id = ?", userID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count user ratings: %w", err)
	}
	items := make([]model.ProfileGalgameItem, 0)
	err := base.Select("galgames.id, galgames.title, galgames.slug, galgames.cover_url, galgames.cover_sensitive, relations.score, relations.created_at, relations.updated_at").
		Order("relations.updated_at DESC").Order("relations.id DESC").Offset((page - 1) * limit).Limit(limit).Scan(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list user ratings: %w", err)
	}
	return items, total, nil
}

func (r *UserProfileRepository) ListFavorites(ctx context.Context, userID uint, page, limit int) ([]model.ProfileGalgameItem, int64, error) {
	base := r.db.WithContext(ctx).Table("galgame_favorites AS relations").Joins("JOIN galgames ON galgames.id = relations.galgame_id AND galgames.status = 1").Where("relations.user_id = ?", userID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count user favorites: %w", err)
	}
	items := make([]model.ProfileGalgameItem, 0)
	err := base.Select("galgames.id, galgames.title, galgames.slug, galgames.cover_url, galgames.cover_sensitive, relations.created_at, relations.created_at AS updated_at").
		Order("relations.created_at DESC").Order("relations.id DESC").Offset((page - 1) * limit).Limit(limit).Scan(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list user favorites: %w", err)
	}
	return items, total, nil
}

func (r *UserProfileRepository) RecordActivity(ctx context.Context, activity *model.UserActivity) error {
	if err := r.db.WithContext(ctx).Create(activity).Error; err != nil {
		return fmt.Errorf("record user activity: %w", err)
	}
	return nil
}

func (r *UserProfileRepository) ListActivities(ctx context.Context, userID uint, activityTypes []string, page, limit int) ([]model.UserActivity, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.UserActivity{}).
		Where("user_id = ? AND activity_type IN ?", userID, activityTypes).
		Where(`(
(activity_type = 'post_created' AND post_id IS NOT NULL) OR
(activity_type = 'comment_created' AND comment_id IS NOT NULL) OR
(activity_type = 'resource_submitted' AND resource_id IN (SELECT id FROM resources WHERE status = 1)) OR
(activity_type IN ('rating_created', 'favorite_created', 'review_approved') AND galgame_id IN (SELECT id FROM galgames WHERE status = 1))
)`)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count user activities: %w", err)
	}
	items := make([]model.UserActivity, 0)
	err := base.Order("created_at DESC").Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list user activities: %w", err)
	}
	return items, total, nil
}

func NewActivity(userID uint, activityType string, entityID *uint, metadata map[string]any) (*model.UserActivity, error) {
	activity := &model.UserActivity{UserID: userID, ActivityType: activityType, Metadata: model.ActivityMetadata(metadata)}
	switch activityType {
	case model.ActivityPostCreated:
		activity.PostID = entityID
	case model.ActivityCommentCreated:
		activity.CommentID = entityID
	case model.ActivityRatingCreated, model.ActivityFavoriteCreated, model.ActivityReviewApproved:
		activity.GalgameID = entityID
	case model.ActivityResourceSubmitted:
		activity.ResourceID = entityID
	default:
		return nil, fmt.Errorf("unknown activity type %q", activityType)
	}
	return activity, nil
}
