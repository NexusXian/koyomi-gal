package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	imageDTO "backend/internal/image/dto"
	imageModel "backend/internal/image/model"
	imageRepo "backend/internal/image/repository"
	imageService "backend/internal/image/service"
	"backend/internal/infrastructures/storage"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/testutil"
	"backend/internal/user/dto"
	userRepo "backend/internal/user/repository"

	"gorm.io/gorm"
)

type fakeStorage struct {
	objects map[string]*storage.ObjectMetadata
	deleted []string
}

func (f *fakeStorage) PresignPut(
	_ context.Context, key string, _ string, _ time.Duration,
) (string, error) {
	return "https://presigned.example.com/" + key, nil
}

func (f *fakeStorage) Head(_ context.Context, key string) (*storage.ObjectMetadata, error) {
	metadata, ok := f.objects[key]
	if !ok {
		return nil, storage.ErrObjectNotFound
	}
	return metadata, nil
}

func (f *fakeStorage) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	delete(f.objects, key)
	return nil
}

type profileTestEnv struct {
	profile *UserProfileService
	images  *imageService.ImageAssetService
	storage *fakeStorage
	db      *gorm.DB
}

func newProfileTestEnv(t *testing.T) *profileTestEnv {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	rbacSvc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbacSvc.SeedDefaults(context.Background()); err != nil {
		t.Fatalf("seed rbac defaults: %v", err)
	}
	objectStorage := &fakeStorage{objects: map[string]*storage.ObjectMetadata{}}
	imageSvc := imageService.NewImageAssetService(
		imageRepo.NewImageAssetRepository(db),
		objectStorage,
		rbacSvc,
		nil,
		"https://img.example.com",
	)
	return &profileTestEnv{
		profile: NewUserProfileService(
			userRepo.NewUserAuthRepository(db),
			userRepo.NewUserPreferenceRepository(db),
			imageSvc,
		),
		images:  imageSvc,
		storage: objectStorage,
		db:      db,
	}
}

func (e *profileTestEnv) createActiveImage(
	t *testing.T,
	userID uint,
	category, contentType string,
) uint {
	t.Helper()
	ctx := context.Background()
	data, err := e.images.CreatePresignedUpload(ctx, userID, &imageDTO.PresignImageRequest{
		Filename: "test.png", ContentType: contentType, Size: 1024, Category: category,
	})
	if err != nil {
		t.Fatalf("presign %s image: %v", category, err)
	}
	e.storage.objects[data.ObjectKey] = &storage.ObjectMetadata{
		Key: data.ObjectKey, Size: 1024, ContentType: contentType,
	}
	asset, err := e.images.CompleteUpload(ctx, userID, data.ID, &imageDTO.CompleteUploadRequest{})
	if err != nil {
		t.Fatalf("complete %s image: %v", category, err)
	}
	return asset.ID
}

func TestUpdateMeAvatar(t *testing.T) {
	env := newProfileTestEnv(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, env.db, "profile-avatar-user")

	first := env.createActiveImage(t, user, imageModel.CategoryAvatar, "image/png")
	me, err := env.profile.UpdateMe(ctx, user, &dto.UpdateMeRequest{AvatarAssetID: &first})
	if err != nil {
		t.Fatalf("update me: %v", err)
	}
	if !strings.HasPrefix(me.Avatar, "https://img.example.com/avatars/") {
		t.Fatalf("avatar must resolve to the CDN url, got %q", me.Avatar)
	}
	if me.ID != user || me.Username != "profile-avatar-user" {
		t.Fatalf("unexpected me data: %+v", me)
	}

	second := env.createActiveImage(t, user, imageModel.CategoryAvatar, "image/png")
	if _, err := env.profile.UpdateMe(ctx, user, &dto.UpdateMeRequest{AvatarAssetID: &second}); err != nil {
		t.Fatalf("replace avatar: %v", err)
	}
	if len(env.storage.deleted) != 1 {
		t.Fatalf("replaced avatar object must be deleted, got %v", env.storage.deleted)
	}
	if _, err := env.images.GetImage(ctx, first); !errors.Is(err, imageService.ErrImageNotFound) {
		t.Fatalf("replaced avatar record must be deleted, got %v", err)
	}

	if _, err := env.profile.UpdateMe(ctx, user, &dto.UpdateMeRequest{}); err != nil {
		t.Fatalf("clear avatar: %v", err)
	}
	if len(env.storage.deleted) != 2 {
		t.Fatalf("cleared avatar object must be deleted, got %v", env.storage.deleted)
	}

	other := testutil.CreateUser(t, env.db, "profile-avatar-other")
	strangersAsset := env.createActiveImage(t, other, imageModel.CategoryAvatar, "image/png")
	if _, err := env.profile.UpdateMe(ctx, user, &dto.UpdateMeRequest{AvatarAssetID: &strangersAsset}); !errors.Is(err, ErrInvalidAvatarAsset) {
		t.Fatalf("using someone else's avatar must be rejected, got %v", err)
	}

	postAsset := env.createActiveImage(t, user, imageModel.CategoryPost, "image/png")
	if _, err := env.profile.UpdateMe(ctx, user, &dto.UpdateMeRequest{AvatarAssetID: &postAsset}); !errors.Is(err, ErrInvalidAvatarAsset) {
		t.Fatalf("non-avatar category must be rejected, got %v", err)
	}
}

