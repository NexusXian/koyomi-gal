package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"backend/internal/image/dto"
	"backend/internal/image/model"
	"backend/internal/image/repository"
	"backend/internal/infrastructures/storage"
	rbacService "backend/internal/rbac/service"
	"backend/pkg/logger"
)

var (
	ErrInvalidImageType      = errors.New("image type not allowed")
	ErrImageTooLarge         = errors.New("image size exceeds category limit")
	ErrInvalidImageCategory  = errors.New("image category not allowed for this user")
	ErrImageNotFound         = errors.New("image not found")
	ErrImageForbidden        = errors.New("not allowed to manage this image")
	ErrImageInvalidState     = errors.New("image is not in a state that allows this action")
	ErrImageUploadIncomplete = errors.New("uploaded object is missing or does not match the request")
	ErrPresignLimitExceeded  = errors.New("too many upload requests")
	ErrImageQuotaExceeded    = errors.New("daily upload quota exceeded")
)

// Permission codes used for admin-only categories and cross-user management.
const (
	PermissionImageManage = "image:manage"
	PermissionImageDelete = "image:delete"
)

const (
	presignTTL         = 5 * time.Minute
	maxImageSize       = 20 << 20
	pendingExpiration  = 24 * time.Hour
	cleanupInterval    = time.Hour
	cleanupBatchSize   = 100
	presignWindow      = 10 * time.Minute
	presignWindowLimit = 60
	dailyQuota         = 500 << 20
	dailyQuotaAdmin    = 2 << 30
	presignCountPrefix = "image:presign:count:"
	presignDailyPrefix = "image:presign:daily:"
)

// mimeTypeExtensions is the allowlist of uploadable image types; the object
// extension always derives from the MIME type, never from the client filename.
var mimeTypeExtensions = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/avif": "avif",
	"image/gif":  "gif",
}

// categorySizeLimits caps declared upload size per category.
var categorySizeLimits = map[string]int64{
	model.CategoryAvatar:        5 << 20,
	model.CategoryPost:          15 << 20,
	model.CategoryComment:       10 << 20,
	model.CategoryGalgame:       maxImageSize,
	model.CategoryBackground:    maxImageSize,
	model.CategoryBanner:        maxImageSize,
	model.CategoryProfileBanner: maxImageSize,
	model.CategoryAdmin:         maxImageSize,
}

// adminCategories require an RBAC permission on top of login.
var adminCategories = map[string]bool{
	model.CategoryGalgame: true,
	model.CategoryBanner:  true,
	model.CategoryAdmin:   true,
}

type ImageAssetService struct {
	assets    *repository.ImageAssetRepository
	storage   storage.ObjectStorage
	rbac      *rbacService.RBACService
	cache     *redis.Client
	publicURL string
}

func NewImageAssetService(
	assets *repository.ImageAssetRepository,
	objectStorage storage.ObjectStorage,
	rbac *rbacService.RBACService,
	cache *redis.Client,
	publicURL string,
) *ImageAssetService {
	return &ImageAssetService{
		assets: assets, storage: objectStorage, rbac: rbac,
		cache: cache, publicURL: publicURL,
	}
}

