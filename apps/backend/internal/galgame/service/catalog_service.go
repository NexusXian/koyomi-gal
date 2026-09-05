package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	contributionModel "backend/internal/contribution/model"
	contributionService "backend/internal/contribution/service"
	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	notificationModel "backend/internal/notification/model"
	notificationService "backend/internal/notification/service"
	rbacService "backend/internal/rbac/service"
	relationModel "backend/internal/relation/model"
	relationRepository "backend/internal/relation/repository"
	userModel "backend/internal/user/model"
	userService "backend/internal/user/service"
	"backend/pkg/logger"

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrDeveloperNotFound    = errors.New("developer not found")
	ErrDeveloperSlugExists  = errors.New("developer slug already exists")
	ErrTagNotFound          = errors.New("tag not found")
	ErrTagNameExists        = errors.New("tag name already exists")
	ErrTagSlugExists        = errors.New("tag slug already exists")
	ErrGalgameNotFound      = errors.New("galgame not found")
	ErrGalgameSlugExists    = errors.New("galgame slug already exists")
	ErrUnknownTagIDs        = errors.New("tag ids contain unknown tag")
	ErrInvalidReleaseDate   = errors.New("invalid release date")
	ErrInvalidAgeRating     = errors.New("invalid age rating")
	ErrInvalidStatus        = errors.New("invalid galgame status")
	ErrInvalidSort          = errors.New("invalid galgame sort")
	ErrInvalidReleaseRange  = errors.New("invalid release year range")
	ErrInvalidCatalogInput  = errors.New("invalid catalog input")
	ErrInvalidMyGalgameType = errors.New("invalid my galgame type")
)

type CatalogService struct {
	galgames      *repository.GalgameRepository
	developers    *repository.DeveloperRepository
	tags          *repository.TagRepository
	relations     *relationRepository.RelationRepository
	contributions *contributionService.ContributionService
	rbac          *rbacService.RBACService
	notifications *notificationService.NotificationService
	activities    userService.ActivityRecorder
}

func (s *CatalogService) SetContributionService(contributions *contributionService.ContributionService) {
	s.contributions = contributions
}

func (s *CatalogService) SetRelationRepository(relations *relationRepository.RelationRepository) {
	s.relations = relations
}

func (s *CatalogService) SetActivityRecorder(recorder userService.ActivityRecorder) {
	s.activities = recorder
}

func (s *CatalogService) SetNotificationDependencies(
	rbac *rbacService.RBACService,
	notifications *notificationService.NotificationService,
) {
	s.rbac = rbac
	s.notifications = notifications
}

func NewCatalogService(
	galgames *repository.GalgameRepository,
	developers *repository.DeveloperRepository,
	tags *repository.TagRepository,
) *CatalogService {
	return &CatalogService{galgames: galgames, developers: developers, tags: tags}
}

func (s *CatalogService) CreateDeveloper(
	ctx context.Context,
	req *dto.CreateDeveloperRequest,
) (*model.Developer, error) {
	name := strings.TrimSpace(req.Name)
	slug := normalizeSlug(req.Slug)
	if name == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	existing, err := s.developers.FindBySlug(ctx, slug)
	if err != nil {
		logger.Error("find developer by slug", zap.String("slug", slug), zap.Error(err))
		return nil, err
	}
	if existing != nil {
		return nil, ErrDeveloperSlugExists
	}

	developer := &model.Developer{
		Name:         name,
		OriginalName: strings.TrimSpace(req.OriginalName),
		Slug:         slug,
		Description:  strings.TrimSpace(req.Description),
		LogoURL:      strings.TrimSpace(req.LogoURL),
		Website:      strings.TrimSpace(req.Website),
	}
	if err := s.developers.Create(ctx, developer); err != nil {
		if hasConstraint(err, "developers_slug_unique") {
			return nil, ErrDeveloperSlugExists
		}
		logger.Error("create developer", zap.String("slug", slug), zap.Error(err))
		return nil, err
	}
	return developer, nil
}

