package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	imageModel "backend/internal/image/model"
	imageService "backend/internal/image/service"
	"backend/internal/user/dto"
	"backend/internal/user/model"
	"backend/internal/user/repository"
	"backend/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrInvalidAvatarAsset     = errors.New("avatar asset is invalid")
	ErrInvalidBackgroundAsset = errors.New("background asset is invalid")
	ErrInvalidBannerAsset     = errors.New("banner asset is invalid")
	ErrInvalidPreferences     = errors.New("invalid background preferences")
	ErrProfileNotFound        = errors.New("user profile not found")
	ErrProfileForbidden       = errors.New("user profile collection is hidden")
	ErrInvalidProfile         = errors.New("invalid user profile")
)

// resolveAvatarURL returns the avatar CDN URL for the user, falling back to the
// legacy avatar column when no usable asset is referenced.
func resolveAvatarURL(
	ctx context.Context,
	images *imageService.ImageAssetService,
	user *model.User,
) string {
	if user.AvatarAssetID == nil || images == nil {
		return user.Avatar
	}
	asset, err := images.GetImage(ctx, *user.AvatarAssetID)
	if err != nil || asset == nil {
		return user.Avatar
	}
	if asset.UserID == nil || *asset.UserID != user.ID {
		return user.Avatar
	}
	return images.BuildPublicURL(asset.ObjectKey)
}

// NewMeData builds the /me payload with the resolved avatar URL.
func NewMeData(ctx context.Context, images *imageService.ImageAssetService, user *model.User) dto.MeData {
	return dto.MeData{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Avatar:   resolveAvatarURL(ctx, images, user),
	}
}

type UserProfileService struct {
	users       *repository.UserAuthRepository
	preferences *repository.UserPreferenceRepository
	images      *imageService.ImageAssetService
	profiles    *repository.UserProfileRepository
	access      *ProfileAccessService
}

func NewUserProfileService(
	users *repository.UserAuthRepository,
	preferences *repository.UserPreferenceRepository,
	images *imageService.ImageAssetService,
	profileRepositories ...*repository.UserProfileRepository,
) *UserProfileService {
	service := &UserProfileService{users: users, preferences: preferences, images: images, access: NewProfileAccessService()}
	if len(profileRepositories) > 0 {
		service.profiles = profileRepositories[0]
	}
	return service
}

