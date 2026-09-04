package service

import (
	"context"
	"errors"
	"strings"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	imageService "backend/internal/image/service"
	"backend/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrGalleryGalgameNotFound = errors.New("galgame not found")
	ErrGalleryImageNotFound   = errors.New("gallery image not found")
	ErrGalleryAssetNotFound   = errors.New("gallery asset not found or not an image")
	ErrGalleryAssetDuplicate  = errors.New("asset already exists in this gallery")
	ErrGalleryLimitExceeded   = errors.New("gallery image limit exceeded")
	ErrGalleryInvalidReorder  = errors.New("gallery reorder ids must exactly cover all gallery images of this galgame")
)

// MaxGalleryImages caps how many images one galgame gallery may hold.
const MaxGalleryImages = 30

type GalleryService struct {
	galgames      *repository.GalgameRepository
	gallery       *repository.GalleryRepository
	images        *imageService.ImageAssetService
	contributions *ContributionService
}

func (s *GalleryService) SetContributionService(contributions *ContributionService) {
	s.contributions = contributions
}

func NewGalleryService(
	galgames *repository.GalgameRepository,
	gallery *repository.GalleryRepository,
	images *imageService.ImageAssetService,
) *GalleryService {
	return &GalleryService{galgames: galgames, gallery: gallery, images: images}
}

// ListPublishedGallery returns the gallery of a published galgame.
func (s *GalleryService) ListPublishedGallery(ctx context.Context, galgameID uint) (dto.GalleryListData, error) {
	galgame, err := s.galgames.FindPublishedByID(ctx, galgameID)
	if err != nil {
		logger.Error("find galgame for gallery", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return dto.GalleryListData{}, err
	}
	if galgame == nil {
		return dto.GalleryListData{}, ErrGalleryGalgameNotFound
	}
	images, err := s.gallery.ListByGalgameID(ctx, galgameID)
	if err != nil {
		logger.Error("list gallery images", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return dto.GalleryListData{}, err
	}
	return dto.NewGalleryImageList(images, s.images), nil
}

// ListAdminGallery returns the gallery of a galgame in any status.
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
	return dto.NewGalleryImageList(images, s.images), nil
}

// CreateGalleryImage validates the galgame and asset, enforces the limit and
// uniqueness, then appends the image at the end of the gallery.
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

	asset, err := s.images.GetImage(ctx, req.AssetID)
	if err != nil {
		if errors.Is(err, imageService.ErrImageNotFound) {
			return nil, ErrGalleryAssetNotFound
		}
		return nil, err
	}
	if !strings.HasPrefix(asset.MimeType, "image/") {
		return nil, ErrGalleryAssetNotFound
	}

	total, err := s.gallery.CountByGalgameID(ctx, galgameID)
	if err != nil {
		logger.Error("count gallery images", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}
	if total >= MaxGalleryImages {
		return nil, ErrGalleryLimitExceeded
	}

	existing, err := s.gallery.FindByGalgameIDAndAssetID(ctx, galgameID, req.AssetID)
	if err != nil {
		logger.Error("find duplicate gallery image", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}
	if existing != nil {
		return nil, ErrGalleryAssetDuplicate
	}

	maxSort, err := s.gallery.MaxSortOrder(ctx, galgameID)
	if err != nil {
		logger.Error("max gallery sort order", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}

	image := &model.GalleryImage{
		GalgameID:   galgameID,
		AssetID:     req.AssetID,
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		SortOrder:   maxSort + 1,
		ImageType:   req.ImageType,
		IsSpoiler:   req.IsSpoiler,
		CreatedBy:   &actorID,
	}
	write := func(gallery *repository.GalleryRepository, db *gorm.DB) error {
		if err := gallery.Create(ctx, image); err != nil {
			return err
		}
		if galgame.Status == model.GalgameStatusPublished && s.contributions != nil {
			sourceType, sourceID := contributionSource(model.ContributionSourceGalleryImage, image.ID)
			return s.contributions.RecordContribution(ctx, RecordContributionInput{
				GalgameID:  galgameID,
				UserID:     actorID,
				Action:     model.ContributionActionGallery,
				SourceType: sourceType,
				SourceID:   sourceID,
			}, db)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewGalleryRepository(db), db)
		})
	} else {
		err = s.gallery.Create(ctx, image)
	}
	if err != nil {
		logger.Error("create gallery image",
			zap.Uint("galgame_id", galgameID), zap.Uint("asset_id", req.AssetID), zap.Error(err))
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
			return s.contributions.RecordContribution(ctx, RecordContributionInput{
				GalgameID: galgameID,
				UserID:    actorIDs[0],
				Action:    model.ContributionActionGallery,
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
			return s.contributions.RecordContribution(ctx, RecordContributionInput{
				GalgameID: galgameID,
				UserID:    actorIDs[0],
				Action:    model.ContributionActionGallery,
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
			return s.contributions.RecordContribution(ctx, RecordContributionInput{
				GalgameID: galgameID,
				UserID:    actorIDs[0],
				Action:    model.ContributionActionGallery,
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