func (s *CatalogService) UpdateDeveloper(
	ctx context.Context,
	id uint,
	req *dto.UpdateDeveloperRequest,
) (*model.Developer, error) {
	developer, err := s.developers.FindByID(ctx, id)
	if err != nil {
		logger.Error("find developer by id", zap.Uint("developer_id", id), zap.Error(err))
		return nil, err
	}
	if developer == nil {
		return nil, ErrDeveloperNotFound
	}

	name := strings.TrimSpace(req.Name)
	slug := normalizeSlug(req.Slug)
	if name == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	existing, err := s.developers.FindBySlug(ctx, slug)
	if err != nil {
		logger.Error("find developer by slug", zap.String("slug", slug), zap.Error(err))
		return nil, err
	}
	if existing != nil && existing.ID != id {
		return nil, ErrDeveloperSlugExists
	}

	developer.Name = name
	developer.OriginalName = strings.TrimSpace(req.OriginalName)
	developer.Slug = slug
	developer.Description = strings.TrimSpace(req.Description)
	developer.LogoURL = strings.TrimSpace(req.LogoURL)
	developer.Website = strings.TrimSpace(req.Website)
	if err := s.developers.Update(ctx, developer); err != nil {
		if hasConstraint(err, "developers_slug_unique") {
			return nil, ErrDeveloperSlugExists
		}
		logger.Error("update developer", zap.Uint("developer_id", id), zap.Error(err))
		return nil, err
	}
	return developer, nil
}

func (s *CatalogService) ListDevelopers(ctx context.Context) ([]model.Developer, error) {
	return s.developers.List(ctx)
}

// GetDeveloper returns a developer by ID for the public detail endpoint.
func (s *CatalogService) GetDeveloper(ctx context.Context, id uint) (*model.Developer, error) {
	developer, err := s.developers.FindByID(ctx, id)
	if err != nil {
		logger.Error("find developer by id", zap.Uint("developer_id", id), zap.Error(err))
		return nil, err
	}
	if developer == nil {
		return nil, ErrDeveloperNotFound
	}
	return developer, nil
}

func (s *CatalogService) CreateTag(ctx context.Context, req *dto.CreateTagRequest) (*model.Tag, error) {
	name := strings.TrimSpace(req.Name)
	slug := normalizeSlug(req.Slug)
	if name == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	if err := s.ensureTagUnique(ctx, 0, name, slug); err != nil {
		return nil, err
	}

	tag := &model.Tag{
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(req.Description),
	}
	if err := s.tags.Create(ctx, tag); err != nil {
		switch {
		case hasConstraint(err, "tags_name_unique"):
			return nil, ErrTagNameExists
		case hasConstraint(err, "tags_slug_unique"):
			return nil, ErrTagSlugExists
		}
		logger.Error("create tag", zap.String("slug", slug), zap.Error(err))
		return nil, err
	}
	return tag, nil
}

func (s *CatalogService) UpdateTag(
	ctx context.Context,
	id uint,
	req *dto.UpdateTagRequest,
) (*model.Tag, error) {
	tag, err := s.tags.FindByID(ctx, id)
	if err != nil {
		logger.Error("find tag by id", zap.Uint("tag_id", id), zap.Error(err))
		return nil, err
	}
	if tag == nil {
		return nil, ErrTagNotFound
	}

	name := strings.TrimSpace(req.Name)
	slug := normalizeSlug(req.Slug)
	if name == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	if err := s.ensureTagUnique(ctx, id, name, slug); err != nil {
		return nil, err
	}
	tag.Name = name
	tag.Slug = slug
	tag.Description = strings.TrimSpace(req.Description)
	if err := s.tags.Update(ctx, tag); err != nil {
		switch {
		case hasConstraint(err, "tags_name_unique"):
			return nil, ErrTagNameExists
		case hasConstraint(err, "tags_slug_unique"):
			return nil, ErrTagSlugExists
		}
		logger.Error("update tag", zap.Uint("tag_id", id), zap.Error(err))
		return nil, err
	}
	return tag, nil
}

func (s *CatalogService) ListTags(ctx context.Context) ([]model.Tag, error) {
	return s.tags.List(ctx)
}

// GetTag returns a tag by ID for the public detail endpoint.
func (s *CatalogService) GetTag(ctx context.Context, id uint) (*model.Tag, error) {
	tag, err := s.tags.FindByID(ctx, id)
	if err != nil {
		logger.Error("find tag by id", zap.Uint("tag_id", id), zap.Error(err))
		return nil, err
	}
	if tag == nil {
		return nil, ErrTagNotFound
	}
	return tag, nil
}