// UpdateMe changes the avatar reference after validating ownership and
// category; replaced avatar assets are deleted.
func (s *UserProfileService) UpdateMe(
	ctx context.Context,
	userID uint,
	req *dto.UpdateMeRequest,
) (*dto.MeData, error) {
	if s.profiles != nil {
		profileReq := &dto.UpdateProfileRequest{AvatarAssetID: req.AvatarAssetID, AvatarSet: true}
		if _, err := s.UpdateProfile(ctx, userID, profileReq); err != nil {
			return nil, err
		}
		user, err := s.users.FindUserByID(ctx, userID)
		if err != nil || user == nil {
			return nil, err
		}
		data := NewMeData(ctx, s.images, user)
		return &data, nil
	}
	user, err := s.users.FindUserByID(ctx, userID)
	if err != nil {
		logger.Error("find user for update me", zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if req.AvatarAssetID != nil {
		asset, err := s.images.GetImage(ctx, *req.AvatarAssetID)
		if err != nil {
			if errors.Is(err, imageService.ErrImageNotFound) {
				return nil, ErrInvalidAvatarAsset
			}
			return nil, err
		}
		if asset.Category != imageModel.CategoryAvatar ||
			asset.UserID == nil || *asset.UserID != userID {
			return nil, ErrInvalidAvatarAsset
		}
	}

	previousAssetID := user.AvatarAssetID
	if err := s.users.UpdateAvatarAssetID(ctx, userID, req.AvatarAssetID); err != nil {
		logger.Error("update avatar asset", zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	if previousAssetID != nil &&
		(req.AvatarAssetID == nil || *req.AvatarAssetID != *previousAssetID) {
		s.deleteOwnedAsset(ctx, userID, *previousAssetID, "avatar")
	}

	user.AvatarAssetID = req.AvatarAssetID
	data := NewMeData(ctx, s.images, user)
	return &data, nil
}

func (s *UserProfileService) GetPublicProfile(ctx context.Context, username string, viewerID *uint) (*dto.PublicUserProfile, error) {
	profile, err := s.profiles.FindByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return s.buildPublicProfile(ctx, profile, viewerID)
}

func (s *UserProfileService) GetMyProfile(ctx context.Context, userID uint) (*dto.PublicUserProfile, error) {
	profile, err := s.profiles.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return s.buildPublicProfile(ctx, profile, &userID)
}

func (s *UserProfileService) buildPublicProfile(ctx context.Context, profile *model.PublicProfileRecord, viewerID *uint) (*dto.PublicUserProfile, error) {
	access := s.access.Resolve(profile, viewerID)
	result := &dto.PublicUserProfile{
		PublicUserSummary: dto.PublicUserSummary{ID: profile.ID, Username: profile.Username, DisplayName: profile.DisplayName, AvatarURL: profile.AvatarURL},
		IsSelf:            viewerID != nil && *viewerID == profile.ID,
		IsPrivate:         profile.ProfileVisibility == model.ProfileVisibilityPrivate,
		IsRestricted:      !access.CanViewProfile, Access: access,
	}
	if !access.CanViewProfile {
		return result, nil
	}
	result.BannerURL = profile.BannerURL
	result.Bio = profile.Bio
	result.Gender = profile.Gender
	result.WebsiteURL = profile.WebsiteURL
	registeredAt := profile.RegisteredAt
	result.RegisteredAt = &registeredAt
	if access.CanViewLocation {
		result.Location = profile.Location
	}
	if access.CanViewBirthday && profile.Birthday != nil {
		birthday := profile.Birthday.Format("2006-01-02")
		result.Birthday = &birthday
	}
	counts, err := s.profiles.Counts(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	if access.CanViewPosts {
		result.PostCount = int64Pointer(counts.Posts)
	}
	if access.CanViewComments {
		result.CommentCount = int64Pointer(counts.Comments)
	}
	if access.CanViewRatings {
		result.RatingCount = int64Pointer(counts.Ratings)
	}
	if access.CanViewFavorites {
		result.FavoriteCount = int64Pointer(counts.Favorites)
	}
	return result, nil
}

func (s *UserProfileService) UpdateProfile(ctx context.Context, userID uint, req *dto.UpdateProfileRequest) (*dto.PublicUserProfile, error) {
	previous, err := s.profiles.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if previous == nil {
		return nil, ErrProfileNotFound
	}
	if req.AvatarSet {
		if err := s.validateProfileAsset(ctx, userID, req.AvatarAssetID, imageModel.CategoryAvatar); err != nil {
			if !errors.Is(err, imageService.ErrImageNotFound) && !errors.Is(err, ErrInvalidProfile) {
				return nil, err
			}
			return nil, ErrInvalidAvatarAsset
		}
	}
	if req.BannerSet {
		if err := s.validateProfileAsset(ctx, userID, req.BannerAssetID, imageModel.CategoryProfileBanner); err != nil {
			if !errors.Is(err, imageService.ErrImageNotFound) && !errors.Is(err, ErrInvalidProfile) {
				return nil, err
			}
			return nil, ErrInvalidBannerAsset
		}
	}
	values := map[string]any{"updated_at": time.Now()}
	if req.DisplayName != nil {
		value := strings.TrimSpace(*req.DisplayName)
		if value == "" {
			return nil, ErrInvalidProfile
		}
		values["display_name"] = value
	}
	if req.Bio != nil {
		values["bio"] = strings.TrimSpace(*req.Bio)
	}
	if req.Gender != nil {
		values["gender"] = strings.TrimSpace(*req.Gender)
	}
	if req.Location != nil {
		values["location"] = strings.TrimSpace(*req.Location)
	}
	if req.WebsiteURL != nil {
		websiteURL := strings.TrimSpace(*req.WebsiteURL)
		if websiteURL != "" {
			parsed, parseErr := url.ParseRequestURI(websiteURL)
			if parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				return nil, ErrInvalidProfile
			}
		}
		values["website_url"] = websiteURL
	}
	if req.BirthdaySet {
		if req.Birthday == nil || strings.TrimSpace(*req.Birthday) == "" {
			values["birthday"] = nil
		} else {
			value := strings.TrimSpace(*req.Birthday)
			birthday, parseErr := time.Parse("2006-01-02", value)
			if parseErr != nil || birthday.After(time.Now()) {
				return nil, ErrInvalidProfile
			}
			values["birthday"] = birthday
		}
	}
	if req.AvatarSet {
		values["avatar_asset_id"] = req.AvatarAssetID
	}
	if req.BannerSet {
		values["banner_asset_id"] = req.BannerAssetID
	}
	if err := s.profiles.UpdateProfile(ctx, userID, values, req.AvatarSet, req.AvatarAssetID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	if req.AvatarSet && previous.AvatarAssetID != nil && !sameUintPointer(previous.AvatarAssetID, req.AvatarAssetID) {
		s.deleteOwnedAsset(ctx, userID, *previous.AvatarAssetID, "avatar")
	}
	if req.BannerSet && previous.BannerAssetID != nil && !sameUintPointer(previous.BannerAssetID, req.BannerAssetID) {
		s.deleteOwnedAsset(ctx, userID, *previous.BannerAssetID, "profile banner")
	}
	return s.GetMyProfile(ctx, userID)
}

func (s *UserProfileService) validateProfileAsset(ctx context.Context, userID uint, assetID *uint, category string) error {
	if assetID == nil {
		return nil
	}
	asset, err := s.images.GetImage(ctx, *assetID)
	if err != nil {
		return err
	}
	if asset.UserID == nil || *asset.UserID != userID || asset.Category != category {
		return ErrInvalidProfile
	}
	return nil
}

func (s *UserProfileService) GetPrivacy(ctx context.Context, userID uint) (*dto.PrivacySettingsData, error) {
	settings, err := s.profiles.Privacy(ctx, userID)
	if err != nil {
		return nil, err
	}
	return privacyData(settings), nil
}

func (s *UserProfileService) UpdatePrivacy(ctx context.Context, userID uint, req *dto.UpdatePrivacyRequest) (*dto.PrivacySettingsData, error) {
	values := map[string]any{"updated_at": time.Now()}
	if req.ProfileVisibility != nil {
		values["profile_visibility"] = *req.ProfileVisibility
	}
	if req.ShowLocation != nil {
		values["show_location"] = *req.ShowLocation
	}
	if req.ShowBirthday != nil {
		values["show_birthday"] = *req.ShowBirthday
	}
	if req.ShowPosts != nil {
		values["show_posts"] = *req.ShowPosts
	}
	if req.ShowComments != nil {
		values["show_comments"] = *req.ShowComments
	}
	if req.ShowRatings != nil {
		values["show_ratings"] = *req.ShowRatings
	}
	if req.ShowFavorites != nil {
		values["show_favorites"] = *req.ShowFavorites
	}
	if req.ShowActivity != nil {
		values["show_activity"] = *req.ShowActivity
	}
	if err := s.profiles.UpdatePrivacy(ctx, userID, values); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return s.GetPrivacy(ctx, userID)
}

func privacyData(settings *model.UserPrivacySettings) *dto.PrivacySettingsData {
	return &dto.PrivacySettingsData{
		ProfileVisibility: settings.ProfileVisibility, ShowLocation: settings.ShowLocation,
		ShowBirthday: settings.ShowBirthday, ShowPosts: settings.ShowPosts,
		ShowComments: settings.ShowComments, ShowRatings: settings.ShowRatings,
		ShowFavorites: settings.ShowFavorites, ShowActivity: settings.ShowActivity,
	}
}

func (s *UserProfileService) profileForCollection(ctx context.Context, username string, viewerID *uint, allowed func(dto.ProfileAccess) bool) (*model.PublicProfileRecord, error) {
	profile, err := s.profiles.FindByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	if !allowed(s.access.Resolve(profile, viewerID)) {
		return nil, ErrProfileForbidden
	}
	return profile, nil
}

func profilePagination(page, limit int) (int, int) {
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	return page, limit
}

func (s *UserProfileService) ListPosts(ctx context.Context, username string, viewerID *uint, page, limit int) ([]dto.ProfilePostData, int64, int, int, error) {
	profile, err := s.profileForCollection(ctx, username, viewerID, func(a dto.ProfileAccess) bool { return a.CanViewPosts })
	page, limit = profilePagination(page, limit)
	if err != nil {
		return nil, 0, page, limit, err
	}
	rows, total, err := s.profiles.ListPosts(ctx, profile.ID, page, limit)
	items := make([]dto.ProfilePostData, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.ProfilePostData{ID: row.ID, GalgameID: row.GalgameID, GalgameTitle: row.GalgameTitle, Title: row.Title, Content: row.Content, EditorMode: row.EditorMode, LikeCount: row.LikeCount, CommentCount: row.CommentCount, FavoriteCount: row.FavoriteCount, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return items, total, page, limit, err
}

func (s *UserProfileService) ListComments(ctx context.Context, username string, viewerID *uint, page, limit int) ([]dto.ProfileCommentData, int64, int, int, error) {
	profile, err := s.profileForCollection(ctx, username, viewerID, func(a dto.ProfileAccess) bool { return a.CanViewComments })
	page, limit = profilePagination(page, limit)
	if err != nil {
		return nil, 0, page, limit, err
	}
	rows, total, err := s.profiles.ListComments(ctx, profile.ID, page, limit)
	items := make([]dto.ProfileCommentData, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.ProfileCommentData{ID: row.ID, PostID: row.PostID, PostTitle: row.PostTitle, ParentID: row.ParentID, Content: row.Content, LikeCount: row.LikeCount, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return items, total, page, limit, err
}

func (s *UserProfileService) ListRatings(ctx context.Context, username string, viewerID *uint, page, limit int) ([]dto.ProfileGalgameData, int64, int, int, error) {
	return s.listGalgames(ctx, username, viewerID, page, limit, false)
}

func (s *UserProfileService) ListFavorites(ctx context.Context, username string, viewerID *uint, page, limit int) ([]dto.ProfileGalgameData, int64, int, int, error) {
	return s.listGalgames(ctx, username, viewerID, page, limit, true)
}

func (s *UserProfileService) listGalgames(ctx context.Context, username string, viewerID *uint, page, limit int, favorites bool) ([]dto.ProfileGalgameData, int64, int, int, error) {
	allowed := func(a dto.ProfileAccess) bool {
		if favorites {
			return a.CanViewFavorites
		}
		return a.CanViewRatings
	}
	profile, err := s.profileForCollection(ctx, username, viewerID, allowed)
	page, limit = profilePagination(page, limit)
	if err != nil {
		return nil, 0, page, limit, err
	}
	var rows []model.ProfileGalgameItem
	var total int64
	if favorites {
		rows, total, err = s.profiles.ListFavorites(ctx, profile.ID, page, limit)
	} else {
		rows, total, err = s.profiles.ListRatings(ctx, profile.ID, page, limit)
	}
	items := make([]dto.ProfileGalgameData, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.ProfileGalgameData{ID: row.ID, Title: row.Title, Slug: row.Slug, CoverURL: row.CoverURL, Score: row.Score, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return items, total, page, limit, err
}

func (s *UserProfileService) ListActivities(ctx context.Context, username string, viewerID *uint, page, limit int) ([]dto.UserActivityData, int64, int, int, error) {
	profile, err := s.profileForCollection(ctx, username, viewerID, func(a dto.ProfileAccess) bool { return a.CanViewActivity })
	page, limit = profilePagination(page, limit)
	if err != nil {
		return nil, 0, page, limit, err
	}
	access := s.access.Resolve(profile, viewerID)
	activityTypes := visibleActivityTypes(access)
	rows, total, err := s.profiles.ListActivities(ctx, profile.ID, activityTypes, page, limit)
	if err != nil {
		return nil, 0, page, limit, err
	}
	items := make([]dto.UserActivityData, 0, len(rows))
	for _, row := range rows {
		entityType, entityID := activityEntity(&row)
		items = append(items, dto.UserActivityData{ID: row.ID, Type: row.ActivityType, TargetType: entityType, TargetID: entityID, Metadata: map[string]any(row.Metadata), CreatedAt: row.CreatedAt})
	}
	return items, total, page, limit, nil
}

func visibleActivityTypes(access dto.ProfileAccess) []string {
	activityTypes := []string{model.ActivityResourceSubmitted, model.ActivityReviewApproved}
	if access.CanViewPosts {
		activityTypes = append(activityTypes, model.ActivityPostCreated)
	}
	if access.CanViewComments {
		activityTypes = append(activityTypes, model.ActivityCommentCreated)
	}
	if access.CanViewRatings {
		activityTypes = append(activityTypes, model.ActivityRatingCreated)
	}
	if access.CanViewFavorites {
		activityTypes = append(activityTypes, model.ActivityFavoriteCreated)
	}
	return activityTypes
}

func activityEntity(activity *model.UserActivity) (string, *uint) {
	switch {
	case activity.PostID != nil:
		return "post", activity.PostID
	case activity.CommentID != nil:
		return "comment", activity.CommentID
	case activity.ResourceID != nil:
		return "resource", activity.ResourceID
	case activity.GalgameID != nil:
		return "galgame", activity.GalgameID
	default:
		switch activity.ActivityType {
		case model.ActivityPostCreated:
			return "post", nil
		case model.ActivityCommentCreated:
			return "comment", nil
		case model.ActivityResourceSubmitted:
			return "resource", nil
		default:
			return "galgame", nil
		}
	}
}

func int64Pointer(value int64) *int64 { return &value }
func sameUintPointer(left, right *uint) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

// GetPreferences returns background preferences, defaulting when unset.
func (s *UserProfileService) GetPreferences(
	ctx context.Context,
	userID uint,
) (*dto.UserPreferencesData, error) {
	preference, err := s.preferences.FindByUserID(ctx, userID)
	if err != nil {
		logger.Error("find user preferences", zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	if preference == nil {
		preference = model.DefaultUserPreference(userID)
	}
	return s.toPreferencesData(ctx, preference), nil
}

// UpdatePreferences validates and persists background preferences; replaced
// custom background assets are deleted.
func (s *UserProfileService) UpdatePreferences(
	ctx context.Context,
	userID uint,
	req *dto.UpdateUserPreferencesRequest,
) (*dto.UserPreferencesData, error) {
	previous, err := s.preferences.FindByUserID(ctx, userID)
	if err != nil {
		logger.Error("find user preferences", zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}

	preference := &model.UserPreference{
		UserID:             userID,
		BackgroundSource:   req.BackgroundSource,
		BackgroundAssetID:  nil,
		BackgroundPreset:   nil,
		BackgroundOpacity:  req.BackgroundOpacity,
		BackgroundBlur:     req.BackgroundBlur,
		BackgroundPosition: defaultString(req.BackgroundPosition, model.DefaultUserPreference(userID).BackgroundPosition),
		BackgroundSize:     defaultString(req.BackgroundSize, model.BackgroundSizeCover),
	}

	switch req.BackgroundSource {
	case model.BackgroundSourceNone:
	case model.BackgroundSourcePreset:
		if req.BackgroundPreset == nil || strings.TrimSpace(*req.BackgroundPreset) == "" {
			return nil, ErrInvalidPreferences
		}
		preset := strings.TrimSpace(*req.BackgroundPreset)
		preference.BackgroundPreset = &preset
	case model.BackgroundSourceCustom:
		if req.BackgroundAssetID == nil {
			return nil, ErrInvalidPreferences
		}
		asset, err := s.images.GetImage(ctx, *req.BackgroundAssetID)
		if err != nil {
			if errors.Is(err, imageService.ErrImageNotFound) {
				return nil, ErrInvalidBackgroundAsset
			}
			return nil, err
		}
		if asset.Category != imageModel.CategoryBackground ||
			asset.UserID == nil || *asset.UserID != userID {
			return nil, ErrInvalidBackgroundAsset
		}
		preference.BackgroundAssetID = req.BackgroundAssetID
	default:
		return nil, ErrInvalidPreferences
	}

	if err := s.preferences.Upsert(ctx, preference); err != nil {
		logger.Error("upsert user preferences", zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}

	if previous != nil && previous.BackgroundAssetID != nil &&
		(preference.BackgroundAssetID == nil || *preference.BackgroundAssetID != *previous.BackgroundAssetID) {
		s.deleteOwnedAsset(ctx, userID, *previous.BackgroundAssetID, "background")
	}

	data := s.toPreferencesData(ctx, preference)
	return data, nil
}

func (s *UserProfileService) toPreferencesData(
	ctx context.Context,
	preference *model.UserPreference,
) *dto.UserPreferencesData {
	data := &dto.UserPreferencesData{
		BackgroundSource:   preference.BackgroundSource,
		BackgroundAssetID:  preference.BackgroundAssetID,
		BackgroundPreset:   preference.BackgroundPreset,
		BackgroundOpacity:  preference.BackgroundOpacity,
		BackgroundBlur:     preference.BackgroundBlur,
		BackgroundPosition: preference.BackgroundPosition,
		BackgroundSize:     preference.BackgroundSize,
	}
	if preference.BackgroundAssetID != nil {
		if asset, err := s.images.GetImage(ctx, *preference.BackgroundAssetID); err == nil && asset != nil {
			data.BackgroundImageURL = s.images.BuildPublicURL(asset.ObjectKey)
		}
	}
	return data
}

// deleteOwnedAsset removes a replaced asset; failures are logged but do not
// fail the request because the reference change is already persisted.
func (s *UserProfileService) deleteOwnedAsset(ctx context.Context, userID, assetID uint, kind string) {
	if err := s.images.DeleteImage(ctx, userID, assetID); err != nil {
		logger.Warn("delete replaced "+kind+" asset",
			zap.Uint("user_id", userID), zap.Uint("asset_id", assetID), zap.Error(err))
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
