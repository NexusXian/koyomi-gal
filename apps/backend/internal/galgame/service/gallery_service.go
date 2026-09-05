package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	contributionModel "backend/internal/contribution/model"
	contributionService "backend/internal/contribution/service"
	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	imageService "backend/internal/image/service"
	relationModel "backend/internal/relation/model"
	"backend/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrGalleryGalgameNotFound = errors.New("galgame not found")
	ErrGalleryImageNotFound   = errors.New("gallery image not found")
	ErrGalleryAssetNotFound   = errors.New("gallery asset not found or not an image")
	ErrGalleryAssetDuplicate  = errors.New("asset already exists in this gallery")
	ErrGalleryURLDuplicate    = errors.New("external url already exists in this gallery")
	ErrGalleryLimitExceeded   = errors.New("gallery image limit exceeded")
	ErrGalleryInvalidReorder  = errors.New("gallery reorder ids must exactly cover all gallery images of this galgame")
	ErrGalleryInvalidSource   = errors.New("exactly one of asset_id or external_url is required")
	ErrGalleryInvalidURL      = errors.New("external url is invalid")
)

const (
	// MaxGalleryImages caps how many pending + published images one galgame
	// gallery may hold; rejected entries no longer consume a slot.
	MaxGalleryImages = 30
	// MaxExternalImageURLLength bounds stored external URLs.
	MaxExternalImageURLLength = 2048
	// MaxBatchGalleryItems caps one batch import request.
	MaxBatchGalleryItems = 100
	// MaxGalleryReviewReasonLength bounds reject reasons.
	MaxGalleryReviewReasonLength = 500
)

type GalleryService struct {
	galgames      *repository.GalgameRepository
	gallery       *repository.GalleryRepository
	images        *imageService.ImageAssetService
	contributions *contributionService.ContributionService
}

func (s *GalleryService) SetContributionService(contributions *contributionService.ContributionService) {
	s.contributions = contributions
}

func NewGalleryService(
	galgames *repository.GalgameRepository,
	gallery *repository.GalleryRepository,
	images *imageService.ImageAssetService,
) *GalleryService {
	return &GalleryService{galgames: galgames, gallery: gallery, images: images}
}