func (s *CatalogService) CreateGalgame(
	ctx context.Context,
	userID uint,
	req *dto.CreateGalgameRequest,
) (*model.Galgame, error) {
	title := strings.TrimSpace(req.Title)
	slug := normalizeSlug(req.Slug)
	if title == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	releaseDate, err := parseReleaseDate(req.ReleaseDate)
	if err != nil {
		return nil, err
	}
	if !validAgeRating(req.AgeRating) {
		return nil, ErrInvalidAgeRating
	}
	if !validStatus(req.Status) {
		return nil, ErrInvalidStatus
	}

	tagIDs := uniqueUint(req.TagIDs)
	if err := s.validateRelations(ctx, req.DeveloperID, tagIDs); err != nil {
		return nil, err
	}
	if err := s.ensureGalgameSlugUnique(ctx, 0, slug); err != nil {
		return nil, err
	}

	description := strings.TrimSpace(req.Description)
	galgame := &model.Galgame{
		Title:             title,
		OriginalTitle:     strings.TrimSpace(req.OriginalTitle),
		RomajiTitle:       strings.TrimSpace(req.RomajiTitle),
		Slug:              slug,
		Description:       description,
		DescriptionSource: descriptionSourceForEdit(description),
		CoverURL:          strings.TrimSpace(req.CoverURL),
		BannerURL:         strings.TrimSpace(req.BannerURL),
		DeveloperID:       req.DeveloperID,
		ReleaseDate:       releaseDate,
		AgeRating:         req.AgeRating,
		CoverSensitive:    req.CoverSensitive,
		Status:            req.Status,
		CreatedBy:         &userID,
	}
	aliases := uniqueNonEmptyStrings(req.Aliases)
	write := func(tx *repository.GalgameRepository, db *gorm.DB) error {
		if err := tx.Create(ctx, galgame); err != nil {
			return err
		}
		if err := tx.ReplaceAliases(ctx, galgame.ID, aliases); err != nil {
			return err
		}
		if err := tx.ReplaceTags(ctx, galgame.ID, tagIDs); err != nil {
			return err
		}
		if galgame.Status == model.GalgameStatusPublished && s.contributions != nil {
			sourceType, sourceID := contributionSource(contributionModel.ContributionSourceGalgameCreate, galgame.ID)
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeGalgame,
				TargetID:   galgame.ID,
				UserID:     userID,
				Action:     contributionModel.ContributionActionCreate,
				SourceType: sourceType,
				SourceID:   sourceID,
			}, db)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewGalgameRepository(db), db)
		})
	} else {
		err = s.galgames.Transaction(ctx, func(tx *repository.GalgameRepository) error {
			return write(tx, nil)
		})
	}
	if err != nil {
		if hasConstraint(err, "galgames_slug_unique") {
			return nil, ErrGalgameSlugExists
		}
		logger.Error("create galgame", zap.String("slug", slug), zap.Error(err))
		return nil, err
	}
	created, err := s.getGalgame(ctx, galgame.ID, false)
	if err != nil {
		return nil, err
	}
	if created.Status == model.GalgameStatusPending {
		s.notifyGalgameSubmitted(ctx, userID, created)
	}
	return created, nil
}

