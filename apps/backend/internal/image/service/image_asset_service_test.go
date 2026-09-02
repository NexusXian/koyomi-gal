package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"backend/internal/image/dto"
	"backend/internal/image/model"
	imageRepo "backend/internal/image/repository"
	"backend/internal/infrastructures/storage"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

type fakeStorage struct {
	objects   map[string]*storage.ObjectMetadata
	deleted   []string
	presigned []string
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{objects: map[string]*storage.ObjectMetadata{}}
}

func (f *fakeStorage) PresignPut(
	_ context.Context,
	key string,
	contentType string,
	_ time.Duration,
) (string, error) {
	f.presigned = append(f.presigned, key)
	return "https://presigned.example.com/" + key + "?ct=" + contentType, nil
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

type imageTestEnv struct {
	images  *ImageAssetService
	storage *fakeStorage
	rbac    *rbacService.RBACService
	db      *gorm.DB
}

func newImageTestEnv(t *testing.T) *imageTestEnv {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	rbacSvc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbacSvc.SeedDefaults(context.Background()); err != nil {
		t.Fatalf("seed rbac defaults: %v", err)
	}
	objectStorage := newFakeStorage()
	return &imageTestEnv{
		images: NewImageAssetService(
			imageRepo.NewImageAssetRepository(db),
			objectStorage,
			rbacSvc,
			nil,
			"https://img.example.com",
		),
		storage: objectStorage,
		rbac:    rbacSvc,
		db:      db,
	}
}

func (e *imageTestEnv) presign(
	t *testing.T,
	userID uint,
	contentType, category string,
	size int64,
) *dto.PresignImageData {
	t.Helper()
	data, err := e.images.CreatePresignedUpload(context.Background(), userID, &dto.PresignImageRequest{
		Filename:    "老婆.png",
		ContentType: contentType,
		Size:        size,
		Category:    category,
	})
	if err != nil {
		t.Fatalf("presign %s upload: %v", category, err)
	}
	return data
}

func TestPresignUploadValidation(t *testing.T) {
	env := newImageTestEnv(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, env.db, "image-validator")

	_, err := env.images.CreatePresignedUpload(ctx, user, &dto.PresignImageRequest{
		Filename: "a.svg", ContentType: "image/svg+xml", Size: 1000, Category: model.CategoryPost,
	})
	if !errors.Is(err, ErrInvalidImageType) {
		t.Fatalf("svg must be rejected, got %v", err)
	}

	_, err = env.images.CreatePresignedUpload(ctx, user, &dto.PresignImageRequest{
		Filename: "a.png", ContentType: "image/png", Size: 16 << 20, Category: model.CategoryPost,
	})
	if !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("oversize post image must be rejected, got %v", err)
	}

	_, err = env.images.CreatePresignedUpload(ctx, user, &dto.PresignImageRequest{
		Filename: "a.png", ContentType: "image/png", Size: 1000, Category: model.CategoryBanner,
	})
	if !errors.Is(err, ErrInvalidImageCategory) {
		t.Fatalf("banner category without permission must be rejected, got %v", err)
	}

	data := env.presign(t, user, "image/png", model.CategoryAvatar, 1000)
	if !strings.HasPrefix(data.ObjectKey, fmt.Sprintf("avatars/%d/", user)) {
		t.Fatalf("object key must isolate by category and user: %s", data.ObjectKey)
	}
	if !strings.HasSuffix(data.ObjectKey, ".png") {
		t.Fatalf("object key extension must derive from the mime type: %s", data.ObjectKey)
	}
	if data.ExpiresIn != 300 {
		t.Fatalf("presign expiry must be 300 seconds, got %d", data.ExpiresIn)
	}
}