// ListPublishedGallery returns the published gallery of a published galgame.
func (s *GalleryService) ListPublishedGallery(ctx context.Context, galgameID uint) (dto.GalleryListData, error) {
	galgame, err := s.galgames.FindPublishedByID(ctx, galgameID)
	if err != nil {
		logger.Error("find galgame for gallery", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return dto.GalleryListData{}, err
	}
	if galgame == nil {
		return dto.GalleryListData{}, ErrGalleryGalgameNotFound
	}
	images, err := s.gallery.ListByGalgameID(ctx, galgameID, model.GalleryImageStatusPublished)
	if err != nil {
		logger.Error("list gallery images", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return dto.GalleryListData{}, err
	}
	return dto.NewGalleryImageList(images, s.images), nil
}

// ListAdminGallery returns every gallery entry of a galgame in any status,
// including pending and rejected images.
func (s *GalleryService) ListAdminGallery(ctx context.Context, galgameID uint) (dto.GalleryListData, error) {
	galgame, err := s.galgames.FindByID(ctx, galgameID)
	if err != nil {
		logger.Error("find galgame for admin gallery", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return dto.GalleryListData{}, err
	}
	if galgame == nil {
		return dto.GalleryListData{}, ErrGalleryGalgameNotFound
	}
	images, err := s.gallery.ListByGalgameID(ctx, galgameID)
	if err != nil {
		logger.Error("list admin gallery images", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return dto.GalleryListData{}, err
	}
	return dto.NewAdminGalleryImageList(images, s.images), nil
}

// validateExternalImageURL accepts http/https URLs without credentials or
// whitespace. The URL is stored verbatim; the server never fetches it, so no
// SSRF surface is introduced by validation itself.
func validateExternalImageURL(raw string) error {
	if raw == "" || len(raw) > MaxExternalImageURLLength {
		return ErrGalleryInvalidURL
	}
	if raw != strings.TrimSpace(raw) || strings.ContainsAny(raw, " \t\r\n\x00") {
		return ErrGalleryInvalidURL
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ErrGalleryInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrGalleryInvalidURL
	}
	if parsed.Host == "" || parsed.User != nil {
		return ErrGalleryInvalidURL
	}
	return nil
}

// resolveSource picks exactly one of asset / external URL and validates it.
func (s *GalleryService) resolveSource(
	ctx context.Context,
	assetID *uint,
	externalURL string,
) (int16, *uint, string, error) {
	externalURL = strings.TrimSpace(externalURL)
	hasAsset := assetID != nil && *assetID > 0
	if hasAsset == (externalURL != "") {
		return 0, nil, "", ErrGalleryInvalidSource
	}
	if hasAsset {
		asset, err := s.images.GetImage(ctx, *assetID)
		if err != nil {
			if errors.Is(err, imageService.ErrImageNotFound) {
				return 0, nil, "", ErrGalleryAssetNotFound
			}
			return 0, nil, "", err
		}
		if !strings.HasPrefix(asset.MimeType, "image/") {
			return 0, nil, "", ErrGalleryAssetNotFound
		}
		return model.GallerySourceAsset, assetID, "", nil
	}
	if err := validateExternalImageURL(externalURL); err != nil {
		return 0, nil, "", err
	}
	return model.GallerySourceExternal, nil, externalURL, nil
}

// buildImage assembles a pending gallery image appended after the last entry.
func buildImage(galgameID uint, actorID uint, sourceType int16, assetID *uint, externalURL, title, description string, imageType int16, isSpoiler bool, sortOrder int) *model.GalleryImage {
	return &model.GalleryImage{
		GalgameID:   galgameID,
		SourceType:  sourceType,
		AssetID:     assetID,
		ExternalURL: externalURL,
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(description),
		SortOrder:   sortOrder,
		ImageType:   imageType,
		IsSpoiler:   isSpoiler,
		Status:      model.GalleryImageStatusPending,
		CreatedBy:   &actorID,
	}
}

// CreateGalleryImage adds an asset-backed or external image. Every creation,
// regardless of who submits it, lands in pending; publishing happens only via
// the review endpoints.
func (s *GalleryService) CreateGalleryImage(
	ctx context.Context,
	galgameID uint,
	actorID uint,
	req *dto.CreateGalleryImageRequest,
) (*dto.GalleryImageData, error) {
	galgame, err := s.galgames.FindByID(ctx, galgameID)
	if err != nil {
		logger.Error("find galgame for gallery create", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}
	if galgame == nil {
		return nil, ErrGalleryGalgameNotFound
	}

	sourceType, assetID, externalURL, err := s.resolveSource(ctx, req.AssetID, req.ExternalURL)
	if err != nil {
		return nil, err
	}

	total, err := s.gallery.CountActiveByGalgameID(ctx, galgameID)
	if err != nil {
		logger.Error("count gallery images", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}
	if total >= MaxGalleryImages {
		return nil, ErrGalleryLimitExceeded
	}

	if sourceType == model.GallerySourceAsset {
		existing, err := s.gallery.FindByGalgameIDAndAssetID(ctx, galgameID, *assetID)
		if err != nil {
			logger.Error("find duplicate gallery image", zap.Uint("galgame_id", galgameID), zap.Error(err))
			return nil, err
		}
		if existing != nil {
			return nil, ErrGalleryAssetDuplicate
		}
	} else {
		existing, err := s.gallery.FindByGalgameIDAndExternalURL(ctx, galgameID, externalURL)
		if err != nil {
			logger.Error("find duplicate gallery image", zap.Uint("galgame_id", galgameID), zap.Error(err))
			return nil, err
		}
		if existing != nil {
			return nil, ErrGalleryURLDuplicate
		}
	}

	maxSort, err := s.gallery.MaxSortOrder(ctx, galgameID)
	if err != nil {
		logger.Error("max gallery sort order", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}

	image := buildImage(galgameID, actorID, sourceType, assetID, externalURL,
		req.Title, req.Description, req.ImageType, req.IsSpoiler, maxSort+1)
	if err := s.gallery.Create(ctx, image); err != nil {
		logger.Error("create gallery image",
			zap.Uint("galgame_id", galgameID), zap.Uintp("asset_id", assetID), zap.Error(err))
		return nil, err
	}
	created, err := s.gallery.FindByID(ctx, image.ID)
	if err != nil {
		logger.Error("reload gallery image", zap.Uint("gallery_id", image.ID), zap.Error(err))
		return nil, err
	}
	if created == nil {
		return nil, ErrGalleryImageNotFound
	}
	data := dto.NewGalleryImageData(created, s.images)
	return &data, nil
}

// BatchCreateGalleryImages imports external URLs in one transaction. Duplicate
// URLs (in-request or already stored) are skipped so importer runs are
// idempotent; invalid URLs count as failed without aborting the batch.
func (s *GalleryService) BatchCreateGalleryImages(
	ctx context.Context,
	galgameID uint,
	actorID uint,
	req *dto.BatchCreateGalleryRequest,
) (*dto.BatchGalleryResultData, error) {
	galgame, err := s.galgames.FindByID(ctx, galgameID)
	if err != nil {
		logger.Error("find galgame for gallery batch", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}
	if galgame == nil {
		return nil, ErrGalleryGalgameNotFound
	}

	total, err := s.gallery.CountActiveByGalgameID(ctx, galgameID)
	if err != nil {
		logger.Error("count gallery images", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}
	if total >= MaxGalleryImages {
		return nil, ErrGalleryLimitExceeded
	}
	remaining := MaxGalleryImages - int(total)

	maxSort, err := s.gallery.MaxSortOrder(ctx, galgameID)
	if err != nil {
		logger.Error("max gallery sort order", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}

	result := &dto.BatchGalleryResultData{}
	seen := make(map[string]struct{}, len(req.Items))
	err = s.gallery.Transaction(ctx, func(tx *repository.GalleryRepository) error {
		for i := range req.Items {
			item := &req.Items[i]
			rawURL := strings.TrimSpace(item.ExternalURL)
			if validateExternalImageURL(rawURL) != nil {
				result.Failed++
				continue
			}
			if _, dup := seen[rawURL]; dup {
				result.Skipped++
				continue
			}
			existing, err := tx.FindByGalgameIDAndExternalURL(ctx, galgameID, rawURL)
			if err != nil {
				return err
			}
			if existing != nil {
				seen[rawURL] = struct{}{}
				result.Skipped++
				continue
			}
			if result.Created >= remaining {
				result.Skipped++
				continue
			}
			maxSort++
			image := buildImage(galgameID, actorID, model.GallerySourceExternal, nil, rawURL,
				item.Title, "", item.ImageType, item.IsSpoiler, maxSort)
			if err := tx.Create(ctx, image); err != nil {
				return err
			}
			seen[rawURL] = struct{}{}
			result.Created++
		}
		return nil
	})
	if err != nil {
		logger.Error("batch create gallery images", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}
	return result, nil
}

// GalleryReviewQuery is the parsed admin review queue filter.
type GalleryReviewQuery struct {
	Status     *int16
	GalgameID  uint
	SourceType *int16
	Page       int
	Limit      int
}

// ListGalleryReviews returns the cross-galgame review queue.
func (s *GalleryService) ListGalleryReviews(ctx context.Context, query GalleryReviewQuery) (dto.GalleryReviewListData, error) {
	page, limit := query.Page, query.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, total, err := s.gallery.ListForReview(ctx, repository.GalleryReviewFilter{
		Status:     query.Status,
		GalgameID:  query.GalgameID,
		SourceType: query.SourceType,
		Page:       page,
		Limit:      limit,
	})
	if err != nil {
		logger.Error("list gallery reviews", zap.Error(err))
		return dto.GalleryReviewListData{}, err
	}
	items := make([]dto.GalleryReviewItemData, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		items = append(items, dto.GalleryReviewItemData{
			ID:                 row.ID,
			GalgameID:          row.GalgameID,
			GalgameTitle:       row.GalgameTitle,
			GalgameSlug:        row.GalgameSlug,
			SourceType:         row.SourceType,
			URL:                dto.GalleryImageURL(&row.GalleryImage, s.images),
			ExternalURL:        row.ExternalURL,
			Title:              row.Title,
			ImageType:          row.ImageType,
			IsSpoiler:          row.IsSpoiler,
			Status:             row.Status,
			RejectReason:       row.RejectReason,
			CreatedByUsername:  row.CreatedByUsername,
			ReviewedByUsername: row.ReviewedByUsername,
			CreatedAt:          row.CreatedAt,
			ReviewedAt:         row.ReviewedAt,
		})
	}
	return dto.GalleryReviewListData{Items: items, Total: total, Page: page, Limit: limit}, nil
}

// ReviewGalleryImagesInput approves or rejects a set of images in one
// transaction. Approvals clear reject_reason; rejects require nothing but may
// carry a reason.
type ReviewGalleryImagesInput struct {
	IDs     []uint
	Approve bool
	Reason  string
	AdminID uint
}

// ReviewGalleryImages applies the review decision and credits the original
// uploader once an image goes published on a published galgame.
func (s *GalleryService) ReviewGalleryImages(ctx context.Context, input ReviewGalleryImagesInput) (int, error) {
	status := model.GalleryImageStatusRejected
	if input.Approve {
		status = model.GalleryImageStatusPublished
	}
	reason := strings.TrimSpace(input.Reason)
	if len(reason) > MaxGalleryReviewReasonLength {
		reason = reason[:MaxGalleryReviewReasonLength]
	}
	if input.Approve {
		reason = ""
	}
	now := time.Now()
	reviewed := 0

	write := func(gallery *repository.GalleryRepository, db *gorm.DB) error {
		images, err := gallery.FindByIDs(ctx, input.IDs)
		if err != nil {
			return err
		}
		if len(images) == 0 {
			return nil
		}
		ids := make([]uint, 0, len(images))
		for i := range images {
			ids = append(ids, images[i].ID)
		}
		affected, err := gallery.ReviewByIDs(ctx, ids, status, input.AdminID, now, reason)
		if err != nil {
			return err
		}
		reviewed = int(affected)

		if !input.Approve || s.contributions == nil {
			return nil
		}
		// Credit the original uploader once per image, only when the image
		// becomes publicly visible on a published galgame.
		publishedGalgame := make(map[uint]bool)
		for i := range images {
			image := &images[i]
			if image.CreatedBy == nil {
				continue
			}
			galgameID := image.GalgameID
			ok, cached := publishedGalgame[galgameID]
			if !cached {
				galgame, err := s.galgames.FindByID(ctx, galgameID)
				if err != nil {
					return err
				}
				ok = galgame != nil && galgame.Status == model.GalgameStatusPublished
				publishedGalgame[galgameID] = ok
			}
			if !ok {
				continue
			}
			exists, err := s.contributions.HasSourceContribution(
				ctx, contributionModel.ContributionSourceGalleryImage, image.ID, db)
			if err != nil {
				return err
			}
			if exists {
				continue
			}
			sourceType, sourceID := contributionSource(contributionModel.ContributionSourceGalleryImage, image.ID)
			if err := s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeGalgame,
				TargetID:   galgameID,
				UserID:     *image.CreatedBy,
				Action:     contributionModel.ContributionActionGallery,
				SourceType: sourceType,
				SourceID:   sourceID,
			}, db); err != nil {
				return err
			}
		}
		return nil
	}

	var err error
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewGalleryRepository(db), db)
		})
	} else {
		err = s.gallery.Transaction(ctx, func(tx *repository.GalleryRepository) error {
			return write(tx, nil)
		})
	}
	if err != nil {
		logger.Error("review gallery images", zap.Uints("ids", input.IDs), zap.Error(err))
		return 0, err
	}
	return reviewed, nil
}

// FindGalleryImageForReview loads a single gallery image (any galgame, any
// status) so the review endpoints can return the updated entity.
func (s *GalleryService) FindGalleryImageForReview(ctx context.Context, id uint) (*dto.GalleryImageData, error) {
	image, err := s.gallery.FindByID(ctx, id)
	if err != nil {
		logger.Error("find gallery image", zap.Uint("gallery_id", id), zap.Error(err))
		return nil, err
	}
	if image == nil {
		return nil, nil
	}
	data := dto.NewGalleryImageData(image, s.images)
	return &data, nil
}

// UpdateGalleryImage edits title, description, image_type and is_spoiler.
// galgame_id, asset_id and created_by are immutable here.
func (s *GalleryService) UpdateGalleryImage(
	ctx context.Context,
	galgameID, galleryID uint,
	req *dto.UpdateGalleryImageRequest,
	actorIDs ...uint,
) (*dto.GalleryImageData, error) {
	image, err := s.findGalgameImage(ctx, galgameID, galleryID)
	if err != nil {
		return nil, err
	}

	before := *image
	if req.Title != nil {
		image.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		image.Description = strings.TrimSpace(*req.Description)
	}
	if req.ImageType != nil {
		image.ImageType = *req.ImageType
	}
	if req.IsSpoiler != nil {
		image.IsSpoiler = *req.IsSpoiler
	}
	changed := before.Title != image.Title || before.Description != image.Description ||
		before.ImageType != image.ImageType || before.IsSpoiler != image.IsSpoiler
	galgame, err := s.galgames.FindByID(ctx, galgameID)
	if err != nil {
		return nil, err
	}
	write := func(gallery *repository.GalleryRepository, db *gorm.DB) error {
		if err := gallery.Update(ctx, image); err != nil {
			return err
		}
		if changed && galgame != nil && galgame.Status == model.GalgameStatusPublished && len(actorIDs) > 0 && s.contributions != nil {
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeGalgame,
				TargetID:   galgameID,
				UserID:     actorIDs[0],
				Action:     contributionModel.ContributionActionGallery,
			}, db)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewGalleryRepository(db), db)
		})
	} else {
		err = s.gallery.Update(ctx, image)
	}
	if err != nil {
		logger.Error("update gallery image", zap.Uint("gallery_id", galleryID), zap.Error(err))
		return nil, err
	}
	data := dto.NewGalleryImageData(image, s.images)
	return &data, nil
}

// DeleteGalleryImage removes only the gallery relation. The asset and its R2
// object are left untouched because other content may reference them.
func (s *GalleryService) DeleteGalleryImage(ctx context.Context, galgameID, galleryID uint, actorIDs ...uint) error {
	image, err := s.findGalgameImage(ctx, galgameID, galleryID)
	if err != nil {
		return err
	}
	galgame, err := s.galgames.FindByID(ctx, galgameID)
	if err != nil {
		return err
	}
	write := func(gallery *repository.GalleryRepository, db *gorm.DB) error {
		if err := gallery.Delete(ctx, image.ID); err != nil {
			return err
		}
		if galgame != nil && galgame.Status == model.GalgameStatusPublished && len(actorIDs) > 0 && s.contributions != nil {
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeGalgame,
				TargetID:   galgameID,
				UserID:     actorIDs[0],
				Action:     contributionModel.ContributionActionGallery,
			}, db)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewGalleryRepository(db), db)
		})
	} else {
		err = s.gallery.Delete(ctx, image.ID)
	}
	if err != nil {
		logger.Error("delete gallery image", zap.Uint("gallery_id", galleryID), zap.Error(err))
		return err
	}
	return nil
}