func (s *CatalogService) UpdateGalgame(
	ctx context.Context,
	id uint,
	req *dto.UpdateGalgameRequest,
	actorIDs ...uint,
) (*model.Galgame, error) {
	title := strings.TrimSpace(req.Title)
	slug := normalizeSlug(req.Slug)
	if title == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	galgame, err := s.galgames.FindByID(ctx, id)
	if err != nil {
		logger.Error("find galgame by id", zap.Uint("galgame_id", id), zap.Error(err))
		return nil, err
	}
	if galgame == nil {
		return nil, ErrGalgameNotFound
	}
	if req.AgeRating == nil || !validAgeRating(*req.AgeRating) {
		return nil, ErrInvalidAgeRating
	}
	if req.CoverSensitive == nil {
		return nil, ErrInvalidCatalogInput
	}
	if req.Status == nil || !validStatus(*req.Status) {
		return nil, ErrInvalidStatus
	}
	releaseDate, err := parseReleaseDate(req.ReleaseDate)
	if err != nil {
		return nil, err
	}
	tagIDs := uniqueUint(req.TagIDs)
	if err := s.validateRelations(ctx, req.DeveloperID, tagIDs); err != nil {
		return nil, err
	}
	if err := s.ensureGalgameSlugUnique(ctx, id, slug); err != nil {
		return nil, err
	}
	oldStatus := galgame.Status
	aliases := uniqueNonEmptyStrings(req.Aliases)
	changed, coverOnly := galgameUpdateChanges(galgame, req, title, slug, releaseDate, aliases, tagIDs)

	description := strings.TrimSpace(req.Description)
	descriptionChanged := galgame.Description != description

	galgame.Title = title
	galgame.OriginalTitle = strings.TrimSpace(req.OriginalTitle)
	galgame.RomajiTitle = strings.TrimSpace(req.RomajiTitle)
	galgame.Slug = slug
	galgame.Description = description
	if descriptionChanged {
		galgame.DescriptionSource = descriptionSourceForEdit(description)
	}
	galgame.CoverURL = strings.TrimSpace(req.CoverURL)
	galgame.BannerURL = strings.TrimSpace(req.BannerURL)
	galgame.DeveloperID = req.DeveloperID
	galgame.ReleaseDate = releaseDate
	galgame.AgeRating = *req.AgeRating
	galgame.CoverSensitive = *req.CoverSensitive
	galgame.Status = *req.Status
	write := func(tx *repository.GalgameRepository, db *gorm.DB) error {
		if err := tx.Update(ctx, galgame); err != nil {
			return err
		}
		if err := tx.ReplaceAliases(ctx, id, aliases); err != nil {
			return err
		}
		if err := tx.ReplaceTags(ctx, id, tagIDs); err != nil {
			return err
		}
		if changed && galgame.Status == model.GalgameStatusPublished && s.contributions != nil {
			if oldStatus != model.GalgameStatusPublished {
				contributorID := uint(0)
				if galgame.CreatedBy != nil {
					contributorID = *galgame.CreatedBy
				} else if len(actorIDs) > 0 {
					contributorID = actorIDs[0]
				}
				if contributorID == 0 {
					return s.recordInitialGalleryContributions(ctx, db, id)
				}
				sourceType, sourceID := contributionSource(contributionModel.ContributionSourceGalgameCreate, id)
				if err := s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
					TargetType: relationModel.WorkTypeGalgame,
					TargetID:   id,
					UserID:     contributorID,
					Action:     contributionModel.ContributionActionCreate,
					SourceType: sourceType,
					SourceID:   sourceID,
				}, db); err != nil {
					return err
				}
				return s.recordInitialGalleryContributions(ctx, db, id)
			}
			if len(actorIDs) == 0 {
				return nil
			}
			action := contributionModel.ContributionActionEdit
			if coverOnly {
				action = contributionModel.ContributionActionCover
			}
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeGalgame,
				TargetID:   id,
				UserID:     actorIDs[0],
				Action:     action,
			}, db)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewGalgameRepository(db), db)
		})
	} else {
		err = s.galgames.Transaction(ctx, func(tx *repository.GalgameRepository) error {
			return write(tx, nil)
		})
	}
	if err != nil {
		if hasConstraint(err, "galgames_slug_unique") {
			return nil, ErrGalgameSlugExists
		}
		logger.Error("update galgame", zap.Uint("galgame_id", id), zap.Error(err))
		return nil, err
	}
	return s.getGalgame(ctx, id, false)
}

func (s *CatalogService) ReviewGalgame(
	ctx context.Context,
	actorID, id uint,
	req *dto.ReviewGalgameRequest,
) (*model.Galgame, error) {
	if req.Status != model.GalgameStatusPublished && req.Status != model.GalgameStatusRejected {
		return nil, ErrInvalidStatus
	}
	galgame, err := s.galgames.FindByID(ctx, id)
	if err != nil {
		logger.Error("find galgame by id", zap.Uint("galgame_id", id), zap.Error(err))
		return nil, err
	}
	if galgame == nil {
		return nil, ErrGalgameNotFound
	}
	if galgame.Status == req.Status {
		return galgame, nil
	}

	if req.Status == model.GalgameStatusPublished && s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			if err := repository.NewGalgameRepository(db).UpdateStatus(ctx, id, req.Status); err != nil {
				return err
			}
			if galgame.CreatedBy != nil {
				sourceType, sourceID := contributionSource(contributionModel.ContributionSourceGalgameCreate, id)
				if err := s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
					TargetType: relationModel.WorkTypeGalgame,
					TargetID:   id,
					UserID:     *galgame.CreatedBy,
					Action:     contributionModel.ContributionActionCreate,
					SourceType: sourceType,
					SourceID:   sourceID,
				}, db); err != nil {
					return err
				}
			}
			return s.recordInitialGalleryContributions(ctx, db, id)
		})
	} else {
		err = s.galgames.UpdateStatus(ctx, id, req.Status)
	}
	if err != nil {
		logger.Error("review galgame", zap.Uint("galgame_id", id), zap.Uint("actor_id", actorID), zap.Error(err))
		return nil, err
	}
	galgame, err = s.galgames.FindByID(ctx, id)
	if err != nil {
		logger.Error("find reviewed galgame", zap.Uint("galgame_id", id), zap.Error(err))
		return nil, err
	}
	s.notifyGalgameReviewResult(ctx, actorID, galgame)
	if req.Status == model.GalgameStatusPublished && galgame.CreatedBy != nil && s.activities != nil {
		if recordErr := s.activities.Record(ctx, *galgame.CreatedBy, userModel.ActivityReviewApproved, &galgame.ID, map[string]any{"title": galgame.Title}); recordErr != nil {
			logger.Error("record galgame approval activity", zap.Uint("galgame_id", galgame.ID), zap.Error(recordErr))
		}
	}
	return galgame, nil
}