func TestPreferencesFlow(t *testing.T) {
	env := newProfileTestEnv(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, env.db, "profile-preference-user")

	defaults, err := env.profile.GetPreferences(ctx, user)
	if err != nil {
		t.Fatalf("get default preferences: %v", err)
	}
	if defaults.BackgroundSource != "none" || defaults.BackgroundSize != "cover" || defaults.BackgroundOpacity != 0.35 {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}

	preset := "default-01"
	saved, err := env.profile.UpdatePreferences(ctx, user, &dto.UpdateUserPreferencesRequest{
		BackgroundSource: "preset", BackgroundPreset: &preset,
		BackgroundOpacity: 0.5, BackgroundBlur: 4,
	})
	if err != nil {
		t.Fatalf("save preset preferences: %v", err)
	}
	if saved.BackgroundPreset == nil || *saved.BackgroundPreset != preset || saved.BackgroundOpacity != 0.5 {
		t.Fatalf("unexpected saved preferences: %+v", saved)
	}

	background := env.createActiveImage(t, user, imageModel.CategoryBackground, "image/webp")
	custom, err := env.profile.UpdatePreferences(ctx, user, &dto.UpdateUserPreferencesRequest{
		BackgroundSource: "custom", BackgroundAssetID: &background,
		BackgroundOpacity: 0.5, BackgroundBlur: 4,
	})
	if err != nil {
		t.Fatalf("save custom preferences: %v", err)
	}
	if custom.BackgroundImageURL == "" {
		t.Fatalf("custom preferences must resolve the image url: %+v", custom)
	}

	replacement := env.createActiveImage(t, user, imageModel.CategoryBackground, "image/webp")
	if _, err := env.profile.UpdatePreferences(ctx, user, &dto.UpdateUserPreferencesRequest{
		BackgroundSource: "custom", BackgroundAssetID: &replacement,
		BackgroundOpacity: 0.5, BackgroundBlur: 4,
	}); err != nil {
		t.Fatalf("replace custom background: %v", err)
	}
	if len(env.storage.deleted) != 1 {
		t.Fatalf("replaced background object must be deleted, got %v", env.storage.deleted)
	}

	if _, err := env.profile.UpdatePreferences(ctx, user, &dto.UpdateUserPreferencesRequest{
		BackgroundSource: "custom", BackgroundOpacity: 0.5,
	}); !errors.Is(err, ErrInvalidPreferences) {
		t.Fatalf("custom source without asset must be rejected, got %v", err)
	}

	avatarAsset := env.createActiveImage(t, user, imageModel.CategoryAvatar, "image/png")
	if _, err := env.profile.UpdatePreferences(ctx, user, &dto.UpdateUserPreferencesRequest{
		BackgroundSource: "custom", BackgroundAssetID: &avatarAsset,
		BackgroundOpacity: 0.5,
	}); !errors.Is(err, ErrInvalidBackgroundAsset) {
		t.Fatalf("non-background category must be rejected, got %v", err)
	}
}
