package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	contributionModel "backend/internal/contribution/model"
	contributionService "backend/internal/contribution/service"
	galgameModel "backend/internal/galgame/model"
	galgameRepository "backend/internal/galgame/repository"
	notificationModel "backend/internal/notification/model"
	notificationService "backend/internal/notification/service"
	"backend/internal/novel/dto"
	"backend/internal/novel/model"
	"backend/internal/novel/repository"
	rbacService "backend/internal/rbac/service"
	relationModel "backend/internal/relation/model"
	relationRepository "backend/internal/relation/repository"
	"backend/pkg/logger"

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNovelNotFound       = errors.New("novel not found")
	ErrNovelSlugExists     = errors.New("novel slug already exists")
	ErrUnknownTagIDs       = errors.New("tag ids contain unknown tag")
	ErrInvalidNovelInput   = errors.New("invalid novel input")
	ErrInvalidNovelURL     = errors.New("invalid novel url")
	ErrInvalidReleaseDate  = errors.New("invalid release date")
	ErrInvalidAgeRating    = errors.New("invalid age rating")
	ErrInvalidStatus       = errors.New("invalid novel status")
	ErrInvalidSort         = errors.New("invalid novel sort")
	ErrInvalidReleaseState = errors.New("invalid novel release status")
)

// detailVolumeLimit caps how many volumes the detail response embeds; larger
// series page through the dedicated volume endpoint.
const detailVolumeLimit = 50

type NovelService struct {
	novels        *repository.NovelRepository
	volumes       *repository.VolumeRepository
	tags          *galgameRepository.TagRepository
	galgames      *galgameRepository.GalgameRepository
	relations     *relationRepository.RelationRepository
	contributions *contributionService.ContributionService
	rbac          *rbacService.RBACService
	notifications *notificationService.NotificationService
}

func NewNovelService(
	novels *repository.NovelRepository,
	volumes *repository.VolumeRepository,
	tags *galgameRepository.TagRepository,
	galgames *galgameRepository.GalgameRepository,
	relations *relationRepository.RelationRepository,
) *NovelService {
	return &NovelService{
		novels:    novels,
		volumes:   volumes,
		tags:      tags,
		galgames:  galgames,
		relations: relations,
	}
}

func (s *NovelService) SetContributionService(contributions *contributionService.ContributionService) {
	s.contributions = contributions
}

func (s *NovelService) SetNotificationDependencies(
	rbac *rbacService.RBACService,
	notifications *notificationService.NotificationService,
) {
	s.rbac = rbac
	s.notifications = notifications
}