// CreatePresignedUpload validates the request, records a pending asset, and
// returns a short-lived PUT URL scoped to the generated object key.
func (s *ImageAssetService) CreatePresignedUpload(
	ctx context.Context,
	userID uint,
	req *dto.PresignImageRequest,
) (*dto.PresignImageData, error) {
	extension, ok := mimeTypeExtensions[req.ContentType]
	if !ok {
		return nil, ErrInvalidImageType
	}
	limit := categorySizeLimits[req.Category]
	if req.Size <= 0 || req.Size > limit {
		return nil, ErrImageTooLarge
	}

	isAdmin := false
	if adminCategories[req.Category] {
		allowed, err := s.rbac.HasPermission(ctx, userID, PermissionImageManage)
		if err != nil {
			logger.Error("check image category permission",
				zap.Uint("user_id", userID), zap.String("category", req.Category), zap.Error(err))
			return nil, err
		}
		if !allowed {
			return nil, ErrInvalidImageCategory
		}
		isAdmin = true
	}

	if err := s.enforcePresignLimits(ctx, userID, req.Size, isAdmin); err != nil {
		return nil, err
	}

	asset := &model.ImageAsset{
		UserID:       &userID,
		ObjectKey:    generateObjectKey(req.Category, userID, extension),
		OriginalName: truncate(strings.TrimSpace(req.Filename), 255),
		MimeType:     req.ContentType,
		Extension:    extension,
		Size:         req.Size,
		Category:     req.Category,
		Status:       model.ImageStatusPending,
	}
	if err := s.assets.Create(ctx, asset); err != nil {
		logger.Error("create image asset", zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}

	uploadURL, err := s.storage.PresignPut(ctx, asset.ObjectKey, asset.MimeType, presignTTL)
	if err != nil {
		logger.Error("presign image upload",
			zap.Uint("asset_id", asset.ID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}

	logger.Info("image presign created",
		zap.Uint("user_id", userID), zap.Uint("asset_id", asset.ID),
		zap.String("object_key", asset.ObjectKey), zap.String("mime", asset.MimeType),
		zap.Int64("size", asset.Size), zap.String("category", asset.Category),
	)
	return &dto.PresignImageData{
		ID:        asset.ID,
		ObjectKey: asset.ObjectKey,
		UploadURL: uploadURL,
		ExpiresIn: int(presignTTL.Seconds()),
	}, nil
}

// CompleteUpload verifies the object exists in storage and matches the
// declared metadata, then activates the asset. Completing an already-active
// asset is idempotent.
func (s *ImageAssetService) CompleteUpload(
	ctx context.Context,
	userID, id uint,
	req *dto.CompleteUploadRequest,
) (*model.ImageAsset, error) {
	asset, err := s.assets.FindByID(ctx, id)
	if err != nil {
		logger.Error("find image asset", zap.Uint("asset_id", id), zap.Error(err))
		return nil, err
	}
	if asset == nil {
		return nil, ErrImageNotFound
	}
	if asset.UserID == nil || *asset.UserID != userID {
		return nil, ErrImageForbidden
	}
	if asset.Status == model.ImageStatusActive {
		return asset, nil
	}
	if asset.Status != model.ImageStatusPending {
		return nil, ErrImageInvalidState
	}

	metadata, err := s.storage.Head(ctx, asset.ObjectKey)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return nil, ErrImageUploadIncomplete
		}
		logger.Error("head image object", zap.Uint("asset_id", id), zap.Error(err))
		return nil, err
	}
	if metadata.Size != asset.Size || metadata.ContentType != asset.MimeType {
		return nil, ErrImageUploadIncomplete
	}

	updated, err := s.assets.MarkActive(ctx, id, metadata.Size, req.Width, req.Height)
	if err != nil {
		logger.Error("mark image asset active", zap.Uint("asset_id", id), zap.Error(err))
		return nil, err
	}
	if !updated {
		asset, err = s.assets.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if asset == nil || asset.Status != model.ImageStatusActive {
			return nil, ErrImageInvalidState
		}
		return asset, nil
	}

	logger.Info("image upload completed",
		zap.Uint("user_id", userID), zap.Uint("asset_id", id),
		zap.String("object_key", asset.ObjectKey), zap.Int64("size", metadata.Size),
	)
	asset, err = s.assets.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return asset, nil
}

// GetImage returns an active asset.
func (s *ImageAssetService) GetImage(ctx context.Context, id uint) (*model.ImageAsset, error) {
	asset, err := s.assets.FindByID(ctx, id)
	if err != nil {
		logger.Error("find image asset", zap.Uint("asset_id", id), zap.Error(err))
		return nil, err
	}
	if asset == nil || asset.Status != model.ImageStatusActive {
		return nil, ErrImageNotFound
	}
	return asset, nil
}

// GetAdmin returns an asset regardless of status for the admin console.
func (s *ImageAssetService) GetAdmin(ctx context.Context, id uint) (*model.ImageAsset, error) {
	asset, err := s.assets.FindByID(ctx, id)
	if err != nil {
		logger.Error("find image asset", zap.Uint("asset_id", id), zap.Error(err))
		return nil, err
	}
	if asset == nil || asset.Status == model.ImageStatusDeleted {
		return nil, ErrImageNotFound
	}
	return asset, nil
}

// DeleteImage removes the object from storage and soft-deletes the record.
// Owners may delete their own images; image:delete allows deleting any.
func (s *ImageAssetService) DeleteImage(ctx context.Context, actorID, id uint) error {
	asset, err := s.assets.FindByID(ctx, id)
	if err != nil {
		logger.Error("find image asset", zap.Uint("asset_id", id), zap.Error(err))
		return err
	}
	if asset == nil {
		return ErrImageNotFound
	}
	if asset.UserID == nil || *asset.UserID != actorID {
		allowed, err := s.rbac.HasPermission(ctx, actorID, PermissionImageDelete)
		if err != nil {
			logger.Error("check image delete permission",
				zap.Uint("actor_id", actorID), zap.Uint("asset_id", id), zap.Error(err))
			return err
		}
		if !allowed {
			return ErrImageForbidden
		}
	}
	if asset.Status == model.ImageStatusDeleted {
		return ErrImageNotFound
	}

	if err := s.storage.Delete(ctx, asset.ObjectKey); err != nil {
		logger.Error("delete image object", zap.Uint("asset_id", id), zap.Error(err))
		return err
	}
	deleted, err := s.assets.SoftDelete(ctx, id)
	if err != nil {
		logger.Error("soft delete image asset", zap.Uint("asset_id", id), zap.Error(err))
		return err
	}
	if !deleted {
		return ErrImageNotFound
	}
	logger.Info("image deleted",
		zap.Uint("actor_id", actorID), zap.Uint("asset_id", id),
		zap.String("object_key", asset.ObjectKey),
	)
	return nil
}

// ListAdmin returns one page of image assets for the admin console.
func (s *ImageAssetService) ListAdmin(
	ctx context.Context,
	query dto.AdminImageQuery,
) ([]model.ImageAsset, int64, int, int, error) {
	page, limit := query.Page, query.Limit
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	assets, total, err := s.assets.ListAdmin(ctx, repository.AdminImageFilter{
		Page: page, Limit: limit, Category: query.Category,
		UserID: query.UserID, Status: query.Status,
	})
	if err != nil {
		logger.Error("list admin image assets", zap.Error(err))
		return nil, 0, page, limit, err
	}
	return assets, total, page, limit, nil
}

// CleanupExpiredPending removes pending assets older than pendingExpiration
// together with their stored objects. It returns the number of purged assets.
func (s *ImageAssetService) CleanupExpiredPending(ctx context.Context) (int, error) {
	assets, err := s.assets.ListPendingBefore(ctx, time.Now().Add(-pendingExpiration), cleanupBatchSize)
	if err != nil {
		logger.Error("list expired pending image assets", zap.Error(err))
		return 0, err
	}
	purged := 0
	for i := range assets {
		if err := s.storage.Delete(ctx, assets[i].ObjectKey); err != nil {
			logger.Error("delete expired image object",
				zap.Uint("asset_id", assets[i].ID), zap.Error(err))
			continue
		}
		if err := s.assets.HardDelete(ctx, assets[i].ID); err != nil {
			logger.Error("hard delete expired image asset",
				zap.Uint("asset_id", assets[i].ID), zap.Error(err))
			continue
		}
		purged++
	}
	return purged, nil
}

// StartCleanupLoop periodically purges abandoned pending uploads until the
// context is cancelled. The returned stop function is safe to call twice.
func (s *ImageAssetService) StartCleanupLoop(parent context.Context) (stop func()) {
	ctx, cancel := context.WithCancel(parent)
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				purged, err := s.CleanupExpiredPending(ctx)
				if err != nil {
					logger.Error("run image cleanup", zap.Error(err))
					continue
				}
				if purged > 0 {
					logger.Info("image cleanup purged pending assets", zap.Int("count", purged))
				}
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(cancel) }
}