// BatchUpdateGalgame applies the whitelisted age_rating / cover_sensitive
// updates to all matched ids and returns the number of updated rows.
func (s *CatalogService) BatchUpdateGalgame(
	ctx context.Context,
	req *dto.BatchUpdateGalgameRequest,
) (int64, error) {
	if req.AgeRating == nil && req.CoverSensitive == nil {
		return 0, ErrInvalidCatalogInput
	}
	if req.AgeRating != nil && !validAgeRating(*req.AgeRating) {
		return 0, ErrInvalidAgeRating
	}

	updates := map[string]any{"updated_at": time.Now()}
	if req.AgeRating != nil {
		updates["age_rating"] = *req.AgeRating
	}
	if req.CoverSensitive != nil {
		updates["cover_sensitive"] = *req.CoverSensitive
	}

	updated, err := s.galgames.BatchUpdate(ctx, uniqueUint(req.IDs), updates)
	if err != nil {
		logger.Error("batch update galgames", zap.Int("id_count", len(req.IDs)), zap.Error(err))
		return 0, err
	}
	return updated, nil
}

func (s *CatalogService) DeleteGalgame(ctx context.Context, id uint) error {
	galgame, err := s.galgames.FindByID(ctx, id)
	if err != nil {
		logger.Error("find galgame by id", zap.Uint("galgame_id", id), zap.Error(err))
		return err
	}
	if galgame == nil {
		return ErrGalgameNotFound
	}
	if s.relations != nil {
		if err := s.relations.DeleteByWork(ctx, relationModel.WorkTypeGalgame, id); err != nil {
			logger.Error("delete galgame work relations", zap.Uint("galgame_id", id), zap.Error(err))
			return err
		}
	}
	if err := s.galgames.Delete(ctx, id); err != nil {
		logger.Error("delete galgame", zap.Uint("galgame_id", id), zap.Error(err))
		return err
	}
	return nil
}

// BatchDeleteGalgames hard-deletes the matched galgames and returns the number
// of deleted rows. Cascades remove aliases, tags, favorites, ratings, states,
// resources, posts, gallery and contribution rows; work relations are removed
// explicitly beforehand.
func (s *CatalogService) BatchDeleteGalgames(
	ctx context.Context,
	req *dto.BatchDeleteGalgameRequest,
) (int64, error) {
	if s.relations != nil {
		for _, id := range uniqueUint(req.IDs) {
			if err := s.relations.DeleteByWork(ctx, relationModel.WorkTypeGalgame, id); err != nil {
				logger.Error("delete galgame work relations", zap.Uint("galgame_id", id), zap.Error(err))
				return 0, err
			}
		}
	}
	deleted, err := s.galgames.BatchDelete(ctx, uniqueUint(req.IDs))
	if err != nil {
		logger.Error("batch delete galgames", zap.Int("id_count", len(req.IDs)), zap.Error(err))
		return 0, err
	}
	return deleted, nil
}

func (s *CatalogService) GetPublishedGalgame(ctx context.Context, id uint) (*model.Galgame, error) {
	return s.getGalgame(ctx, id, true)
}

// GetGalgame returns a galgame of any status for the galgame:review admin
// detail endpoint.
func (s *CatalogService) GetGalgame(ctx context.Context, id uint) (*model.Galgame, error) {
	return s.getGalgame(ctx, id, false)
}

