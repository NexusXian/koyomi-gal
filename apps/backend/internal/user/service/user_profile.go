package service

import (
	"context"
	"errors"
	"strings"

	imageModel "backend/internal/image/model"
	imageService "backend/internal/image/service"
	"backend/internal/user/dto"
	"backend/internal/user/model"
	"backend/internal/user/repository"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

var (
	ErrInvalidAvatarAsset     = errors.New("avatar asset is invalid")
	ErrInvalidBackgroundAsset = errors.New("background asset is invalid")
	ErrInvalidPreferences     = errors.New("invalid background preferences")
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
}

func NewUserProfileService(
	users *repository.UserAuthRepository,
	preferences *repository.UserPreferenceRepository,
	images *imageService.ImageAssetService,
) *UserProfileService {
	return &UserProfileService{users: users, preferences: preferences, images: images}
}

// UpdateMe changes the avatar reference after validating ownership and
// category; replaced avatar assets are deleted.
func (s *UserProfileService) UpdateMe(
	ctx context.Context,
	userID uint,
	req *dto.UpdateMeRequest,
) (*dto.MeData, error) {
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