func (s *NovelService) CreateNovel(
	ctx context.Context,
	userID uint,
	req *dto.CreateNovelRequest,
) (*model.Novel, error) {
	title := strings.TrimSpace(req.Title)
	slug := normalizeSlug(req.Slug)
	if title == "" || slug == "" {
		return nil, ErrInvalidNovelInput
	}
	releaseStatus := strings.TrimSpace(req.ReleaseStatus)
	if releaseStatus == "" {
		releaseStatus = model.ReleaseStatusUnknown
	}
	firstReleaseDate, err := parseReleaseDate(req.FirstReleaseDate)
	if err != nil {
		return nil, err
	}
	if !model.ValidReleaseStatus(releaseStatus) {
		return nil, ErrInvalidReleaseState
	}
	if !validAgeRating(req.AgeRating) {
		return nil, ErrInvalidAgeRating
	}
	if req.Status != model.NovelStatusPending && req.Status != model.NovelStatusPublished {
		return nil, ErrInvalidStatus
	}
	if !validHTTPURL(req.CoverURL) || !validHTTPURL(req.OfficialWebsite) {
		return nil, ErrInvalidNovelURL
	}
	tagIDs := uniqueUint(req.TagIDs)
	if err := s.validateTags(ctx, tagIDs); err != nil {
		return nil, err
	}
	if err := s.ensureNovelSlugUnique(ctx, 0, slug); err != nil {
		return nil, err
	}

	novel := &model.Novel{
		Title:            title,
		OriginalTitle:    strings.TrimSpace(req.OriginalTitle),
		Slug:             slug,
		Summary:          strings.TrimSpace(req.Summary),
		CoverURL:         strings.TrimSpace(req.CoverURL),
		Author:           strings.TrimSpace(req.Author),
		Illustrator:      strings.TrimSpace(req.Illustrator),
		Publisher:        strings.TrimSpace(req.Publisher),
		Label:            strings.TrimSpace(req.Label),
		Language:         strings.TrimSpace(req.Language),
		Region:           strings.TrimSpace(req.Region),
		ReleaseStatus:    releaseStatus,
		FirstReleaseDate: firstReleaseDate,
		AgeRating:        req.AgeRating,
		IsCoverSensitive: req.IsCoverSensitive,
		OfficialWebsite:  strings.TrimSpace(req.OfficialWebsite),
		Status:           req.Status,
		CreatedBy:        &userID,
	}
	write := func(tx *repository.NovelRepository, db *gorm.DB) error {
		if err := tx.Create(ctx, novel); err != nil {
			return err
		}
		if err := tx.ReplaceTags(ctx, novel.ID, tagIDs); err != nil {
			return err
		}
		if novel.Status == model.NovelStatusPublished && s.contributions != nil {
			sourceType := contributionModel.ContributionSourceNovelCreate
			sourceID := novel.ID
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeNovel,
				TargetID:   novel.ID,
				UserID:     userID,
				Action:     contributionModel.ContributionActionCreate,
				SourceType: &sourceType,
				SourceID:   &sourceID,
			}, db)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewNovelRepository(db), db)
		})
	} else {
		err = s.novels.Transaction(ctx, func(tx *repository.NovelRepository) error {
			return write(tx, nil)
		})
	}
	if err != nil {
		if hasConstraint(err, "novels_slug_unique") {
			return nil, ErrNovelSlugExists
		}
		logger.Error("create novel", zap.String("slug", slug), zap.Error(err))
		return nil, err
	}
	created, err := s.getNovel(ctx, novel.ID, false)
	if err != nil {
		return nil, err
	}
	if created.Status == model.NovelStatusPending {
		s.notifyNovelSubmitted(ctx, userID, created)
	}
	return created, nil
}

func (s *NovelService) UpdateNovel(
	ctx context.Context,
	actorID, id uint,
	req *dto.UpdateNovelRequest,
) (*model.Novel, error) {
	title := strings.TrimSpace(req.Title)
	slug := normalizeSlug(req.Slug)
	if title == "" || slug == "" {
		return nil, ErrInvalidNovelInput
	}
	novel, err := s.novels.FindByID(ctx, id)
	if err != nil {
		logger.Error("find novel by id", zap.Uint("novel_id", id), zap.Error(err))
		return nil, err
	}
	if novel == nil {
		return nil, ErrNovelNotFound
	}
	if !model.ValidReleaseStatus(req.ReleaseStatus) {
		return nil, ErrInvalidReleaseState
	}
	if req.AgeRating == nil || !validAgeRating(*req.AgeRating) {
		return nil, ErrInvalidAgeRating
	}
	if req.IsCoverSensitive == nil {
		return nil, ErrInvalidNovelInput
	}
	if req.Status == nil || !validNovelStatus(*req.Status) {
		return nil, ErrInvalidStatus
	}
	if !validHTTPURL(req.CoverURL) || !validHTTPURL(req.OfficialWebsite) {
		return nil, ErrInvalidNovelURL
	}
	firstReleaseDate, err := parseReleaseDate(req.FirstReleaseDate)
	if err != nil {
		return nil, err
	}
	tagIDs := uniqueUint(req.TagIDs)
	if err := s.validateTags(ctx, tagIDs); err != nil {
		return nil, err
	}
	if err := s.ensureNovelSlugUnique(ctx, id, slug); err != nil {
		return nil, err
	}

	oldStatus := novel.Status
	changed, coverOnly := novelUpdateChanges(novel, req, title, slug, firstReleaseDate, tagIDs)

	novel.Title = title
	novel.OriginalTitle = strings.TrimSpace(req.OriginalTitle)
	novel.Slug = slug
	novel.Summary = strings.TrimSpace(req.Summary)
	novel.CoverURL = strings.TrimSpace(req.CoverURL)
	novel.Author = strings.TrimSpace(req.Author)
	novel.Illustrator = strings.TrimSpace(req.Illustrator)
	novel.Publisher = strings.TrimSpace(req.Publisher)
	novel.Label = strings.TrimSpace(req.Label)
	novel.Language = strings.TrimSpace(req.Language)
	novel.Region = strings.TrimSpace(req.Region)
	novel.ReleaseStatus = req.ReleaseStatus
	novel.FirstReleaseDate = firstReleaseDate
	novel.AgeRating = *req.AgeRating
	novel.IsCoverSensitive = *req.IsCoverSensitive
	novel.OfficialWebsite = strings.TrimSpace(req.OfficialWebsite)
	novel.Status = *req.Status

	write := func(tx *repository.NovelRepository, db *gorm.DB) error {
		if err := tx.Update(ctx, novel); err != nil {
			return err
		}
		if err := tx.ReplaceTags(ctx, id, tagIDs); err != nil {
			return err
		}
		if changed && novel.Status == model.NovelStatusPublished && s.contributions != nil {
			if oldStatus != model.NovelStatusPublished {
				contributorID := uint(0)
				if novel.CreatedBy != nil {
					contributorID = *novel.CreatedBy
				} else {
					contributorID = actorID
				}
				if contributorID != 0 {
					sourceType := contributionModel.ContributionSourceNovelCreate
					sourceID := id
					if err := s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
						TargetType: relationModel.WorkTypeNovel,
						TargetID:   id,
						UserID:     contributorID,
						Action:     contributionModel.ContributionActionCreate,
						SourceType: &sourceType,
						SourceID:   &sourceID,
					}, db); err != nil {
						return err
					}
				}
				return s.recordPublishedVolumeContributions(ctx, db, id)
			}
			action := contributionModel.ContributionActionEdit
			if coverOnly {
				action = contributionModel.ContributionActionCover
			}
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeNovel,
				TargetID:   id,
				UserID:     actorID,
				Action:     action,
			}, db)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewNovelRepository(db), db)
		})
	} else {
		err = s.novels.Transaction(ctx, func(tx *repository.NovelRepository) error {
			return write(tx, nil)
		})
	}
	if err != nil {
		if hasConstraint(err, "novels_slug_unique") {
			return nil, ErrNovelSlugExists
		}
		logger.Error("update novel", zap.Uint("novel_id", id), zap.Error(err))
		return nil, err
	}
	return s.getNovel(ctx, id, false)
}