// ReorderGallery rewrites sort_order so ids map to 0..n-1. The id set must
// exactly match the galgame's gallery to avoid stale sort_order leftovers.
func (s *GalleryService) ReorderGallery(ctx context.Context, galgameID uint, req *dto.ReorderGalleryRequest, actorIDs ...uint) error {
	galgame, err := s.galgames.FindByID(ctx, galgameID)
	if err != nil {
		logger.Error("find galgame for gallery reorder", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return err
	}
	if galgame == nil {
		return ErrGalleryGalgameNotFound
	}

	images, err := s.gallery.ListByGalgameID(ctx, galgameID)
	if err != nil {
		logger.Error("list gallery images for reorder", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return err
	}
	if len(req.IDs) != len(images) {
		return ErrGalleryInvalidReorder
	}

	known := make(map[uint]struct{}, len(images))
	for i := range images {
		known[images[i].ID] = struct{}{}
	}
	seen := make(map[uint]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		if _, ok := known[id]; !ok {
			return ErrGalleryInvalidReorder
		}
		if _, dup := seen[id]; dup {
			return ErrGalleryInvalidReorder
		}
		seen[id] = struct{}{}
	}

	changed := false
	for index, image := range images {
		if req.IDs[index] != image.ID {
			changed = true
			break
		}
	}
	write := func(gallery *repository.GalleryRepository, db *gorm.DB) error {
		if err := gallery.UpdateOrder(ctx, galgameID, req.IDs); err != nil {
			return err
		}
		if changed && galgame.Status == model.GalgameStatusPublished && len(actorIDs) > 0 && s.contributions != nil {
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeGalgame,
				TargetID:   galgameID,
				UserID:     actorIDs[0],
				Action:     contributionModel.ContributionActionGallery,
			}, db)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewGalleryRepository(db), db)
		})
	} else {
		err = s.gallery.Transaction(ctx, func(tx *repository.GalleryRepository) error {
			return write(tx, nil)
		})
	}
	if err != nil {
		logger.Error("reorder gallery images", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return err
	}
	return nil
}

// findGalgameImage loads a gallery image scoped to its galgame so a valid
// galleryID under another galgame is rejected.
func (s *GalleryService) findGalgameImage(
	ctx context.Context,
	galgameID, galleryID uint,
) (*model.GalleryImage, error) {
	image, err := s.gallery.FindByID(ctx, galleryID)
	if err != nil {
		logger.Error("find gallery image", zap.Uint("gallery_id", galleryID), zap.Error(err))
		return nil, err
	}
	if image == nil || image.GalgameID != galgameID {
		return nil, ErrGalleryImageNotFound
	}
	return image, nil
}