func TestCompleteUploadFlow(t *testing.T) {
	env := newImageTestEnv(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, env.db, "image-completer")

	data := env.presign(t, user, "image/webp", model.CategoryPost, 4096)

	other := testutil.CreateUser(t, env.db, "image-completer-other")
	_, err := env.images.CompleteUpload(ctx, other, data.ID, &dto.CompleteUploadRequest{})
	if !errors.Is(err, ErrImageForbidden) {
		t.Fatalf("non-owner complete must be forbidden, got %v", err)
	}

	_, err = env.images.CompleteUpload(ctx, user, data.ID, &dto.CompleteUploadRequest{})
	if !errors.Is(err, ErrImageUploadIncomplete) {
		t.Fatalf("missing object must fail completion, got %v", err)
	}

	env.storage.objects[data.ObjectKey] = &storage.ObjectMetadata{
		Key: data.ObjectKey, Size: 4096, ContentType: "image/webp",
	}
	width, height := 1920, 1080
	asset, err := env.images.CompleteUpload(ctx, user, data.ID, &dto.CompleteUploadRequest{
		Width: &width, Height: &height,
	})
	if err != nil {
		t.Fatalf("complete upload: %v", err)
	}
	if asset.Status != model.ImageStatusActive ||
		asset.Width == nil || *asset.Width != width || asset.Height == nil || *asset.Height != height {
		t.Fatalf("unexpected completed asset: %+v", asset)
	}
	if url := env.images.BuildPublicURL(asset.ObjectKey); url != "https://img.example.com/"+asset.ObjectKey {
		t.Fatalf("unexpected public url: %s", url)
	}

	again, err := env.images.CompleteUpload(ctx, user, data.ID, &dto.CompleteUploadRequest{})
	if err != nil || again.Status != model.ImageStatusActive {
		t.Fatalf("completing an active asset must be idempotent, got %v %+v", err, again)
	}
}

func TestCompleteUploadRejectsMismatch(t *testing.T) {
	env := newImageTestEnv(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, env.db, "image-mismatch")

	data := env.presign(t, user, "image/png", model.CategoryPost, 2048)
	env.storage.objects[data.ObjectKey] = &storage.ObjectMetadata{
		Key: data.ObjectKey, Size: 9999, ContentType: "image/png",
	}
	if _, err := env.images.CompleteUpload(ctx, user, data.ID, &dto.CompleteUploadRequest{}); !errors.Is(err, ErrImageUploadIncomplete) {
		t.Fatalf("size mismatch must fail completion, got %v", err)
	}
}

func TestDeleteImageOwnership(t *testing.T) {
	env := newImageTestEnv(t)
	ctx := context.Background()
	owner := testutil.CreateUser(t, env.db, "image-owner")
	other := testutil.CreateUser(t, env.db, "image-other")

	data := env.presign(t, owner, "image/png", model.CategoryPost, 1000)
	env.storage.objects[data.ObjectKey] = &storage.ObjectMetadata{
		Key: data.ObjectKey, Size: 1000, ContentType: "image/png",
	}
	if _, err := env.images.CompleteUpload(ctx, owner, data.ID, &dto.CompleteUploadRequest{}); err != nil {
		t.Fatalf("complete upload: %v", err)
	}

	if err := env.images.DeleteImage(ctx, other, data.ID); !errors.Is(err, ErrImageForbidden) {
		t.Fatalf("delete by non-owner without permission must be forbidden, got %v", err)
	}

	admin := testutil.CreateUser(t, env.db, "image-admin")
	if err := env.rbac.AssignRoleByCode(context.Background(), admin, rbacService.RoleCodeAdmin); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}
	if err := env.images.DeleteImage(ctx, admin, data.ID); err != nil {
		t.Fatalf("delete by image:delete permission: %v", err)
	}
	if len(env.storage.deleted) != 1 || env.storage.deleted[0] != data.ObjectKey {
		t.Fatalf("object must be removed from storage, got %v", env.storage.deleted)
	}
	if err := env.images.DeleteImage(ctx, owner, data.ID); !errors.Is(err, ErrImageNotFound) {
		t.Fatalf("deleting a deleted asset must be not found, got %v", err)
	}
}

func TestCleanupExpiredPending(t *testing.T) {
	env := newImageTestEnv(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, env.db, "image-cleaner")

	stale := env.presign(t, user, "image/png", model.CategoryPost, 1000)
	env.presign(t, user, "image/png", model.CategoryPost, 1000)
	if err := env.db.Exec(
		"UPDATE image_assets SET created_at = NOW() - INTERVAL '25 hours' WHERE id = ?", stale.ID,
	).Error; err != nil {
		t.Fatalf("age pending asset: %v", err)
	}

	purged, err := env.images.CleanupExpiredPending(ctx)
	if err != nil {
		t.Fatalf("cleanup expired pending: %v", err)
	}
	if purged != 1 {
		t.Fatalf("expected 1 purged asset, got %d", purged)
	}
	var count int64
	if err := env.db.Model(&model.ImageAsset{}).Count(&count).Error; err != nil {
		t.Fatalf("count image assets: %v", err)
	}
	if count != 1 {
		t.Fatalf("only the fresh asset must remain, got %d", count)
	}
	if len(env.storage.deleted) != 1 || env.storage.deleted[0] != stale.ObjectKey {
		t.Fatalf("stale object must be deleted, got %v", env.storage.deleted)
	}
}