func (s *NovelService) ReviewNovel(
	ctx context.Context,
	actorID, id uint,
	req *dto.ReviewNovelRequest,
) (*model.Novel, error) {
	if req.Status != model.NovelStatusPublished && req.Status != model.NovelStatusRejected {
		return nil, ErrInvalidStatus
	}
	novel, err := s.novels.FindByID(ctx, id)
	if err != nil {
		logger.Error("find novel by id", zap.Uint("novel_id", id), zap.Error(err))
		return nil, err
	}
	if novel == nil {
		return nil, ErrNovelNotFound
	}
	if novel.Status == req.Status {
		return novel, nil
	}

	reason := strings.TrimSpace(req.Reason)
	now := time.Now()
	if req.Status == model.NovelStatusPublished && s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			if err := repository.NewNovelRepository(db).UpdateReview(ctx, id, req.Status, &actorID, &now, ""); err != nil {
				return err
			}
			if novel.CreatedBy != nil {
				sourceType := contributionModel.ContributionSourceNovelCreate
				sourceID := id
				if err := s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
					TargetType: relationModel.WorkTypeNovel,
					TargetID:   id,
					UserID:     *novel.CreatedBy,
					Action:     contributionModel.ContributionActionCreate,
					SourceType: &sourceType,
					SourceID:   &sourceID,
				}, db); err != nil {
					return err
				}
			}
			return s.recordPublishedVolumeContributions(ctx, db, id)
		})
	} else {
		err = s.novels.UpdateReview(ctx, id, req.Status, &actorID, &now, reason)
	}
	if err != nil {
		logger.Error("review novel", zap.Uint("novel_id", id), zap.Uint("actor_id", actorID), zap.Error(err))
		return nil, err
	}
	reviewed, err := s.getNovel(ctx, id, false)
	if err != nil {
		return nil, err
	}
	s.notifyNovelReviewResult(ctx, actorID, reviewed, reason)
	return reviewed, nil
}