func (s *ImageAssetService) BuildPublicURL(objectKey string) string {
	return strings.TrimRight(s.publicURL, "/") + "/" + strings.TrimLeft(objectKey, "/")
}

// enforcePresignLimits applies a fixed-window request counter and a daily
// declared-size quota. Limits are skipped when Redis is unavailable.
func (s *ImageAssetService) enforcePresignLimits(ctx context.Context, userID uint, size int64, isAdmin bool) error {
	if s.cache == nil {
		return nil
	}

	countKey := presignCountPrefix + fmt.Sprintf("%d", userID)
	count, err := s.cache.Incr(ctx, countKey).Result()
	if err != nil {
		logger.Warn("image presign count limit unavailable", zap.Uint("user_id", userID), zap.Error(err))
		return nil
	}
	if count == 1 {
		s.cache.Expire(ctx, countKey, presignWindow)
	}
	if count > presignWindowLimit {
		return ErrPresignLimitExceeded
	}

	quota := int64(dailyQuota)
	if isAdmin {
		quota = dailyQuotaAdmin
	}
	dailyKey := presignDailyPrefix + time.Now().UTC().Format("20060102") + ":" + fmt.Sprintf("%d", userID)
	bytes, err := s.cache.IncrBy(ctx, dailyKey, size).Result()
	if err != nil {
		logger.Warn("image presign quota limit unavailable", zap.Uint("user_id", userID), zap.Error(err))
		return nil
	}
	s.cache.Expire(ctx, dailyKey, 48*time.Hour)
	if bytes > quota {
		return ErrImageQuotaExceeded
	}
	return nil
}

// generateObjectKey always builds the storage path server-side so clients can
// never influence where an object is written.
func generateObjectKey(category string, userID uint, extension string) string {
	now := time.Now()
	return fmt.Sprintf(
		"%s/%d/%04d/%02d/%s.%s",
		category, userID, now.Year(), int(now.Month()), uuid.NewString(), extension,
	)
}

// truncate caps the string to max runes so multi-byte filenames stay valid
// UTF-8 for the varchar(255) column.
func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