// ListAllGalgames lists galgames of every status with an optional status
// filter for the galgame:review admin listing.
func (s *CatalogService) ListAllGalgames(
	ctx context.Context,
	query *dto.AdminGalgameQuery,
) ([]model.Galgame, int64, int, int, error) {
	page := query.Page
	if page == 0 {
		page = 1
	}
	limit := query.Limit
	if limit == 0 {
		limit = 20
	}
	sort := strings.ToLower(strings.TrimSpace(query.Sort))
	if sort == "" {
		sort = "latest"
	}
	if !validSort(sort) {
		return nil, 0, page, limit, ErrInvalidSort
	}
	if query.Status != nil && !validStatus(*query.Status) {
		return nil, 0, page, limit, ErrInvalidStatus
	}
	if query.AgeRating != nil && !validAgeRating(*query.AgeRating) {
		return nil, 0, page, limit, ErrInvalidAgeRating
	}
	if err := validateAIFilters(query); err != nil {
		return nil, 0, page, limit, err
	}

	galgames, total, err := s.galgames.ListAdmin(ctx, repository.GalgameListOptions{
		Keyword:          strings.TrimSpace(query.Keyword),
		Status:           query.Status,
		AgeRating:        query.AgeRating,
		CoverSensitive:   query.CoverSensitive,
		AIClassification: query.AIClassification,
		AIStatus:         query.AIStatus,
		AIConflict:       query.AIConflict,
		AIMinConfidence:  query.AIMinConfidence,
		AIMaxConfidence:  query.AIMaxConfidence,
		Sort:             sort,
		Page:             page,
		Limit:            limit,
	})
	if err != nil {
		logger.Error("list all galgames", zap.Error(err))
		return nil, 0, page, limit, err
	}
	return galgames, total, page, limit, nil
}

// ErrInvalidAIFilter reports invalid admin AI classification filters.
var ErrInvalidAIFilter = errors.New("ai filter is invalid")

func validateAIFilters(query *dto.AdminGalgameQuery) error {
	if query.AIMinConfidence != nil && query.AIMaxConfidence != nil &&
		*query.AIMinConfidence > *query.AIMaxConfidence {
		return ErrInvalidAIFilter
	}
	return nil
}