// DeleteNovel soft-deletes the novel and its volumes, then physically removes
// tag bindings, work relations, and external mappings to avoid orphan rows.
// Resources keep their own lifecycle.
func (s *NovelService) DeleteNovel(ctx context.Context, id uint) error {
	novel, err := s.novels.FindByID(ctx, id)
	if err != nil {
		logger.Error("find novel by id", zap.Uint("novel_id", id), zap.Error(err))
		return err
	}
	if novel == nil {
		return ErrNovelNotFound
	}
	deleteFn := func(tx *repository.NovelRepository, db *gorm.DB) error {
		if err := tx.Delete(ctx, id); err != nil {
			return err
		}
		if err := tx.DeleteTags(ctx, id); err != nil {
			return err
		}
		volumes, err := repository.NewVolumeRepository(db).ListByNovel(ctx, id, false)
		if err != nil {
			return err
		}
		volumeRepo := repository.NewVolumeRepository(db)
		for _, volume := range volumes {
			if err := volumeRepo.Delete(ctx, volume.ID); err != nil {
				return err
			}
		}
		if err := relationRepository.NewRelationRepository(db).DeleteByWork(ctx, relationModel.WorkTypeNovel, id); err != nil {
			return err
		}
		return tx.DeleteExternalMappings(ctx, id)
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return deleteFn(repository.NewNovelRepository(db), db)
		})
	} else {
		err = s.novels.Transaction(ctx, func(tx *repository.NovelRepository) error {
			return deleteFn(tx, nil)
		})
	}
	if err != nil {
		logger.Error("delete novel", zap.Uint("novel_id", id), zap.Error(err))
		return err
	}
	return nil
}

func (s *NovelService) GetPublishedNovel(ctx context.Context, id uint) (*model.Novel, error) {
	return s.getNovel(ctx, id, true)
}

// GetNovel returns a novel of any status for the novel:review admin detail.
func (s *NovelService) GetNovel(ctx context.Context, id uint) (*model.Novel, error) {
	return s.getNovel(ctx, id, false)
}

func (s *NovelService) ListPublishedNovels(
	ctx context.Context,
	query *dto.NovelQuery,
) ([]model.Novel, int64, int, int, error) {
	page, limit := normalizePage(query.Page, query.Limit)
	sort := normalizeSort(query.Sort)
	if !validSort(sort) {
		return nil, 0, page, limit, ErrInvalidSort
	}
	if query.ReleaseStatus != "" && !model.ValidReleaseStatus(query.ReleaseStatus) {
		return nil, 0, page, limit, ErrInvalidReleaseState
	}
	novels, total, err := s.novels.ListPublished(ctx, repository.NovelListOptions{
		Keyword:       strings.TrimSpace(query.Keyword),
		TagIDs:        uniqueUint(query.TagIDs),
		Author:        strings.TrimSpace(query.Author),
		Publisher:     strings.TrimSpace(query.Publisher),
		Label:         strings.TrimSpace(query.Label),
		Language:      strings.TrimSpace(query.Language),
		ReleaseStatus: query.ReleaseStatus,
		Sort:          sort,
		Page:          page,
		Limit:         limit,
	})
	if err != nil {
		logger.Error("list published novels", zap.Error(err))
		return nil, 0, page, limit, err
	}
	return novels, total, page, limit, nil
}

func (s *NovelService) ListAdminNovels(
	ctx context.Context,
	query *dto.AdminNovelQuery,
) ([]model.Novel, int64, int, int, error) {
	page, limit := normalizePage(query.Page, query.Limit)
	sort := normalizeSort(query.Sort)
	if !validSort(sort) {
		return nil, 0, page, limit, ErrInvalidSort
	}
	if query.Status != nil && !validNovelStatus(*query.Status) {
		return nil, 0, page, limit, ErrInvalidStatus
	}
	if query.ReleaseStatus != "" && !model.ValidReleaseStatus(query.ReleaseStatus) {
		return nil, 0, page, limit, ErrInvalidReleaseState
	}
	novels, total, err := s.novels.ListAdmin(ctx, repository.NovelListOptions{
		Keyword:       strings.TrimSpace(query.Keyword),
		ReleaseStatus: query.ReleaseStatus,
		Status:        query.Status,
		Sort:          sort,
		Page:          page,
		Limit:         limit,
	})
	if err != nil {
		logger.Error("list admin novels", zap.Error(err))
		return nil, 0, page, limit, err
	}
	return novels, total, page, limit, nil
}

func (s *NovelService) getNovel(
	ctx context.Context,
	id uint,
	publishedOnly bool,
) (*model.Novel, error) {
	var (
		novel *model.Novel
		err   error
	)
	if publishedOnly {
		novel, err = s.novels.FindPublishedByID(ctx, id)
	} else {
		novel, err = s.novels.FindByID(ctx, id)
	}
	if err != nil {
		logger.Error("find novel by id", zap.Uint("novel_id", id), zap.Error(err))
		return nil, err
	}
	if novel == nil {
		return nil, ErrNovelNotFound
	}
	novel.VolumeCount, err = s.novels.CountPublishedVolumes(ctx, id)
	if err != nil {
		return nil, err
	}
	publishedOnlyVolumes := publishedOnly
	volumes, err := s.volumes.ListByNovel(ctx, id, publishedOnlyVolumes)
	if err != nil {
		logger.Error("list novel volumes", zap.Uint("novel_id", id), zap.Error(err))
		return nil, err
	}
	if len(volumes) > detailVolumeLimit {
		volumes = volumes[:detailVolumeLimit]
	}
	novel.Volumes = volumes
	if s.contributions != nil {
		contributors, total, err := s.contributions.ListContributorRows(ctx, relationModel.WorkTypeNovel, id, 1, 10)
		if err != nil {
			return nil, err
		}
		novel.Contributors = contributors
		novel.ContributorCount = total
	}
	novel.RelatedGalgames, err = s.relations.ListRelatedGalgamesForNovel(ctx, id)
	if err != nil {
		logger.Error("list related galgames for novel", zap.Uint("novel_id", id), zap.Error(err))
		return nil, err
	}
	return novel, nil
}

// recordPublishedVolumeContributions credits the creator of every published
// volume once, mirroring the galgame gallery approval backfill.
func (s *NovelService) recordPublishedVolumeContributions(ctx context.Context, db *gorm.DB, novelID uint) error {
	volumes, err := repository.NewVolumeRepository(db).ListByNovel(ctx, novelID, true)
	if err != nil {
		return err
	}
	for _, volume := range volumes {
		if volume.CreatedBy == nil {
			continue
		}
		sourceType := contributionModel.ContributionSourceNovelVolume
		sourceID := volume.ID
		if err := s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
			TargetType: relationModel.WorkTypeNovel,
			TargetID:   novelID,
			UserID:     *volume.CreatedBy,
			Action:     contributionModel.ContributionActionAddVolume,
			SourceType: &sourceType,
			SourceID:   &sourceID,
		}, db); err != nil {
			return err
		}
	}
	return nil
}