func (s *CatalogService) ListPublishedGalgames(
	ctx context.Context,
	query *dto.GalgameQuery,
) ([]model.Galgame, int64, int, int, error) {
	page := query.Page
	if page == 0 {
		page = 1
	}
	limit := query.Limit
	if limit == 0 {
		limit = 20
	}
	sort := strings.ToLower(strings.TrimSpace(query.Sort))
	if sort == "" {
		sort = "latest"
	}
	if !validSort(sort) {
		return nil, 0, page, limit, ErrInvalidSort
	}
	if query.ReleaseFrom != nil && query.ReleaseTo != nil && *query.ReleaseFrom > *query.ReleaseTo {
		return nil, 0, page, limit, ErrInvalidReleaseRange
	}
	if query.AgeRating != nil && !validAgeRating(*query.AgeRating) {
		return nil, 0, page, limit, ErrInvalidAgeRating
	}

	galgames, total, err := s.galgames.ListPublished(ctx, repository.GalgameListOptions{
		Keyword:     strings.TrimSpace(query.Keyword),
		DeveloperID: query.DeveloperID,
		TagIDs:      uniqueUint(query.TagIDs),
		ReleaseFrom: query.ReleaseFrom,
		ReleaseTo:   query.ReleaseTo,
		AgeRating:   query.AgeRating,
		Sort:        sort,
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		logger.Error("list published galgames", zap.Error(err))
		return nil, 0, page, limit, err
	}
	return galgames, total, page, limit, nil
}

func (s *CatalogService) ListMyGalgames(
	ctx context.Context,
	userID uint,
	query *dto.MyGalgameQuery,
) ([]model.Galgame, int64, int, int, error) {
	page := query.Page
	if page == 0 {
		page = 1
	}
	limit := query.Limit
	if limit == 0 {
		limit = 20
	}
	collectionType := strings.ToLower(strings.TrimSpace(query.Type))
	if collectionType == "" {
		collectionType = "uploaded"
	}

	var (
		galgames []model.Galgame
		total    int64
		err      error
	)
	switch collectionType {
	case "uploaded":
		galgames, total, err = s.galgames.ListByCreator(ctx, userID, page, limit)
	case "favorite":
		galgames, total, err = s.galgames.ListFavoritesByUser(ctx, userID, page, limit)
	default:
		return nil, 0, page, limit, ErrInvalidMyGalgameType
	}
	if err != nil {
		logger.Error("list my galgames", zap.Uint("user_id", userID), zap.Error(err))
		return nil, 0, page, limit, err
	}
	return galgames, total, page, limit, nil
}

func (s *CatalogService) getGalgame(
	ctx context.Context,
	id uint,
	publishedOnly bool,
) (*model.Galgame, error) {
	var (
		galgame *model.Galgame
		err     error
	)
	if publishedOnly {
		galgame, err = s.galgames.FindPublishedByID(ctx, id)
	} else {
		galgame, err = s.galgames.FindByID(ctx, id)
	}
	if err != nil {
		logger.Error("find galgame by id", zap.Uint("galgame_id", id), zap.Error(err))
		return nil, err
	}
	if galgame == nil {
		return nil, ErrGalgameNotFound
	}
	if s.contributions != nil {
		contributors, total, err := s.contributions.ListContributorRows(ctx, relationModel.WorkTypeGalgame, id, 1, 10)
		if err != nil {
			return nil, err
		}
		galgame.Contributors = contributors
		galgame.ContributorCount = total
	}
	if s.relations != nil {
		related, err := s.relations.ListRelatedNovelsForGalgame(ctx, id)
		if err != nil {
			logger.Error("list related novels for galgame", zap.Uint("galgame_id", id), zap.Error(err))
			return nil, err
		}
		galgame.RelatedNovels = related
	}
	return galgame, nil
}

func contributionSource(sourceType string, sourceID uint) (*string, *uint) {
	return &sourceType, &sourceID
}

func (s *CatalogService) recordInitialGalleryContributions(ctx context.Context, db *gorm.DB, galgameID uint) error {
	galleryImages, err := repository.NewGalleryRepository(db).ListByGalgameID(ctx, galgameID)
	if err != nil {
		return err
	}
	for _, image := range galleryImages {
		// Pending/rejected images are credited when (and only if) their own
		// review approves them, so skip them here to avoid double credit.
		if image.Status != model.GalleryImageStatusPublished {
			continue
		}
		if image.CreatedBy == nil {
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

// descriptionSourceForEdit records human-submitted descriptions as manual so
// automatic enrichment can no longer overwrite them. Clearing a description
// resets the source to unknown instead.
func descriptionSourceForEdit(description string) string {
	if description == "" {
		return model.DescriptionSourceUnknown
	}
	return model.DescriptionSourceManual
}

func galgameUpdateChanges(
	galgame *model.Galgame,
	req *dto.UpdateGalgameRequest,
	title, slug string,
	releaseDate *time.Time,
	aliases []string,
	tagIDs []uint,
) (bool, bool) {
	coverChanged := galgame.CoverURL != strings.TrimSpace(req.CoverURL) ||
		galgame.BannerURL != strings.TrimSpace(req.BannerURL)
	otherChanged := galgame.Title != title ||
		galgame.OriginalTitle != strings.TrimSpace(req.OriginalTitle) ||
		galgame.RomajiTitle != strings.TrimSpace(req.RomajiTitle) ||
		galgame.Slug != slug ||
		galgame.Description != strings.TrimSpace(req.Description) ||
		!equalUintPointers(galgame.DeveloperID, req.DeveloperID) ||
		!equalTimePointers(galgame.ReleaseDate, releaseDate) ||
		galgame.AgeRating != *req.AgeRating ||
		galgame.CoverSensitive != *req.CoverSensitive ||
		galgame.Status != *req.Status ||
		!equalAliases(galgame.Aliases, aliases) ||
		!equalTagIDs(galgame.Tags, tagIDs)
	return coverChanged || otherChanged, coverChanged && !otherChanged
}

func equalUintPointers(left, right *uint) bool {
	return (left == nil && right == nil) ||
		(left != nil && right != nil && *left == *right)
}

func equalTimePointers(left, right *time.Time) bool {
	return (left == nil && right == nil) ||
		(left != nil && right != nil && left.Equal(*right))
}

func equalAliases(current []model.Alias, expected []string) bool {
	if len(current) != len(expected) {
		return false
	}
	counts := make(map[string]int, len(current))
	for _, alias := range current {
		counts[alias.Alias]++
	}
	for _, alias := range expected {
		counts[alias]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

func equalTagIDs(current []model.Tag, expected []uint) bool {
	if len(current) != len(expected) {
		return false
	}
	ids := make(map[uint]struct{}, len(current))
	for _, tag := range current {
		ids[tag.ID] = struct{}{}
	}
	for _, id := range expected {
		if _, ok := ids[id]; !ok {
			return false
		}
	}
	return true
}

func (s *CatalogService) validateRelations(ctx context.Context, developerID *uint, tagIDs []uint) error {
	if developerID != nil {
		developer, err := s.developers.FindByID(ctx, *developerID)
		if err != nil {
			logger.Error("find developer by id", zap.Uint("developer_id", *developerID), zap.Error(err))
			return err
		}
		if developer == nil {
			return ErrDeveloperNotFound
		}
	}
	if len(tagIDs) == 0 {
		return nil
	}
	count, err := s.tags.CountByIDs(ctx, tagIDs)
	if err != nil {
		logger.Error("count tags by ids", zap.Error(err))
		return err
	}
	if count != int64(len(tagIDs)) {
		return ErrUnknownTagIDs
	}
	return nil
}

func (s *CatalogService) ensureGalgameSlugUnique(ctx context.Context, id uint, slug string) error {
	existing, err := s.galgames.FindBySlug(ctx, slug)
	if err != nil {
		logger.Error("find galgame by slug", zap.String("slug", slug), zap.Error(err))
		return err
	}
	if existing != nil && existing.ID != id {
		return ErrGalgameSlugExists
	}
	return nil
}

func (s *CatalogService) ensureTagUnique(ctx context.Context, id uint, name, slug string) error {
	existing, err := s.tags.FindByName(ctx, name)
	if err != nil {
		logger.Error("find tag by name", zap.String("name", name), zap.Error(err))
		return err
	}
	if existing != nil && existing.ID != id {
		return ErrTagNameExists
	}
	existing, err = s.tags.FindBySlug(ctx, slug)
	if err != nil {
		logger.Error("find tag by slug", zap.String("slug", slug), zap.Error(err))
		return err
	}
	if existing != nil && existing.ID != id {
		return ErrTagSlugExists
	}
	return nil
}

func (s *CatalogService) notifyGalgameSubmitted(ctx context.Context, actorID uint, galgame *model.Galgame) {
	if s.rbac == nil || s.notifications == nil {
		return
	}
	recipientIDs, err := s.rbac.FindUserIDsByPermission(ctx, "galgame:review")
	if err != nil {
		logger.Error("find galgame reviewers for notification", zap.Uint("galgame_id", galgame.ID), zap.Error(err))
		return
	}
	inputs := make([]notificationService.CreateInput, 0, len(recipientIDs))
	for _, recipientID := range recipientIDs {
		inputs = append(inputs, notificationService.CreateInput{
			RecipientID: recipientID,
			ActorID:     &actorID,
			Category:    notificationModel.CategoryReview,
			Type:        notificationModel.TypeGalgameSubmitted,
			EntityType:  "galgame",
			EntityID:    galgame.ID,
			Title:       "新的 Galgame 待审核",
			Content:     fmt.Sprintf("提交了 Galgame「%s」，等待审核", galgame.Title),
			TargetURL:   "/admin/galgames",
			Metadata:    map[string]any{"title": galgame.Title},
		})
	}
	if _, err := s.notifications.CreateMany(ctx, inputs); err != nil {
		logger.Error("create galgame submission notifications", zap.Uint("galgame_id", galgame.ID), zap.Error(err))
	}
}

func (s *CatalogService) notifyGalgameReviewResult(ctx context.Context, actorID uint, galgame *model.Galgame) {
	if s.notifications == nil || galgame.CreatedBy == nil {
		return
	}
	notificationType := notificationModel.TypeGalgameApproved
	title := "Galgame 审核通过"
	content := fmt.Sprintf("你提交的 Galgame「%s」已通过审核", galgame.Title)
	if galgame.Status == model.GalgameStatusRejected {
		notificationType = notificationModel.TypeGalgameRejected
		title = "Galgame 审核未通过"
		content = fmt.Sprintf("你提交的 Galgame「%s」未通过审核", galgame.Title)
	}
	if _, err := s.notifications.Create(ctx, notificationService.CreateInput{
		RecipientID: *galgame.CreatedBy,
		ActorID:     &actorID,
		Category:    notificationModel.CategoryReview,
		Type:        notificationType,
		EntityType:  "galgame",
		EntityID:    galgame.ID,
		Title:       title,
		Content:     content,
		TargetURL:   fmt.Sprintf("/galgames/%d", galgame.ID),
		Metadata:    map[string]any{"status": galgame.Status, "title": galgame.Title},
	}); err != nil {
		logger.Error("create galgame review notification", zap.Uint("galgame_id", galgame.ID), zap.Error(err))
	}
}

func parseReleaseDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, ErrInvalidReleaseDate
	}
	return &parsed, nil
}

func validAgeRating(value int16) bool {
	switch value {
	case model.AgeRatingUnknown,
		model.AgeRatingAll,
		model.AgeRatingR12,
		model.AgeRatingR15,
		model.AgeRatingR17,
		model.AgeRatingR18:
		return true
	default:
		return false
	}
}

func validStatus(value int16) bool {
	return value >= model.GalgameStatusPending && value <= model.GalgameStatusHidden
}

func validSort(value string) bool {
	switch value {
	case "latest", "oldest", "rating", "favorite", "popular":
		return true
	default:
		return false
	}
}

func normalizeSlug(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func hasConstraint(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == constraint
}

func uniqueUint(values []uint) []uint {
	seen := make(map[uint]struct{}, len(values))
	unique := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