func (s *NovelService) validateTags(ctx context.Context, tagIDs []uint) error {
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

func (s *NovelService) ensureNovelSlugUnique(ctx context.Context, id uint, slug string) error {
	existing, err := s.novels.FindBySlug(ctx, slug)
	if err != nil {
		logger.Error("find novel by slug", zap.String("slug", slug), zap.Error(err))
		return err
	}
	if existing != nil && existing.ID != id {
		return ErrNovelSlugExists
	}
	return nil
}

func (s *NovelService) notifyNovelSubmitted(ctx context.Context, actorID uint, novel *model.Novel) {
	if s.rbac == nil || s.notifications == nil {
		return
	}
	recipientIDs, err := s.rbac.FindUserIDsByPermission(ctx, "novel:review")
	if err != nil {
		logger.Error("find novel reviewers for notification", zap.Uint("novel_id", novel.ID), zap.Error(err))
		return
	}
	inputs := make([]notificationService.CreateInput, 0, len(recipientIDs))
	for _, recipientID := range recipientIDs {
		inputs = append(inputs, notificationService.CreateInput{
			RecipientID: recipientID,
			ActorID:     &actorID,
			Category:    notificationModel.CategoryReview,
			Type:        notificationModel.TypeNovelSubmitted,
			EntityType:  "novel",
			EntityID:    novel.ID,
			Title:       "新的小说待审核",
			Content:     fmt.Sprintf("提交了小说「%s」，等待审核", novel.Title),
			TargetURL:   "/admin/novels",
			Metadata:    map[string]any{"title": novel.Title},
		})
	}
	if _, err := s.notifications.CreateMany(ctx, inputs); err != nil {
		logger.Error("create novel submission notifications", zap.Uint("novel_id", novel.ID), zap.Error(err))
	}
}

func (s *NovelService) notifyNovelReviewResult(ctx context.Context, actorID uint, novel *model.Novel, reason string) {
	if s.notifications == nil || novel.CreatedBy == nil {
		return
	}
	notificationType := notificationModel.TypeNovelApproved
	title := "小说审核通过"
	content := fmt.Sprintf("你提交的小说「%s」已通过审核", novel.Title)
	if novel.Status == model.NovelStatusRejected {
		notificationType = notificationModel.TypeNovelRejected
		title = "小说审核未通过"
		content = fmt.Sprintf("你提交的小说「%s」未通过审核", novel.Title)
		if reason != "" {
			content = fmt.Sprintf("你提交的小说「%s」未通过审核：%s", novel.Title, reason)
		}
	}
	if _, err := s.notifications.Create(ctx, notificationService.CreateInput{
		RecipientID: *novel.CreatedBy,
		ActorID:     &actorID,
		Category:    notificationModel.CategoryReview,
		Type:        notificationType,
		EntityType:  "novel",
		EntityID:    novel.ID,
		Title:       title,
		Content:     content,
		TargetURL:   fmt.Sprintf("/novels/%d", novel.ID),
		Metadata: map[string]any{
			"status": novel.Status,
			"title":  novel.Title,
			"reason": reason,
		},
	}); err != nil {
		logger.Error("create novel review notification", zap.Uint("novel_id", novel.ID), zap.Error(err))
	}
}

func novelUpdateChanges(
	novel *model.Novel,
	req *dto.UpdateNovelRequest,
	title, slug string,
	firstReleaseDate *time.Time,
	tagIDs []uint,
) (bool, bool) {
	coverChanged := novel.CoverURL != strings.TrimSpace(req.CoverURL)
	otherChanged := novel.Title != title ||
		novel.OriginalTitle != strings.TrimSpace(req.OriginalTitle) ||
		novel.Slug != slug ||
		novel.Summary != strings.TrimSpace(req.Summary) ||
		novel.Author != strings.TrimSpace(req.Author) ||
		novel.Illustrator != strings.TrimSpace(req.Illustrator) ||
		novel.Publisher != strings.TrimSpace(req.Publisher) ||
		novel.Label != strings.TrimSpace(req.Label) ||
		novel.Language != strings.TrimSpace(req.Language) ||
		novel.Region != strings.TrimSpace(req.Region) ||
		novel.ReleaseStatus != req.ReleaseStatus ||
		!equalTimePointers(novel.FirstReleaseDate, firstReleaseDate) ||
		novel.AgeRating != *req.AgeRating ||
		novel.IsCoverSensitive != *req.IsCoverSensitive ||
		novel.OfficialWebsite != strings.TrimSpace(req.OfficialWebsite) ||
		novel.Status != *req.Status ||
		!equalTagIDs(novel.Tags, tagIDs)
	return coverChanged || otherChanged, coverChanged && !otherChanged
}

func equalTagIDs(current []galgameModel.Tag, expected []uint) bool {
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

func equalTimePointers(left, right *time.Time) bool {
	return (left == nil && right == nil) ||
		(left != nil && right != nil && left.Equal(*right))
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
	case galgameModel.AgeRatingUnknown,
		galgameModel.AgeRatingAll,
		galgameModel.AgeRatingR12,
		galgameModel.AgeRatingR15,
		galgameModel.AgeRatingR17,
		galgameModel.AgeRatingR18:
		return true
	default:
		return false
	}
}

func validNovelStatus(value int16) bool {
	return value >= model.NovelStatusPending && value <= model.NovelStatusHidden
}

func validSort(value string) bool {
	switch value {
	case "latest", "oldest", "updated", "release", "release_asc":
		return true
	default:
		return false
	}
}

func normalizeSort(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "latest"
	}
	return value
}

func normalizePage(page, limit int) (int, int) {
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	return page, limit
}

func normalizeSlug(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// validHTTPURL accepts empty values and absolute http(s) URLs; remote covers
// and official websites may be entered directly for data imports.
func validHTTPURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
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
