package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	contributionModel "backend/internal/contribution/model"
	contributionService "backend/internal/contribution/service"
	galgameRepository "backend/internal/galgame/repository"
	notificationModel "backend/internal/notification/model"
	notificationService "backend/internal/notification/service"
	novelRepository "backend/internal/novel/repository"
	rbacService "backend/internal/rbac/service"
	relationModel "backend/internal/relation/model"
	"backend/internal/resource/dto"
	"backend/internal/resource/model"
	"backend/internal/resource/repository"
	userModel "backend/internal/user/model"
	userService "backend/internal/user/service"
	"backend/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrTargetNotFound        = errors.New("resource target not found")
	ErrResourceNotFound      = errors.New("resource not found")
	ErrForbiddenResource     = errors.New("not allowed to manage this resource")
	ErrInvalidResourceInput  = errors.New("invalid resource input")
	ErrInvalidResourceType   = errors.New("invalid resource type")
	ErrInvalidTargetType     = errors.New("invalid resource target type")
	ErrInvalidResourceStatus = errors.New("invalid resource status")
	ErrEmptyResourceLinks    = errors.New("resource links must not be empty")
)

// Permission codes used for managing resources owned by other users.
const (
	PermissionResourceUpdate = "resource:update"
	PermissionResourceDelete = "resource:delete"
)

type ResourceService struct {
	resources     *repository.ResourceRepository
	galgames      *galgameRepository.GalgameRepository
	novels        *novelRepository.NovelRepository
	rbac          *rbacService.RBACService
	notifications *notificationService.NotificationService
	activities    userService.ActivityRecorder
	contributions *contributionService.ContributionService
}

func (s *ResourceService) SetActivityRecorder(recorder userService.ActivityRecorder) {
	s.activities = recorder
}

func (s *ResourceService) SetContributionService(contributions *contributionService.ContributionService) {
	s.contributions = contributions
}

func (s *ResourceService) SetNotificationDependencies(
	rbac *rbacService.RBACService,
	notifications *notificationService.NotificationService,
) {
	s.rbac = rbac
	s.notifications = notifications
}

func NewResourceService(
	resources *repository.ResourceRepository,
	galgames *galgameRepository.GalgameRepository,
	novels *novelRepository.NovelRepository,
	rbac *rbacService.RBACService,
) *ResourceService {
	return &ResourceService{
		resources: resources,
		galgames:  galgames,
		novels:    novels,
		rbac:      rbac,
	}
}

// ListPublishedByTarget returns one page of the target work's published
// resources with links.
func (s *ResourceService) ListPublishedByTarget(
	ctx context.Context,
	targetType string,
	targetID uint,
	page, limit int,
) ([]model.Resource, int64, int, int, error) {
	if err := s.ensurePublishedTarget(ctx, targetType, targetID); err != nil {
		return nil, 0, page, limit, err
	}
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	resources, total, err := s.resources.ListPublishedByTarget(ctx, targetType, targetID, page, limit)
	if err != nil {
		logger.Error("list published resources",
			zap.String("target_type", targetType), zap.Uint("target_id", targetID), zap.Error(err))
		return nil, 0, page, limit, err
	}
	return resources, total, page, limit, nil
}

// GetPublishedResource returns a published resource with links.
func (s *ResourceService) GetPublishedResource(ctx context.Context, id uint) (*model.Resource, error) {
	resource, err := s.resources.FindPublishedByID(ctx, id)
	if err != nil {
		logger.Error("find resource by id", zap.Uint("resource_id", id), zap.Error(err))
		return nil, err
	}
	if resource == nil {
		return nil, ErrResourceNotFound
	}
	return resource, nil
}

// CreateResource creates the resource, its links, and increments the target
// work's resource_count in one transaction. Any authenticated user may
// upload; uploader_id always comes from the login state.
func (s *ResourceService) CreateResource(
	ctx context.Context,
	uploaderID uint,
	req *dto.CreateResourceRequest,
) (*model.Resource, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrInvalidResourceInput
	}
	if !validResourceType(req.Type) {
		return nil, ErrInvalidResourceType
	}
	targetType := strings.TrimSpace(req.TargetType)
	if targetType == "" {
		targetType = relationModel.WorkTypeGalgame
	}
	if !relationModel.ValidWorkType(targetType) {
		return nil, ErrInvalidTargetType
	}
	status := model.ResourceStatusPending
	if req.Status != nil {
		if !validResourceStatus(*req.Status) {
			return nil, ErrInvalidResourceStatus
		}
		status = *req.Status
	}
	links := normalizeLinks(req.Links)
	if len(links) == 0 {
		return nil, ErrEmptyResourceLinks
	}
	if err := s.ensurePublishedTarget(ctx, targetType, req.TargetID); err != nil {
		return nil, err
	}

	resource := &model.Resource{
		TargetType:  targetType,
		TargetID:    req.TargetID,
		UploaderID:  &uploaderID,
		Title:       title,
		Type:        req.Type,
		Description: strings.TrimSpace(req.Description),
		Status:      status,
	}
	write := func(tx *repository.ResourceRepository, db *gorm.DB) error {
		if err := tx.Create(ctx, resource); err != nil {
			return err
		}
		if err := tx.CreateLinks(ctx, resource.ID, links); err != nil {
			return err
		}
		if err := tx.IncrementResourceCount(ctx, targetType, req.TargetID); err != nil {
			return err
		}
		if status == model.ResourceStatusPublished && s.contributions != nil {
			sourceType := contributionModel.ContributionSourceResource
			sourceID := resource.ID
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: targetType,
				TargetID:   req.TargetID,
				UserID:     uploaderID,
				Action:     contributionModel.ContributionActionResource,
				SourceType: &sourceType,
				SourceID:   &sourceID,
			}, db)
		}
		return nil
	}
	var err error
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewResourceRepository(db), db)
		})
	} else {
		err = s.resources.Transaction(ctx, func(tx *repository.ResourceRepository) error {
			return write(tx, nil)
		})
	}
	if err != nil {
		logger.Error("create resource",
			zap.String("target_type", targetType), zap.Uint("target_id", req.TargetID),
			zap.Uint("uploader_id", uploaderID), zap.Error(err))
		return nil, err
	}
	created, err := s.resources.FindByID(ctx, resource.ID)
	if err != nil {
		return nil, err
	}
	if created.Status == model.ResourceStatusPending {
		s.notifyResourceSubmitted(ctx, uploaderID, created)
	}
	if s.activities != nil {
		metadata := map[string]any{
			"title":       created.Title,
			"target_type": created.TargetType,
			"target_id":   created.TargetID,
		}
		if recordErr := s.activities.Record(ctx, uploaderID, userModel.ActivityResourceSubmitted, &created.ID, metadata); recordErr != nil {
			logger.Error("record resource activity", zap.Uint("resource_id", created.ID), zap.Error(recordErr))
		}
	}
	return created, nil
}

// UpdateResource fully replaces resource fields and links. The actor must be
// the uploader or hold the resource:update permission.
func (s *ResourceService) UpdateResource(
	ctx context.Context,
	actorID, id uint,
	req *dto.UpdateResourceRequest,
) (*model.Resource, error) {
	resource, err := s.resources.FindByID(ctx, id)
	if err != nil {
		logger.Error("find resource by id", zap.Uint("resource_id", id), zap.Error(err))
		return nil, err
	}
	if resource == nil {
		return nil, ErrResourceNotFound
	}
	if err := s.ensureCanManage(ctx, actorID, resource, PermissionResourceUpdate); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrInvalidResourceInput
	}
	if !validResourceType(req.Type) {
		return nil, ErrInvalidResourceType
	}
	if req.Status == nil || !validResourceStatus(*req.Status) {
		return nil, ErrInvalidResourceStatus
	}
	links := normalizeLinks(req.Links)
	if len(links) == 0 {
		return nil, ErrEmptyResourceLinks
	}

	oldStatus := resource.Status
	changed := resource.Title != title || resource.Type != req.Type ||
		resource.Description != strings.TrimSpace(req.Description) ||
		resource.Status != *req.Status || !resourceLinksEqual(resource.Links, links)
	resource.Title = title
	resource.Type = req.Type
	resource.Description = strings.TrimSpace(req.Description)
	resource.Status = *req.Status
	write := func(tx *repository.ResourceRepository, db *gorm.DB) error {
		if err := tx.Update(ctx, resource); err != nil {
			return err
		}
		if err := tx.ReplaceLinks(ctx, id, links); err != nil {
			return err
		}
		if changed && resource.Status == model.ResourceStatusPublished && s.contributions != nil {
			input := contributionService.RecordContributionInput{
				TargetType: resource.TargetType,
				TargetID:   resource.TargetID,
				UserID:     actorID,
				Action:     contributionModel.ContributionActionResource,
			}
			if oldStatus != model.ResourceStatusPublished {
				sourceType := contributionModel.ContributionSourceResource
				sourceID := resource.ID
				input.SourceType = &sourceType
				input.SourceID = &sourceID
			}
			return s.contributions.RecordContribution(ctx, input, db)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewResourceRepository(db), db)
		})
	} else {
		err = s.resources.Transaction(ctx, func(tx *repository.ResourceRepository) error {
			return write(tx, nil)
		})
	}
	if err != nil {
		logger.Error("update resource",
			zap.Uint("resource_id", id), zap.Uint("actor_id", actorID), zap.Error(err))
		return nil, err
	}
	return s.resources.FindByID(ctx, id)
}

// DeleteResource removes the resource (links cascade) and atomically
// decrements resource_count in one transaction. The actor must be the
// uploader or hold the resource:delete permission.
func (s *ResourceService) DeleteResource(ctx context.Context, actorID, id uint) error {
	resource, err := s.resources.FindByID(ctx, id)
	if err != nil {
		logger.Error("find resource by id", zap.Uint("resource_id", id), zap.Error(err))
		return err
	}
	if resource == nil {
		return ErrResourceNotFound
	}
	if err := s.ensureCanManage(ctx, actorID, resource, PermissionResourceDelete); err != nil {
		return err
	}

	err = s.resources.Transaction(ctx, func(tx *repository.ResourceRepository) error {
		removed, err := tx.Delete(ctx, id)
		if err != nil {
			return err
		}
		if !removed {
			return ErrResourceNotFound
		}
		return tx.DecrementResourceCount(ctx, resource.TargetType, resource.TargetID)
	})
	if err != nil {
		logger.Error("delete resource",
			zap.Uint("resource_id", id), zap.Uint("actor_id", actorID), zap.Error(err))
		return err
	}
	return nil
}

// ListAdminResources returns resources across all statuses with an optional
// status filter, newest first, for admins holding resource:review.
func (s *ResourceService) ListAdminResources(
	ctx context.Context,
	status *int16,
	page, limit int,
) ([]model.Resource, int64, int, int, error) {
	if status != nil && !validResourceStatus(*status) {
		return nil, 0, page, limit, ErrInvalidResourceStatus
	}
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	resources, total, err := s.resources.ListAdmin(ctx, repository.ResourceListOptions{
		Status: status,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		logger.Error("list admin resources", zap.Error(err))
		return nil, 0, page, limit, err
	}
	return resources, total, page, limit, nil
}

// ReviewResource transitions a resource to published, rejected, or hidden.
func (s *ResourceService) ReviewResource(
	ctx context.Context,
	id uint,
	req *dto.ReviewResourceRequest,
	actorIDs ...uint,
) (*model.Resource, error) {
	if req.Status != model.ResourceStatusPublished &&
		req.Status != model.ResourceStatusRejected &&
		req.Status != model.ResourceStatusHidden {
		return nil, ErrInvalidResourceStatus
	}
	resource, err := s.resources.FindByID(ctx, id)
	if err != nil {
		logger.Error("find resource by id", zap.Uint("resource_id", id), zap.Error(err))
		return nil, err
	}
	if resource == nil {
		return nil, ErrResourceNotFound
	}
	if resource.Status == req.Status {
		return resource, nil
	}

	resource.Status = req.Status
	if req.Status == model.ResourceStatusPublished && resource.UploaderID != nil && s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			if err := repository.NewResourceRepository(db).Update(ctx, resource); err != nil {
				return err
			}
			sourceType := contributionModel.ContributionSourceResource
			sourceID := resource.ID
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: resource.TargetType,
				TargetID:   resource.TargetID,
				UserID:     *resource.UploaderID,
				Action:     contributionModel.ContributionActionResource,
				SourceType: &sourceType,
				SourceID:   &sourceID,
			}, db)
		})
	} else {
		err = s.resources.Update(ctx, resource)
	}
	if err != nil {
		logger.Error("review resource", zap.Uint("resource_id", id), zap.Error(err))
		return nil, err
	}
	reviewed, err := s.resources.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	var actorID *uint
	if len(actorIDs) > 0 {
		actorID = &actorIDs[0]
	}
	s.notifyResourceReviewResult(ctx, actorID, reviewed)
	return reviewed, nil
}

func (s *ResourceService) notifyResourceSubmitted(ctx context.Context, actorID uint, resource *model.Resource) {
	if s.rbac == nil || s.notifications == nil {
		return
	}
	recipientIDs, err := s.rbac.FindUserIDsByPermission(ctx, "resource:review")
	if err != nil {
		logger.Error("find resource reviewers for notification", zap.Uint("resource_id", resource.ID), zap.Error(err))
		return
	}
	inputs := make([]notificationService.CreateInput, 0, len(recipientIDs))
	for _, recipientID := range recipientIDs {
		inputs = append(inputs, notificationService.CreateInput{
			RecipientID: recipientID,
			ActorID:     &actorID,
			Category:    notificationModel.CategoryReview,
			Type:        notificationModel.TypeResourceSubmitted,
			EntityType:  "resource",
			EntityID:    resource.ID,
			Title:       "新的资源待审核",
			Content:     fmt.Sprintf("提交了资源「%s」，等待审核", resource.Title),
			TargetURL:   "/admin/resources",
			Metadata: map[string]any{
				"target_type": resource.TargetType,
				"target_id":   resource.TargetID,
				"title":       resource.Title,
			},
		})
	}
	if _, err := s.notifications.CreateMany(ctx, inputs); err != nil {
		logger.Error("create resource submission notifications", zap.Uint("resource_id", resource.ID), zap.Error(err))
	}
}

func (s *ResourceService) notifyResourceReviewResult(
	ctx context.Context,
	actorID *uint,
	resource *model.Resource,
) {
	if s.notifications == nil || resource.UploaderID == nil {
		return
	}
	notificationType := notificationModel.TypeResourceApproved
	title := "资源审核通过"
	content := fmt.Sprintf("你提交的「%s」资源已通过审核", resource.Title)
	switch resource.Status {
	case model.ResourceStatusRejected:
		notificationType = notificationModel.TypeResourceRejected
		title = "资源审核未通过"
		content = fmt.Sprintf("你提交的「%s」资源未通过审核", resource.Title)
	case model.ResourceStatusHidden:
		notificationType = notificationModel.TypeResourceHidden
		title = "资源已被隐藏"
		content = fmt.Sprintf("你提交的「%s」资源已被管理员隐藏", resource.Title)
	}
	if _, err := s.notifications.Create(ctx, notificationService.CreateInput{
		RecipientID: *resource.UploaderID,
		ActorID:     actorID,
		Category:    notificationModel.CategoryReview,
		Type:        notificationType,
		EntityType:  "resource",
		EntityID:    resource.ID,
		Title:       title,
		Content:     content,
		TargetURL:   resourceTargetURL(resource),
		Metadata: map[string]any{
			"target_type": resource.TargetType,
			"target_id":   resource.TargetID,
			"status":      resource.Status,
			"title":       resource.Title,
		},
	}); err != nil {
		logger.Error("create resource review notification", zap.Uint("resource_id", resource.ID), zap.Error(err))
	}
}

func resourceTargetURL(resource *model.Resource) string {
	if resource.TargetType == relationModel.WorkTypeNovel {
		return fmt.Sprintf("/novels/%d", resource.TargetID)
	}
	return fmt.Sprintf("/galgames/%d", resource.TargetID)
}

// ensureCanManage allows the uploader to manage their own resource and falls
// back to the given RBAC permission for everyone else.
func (s *ResourceService) ensureCanManage(
	ctx context.Context,
	actorID uint,
	resource *model.Resource,
	permission string,
) error {
	if resource.UploaderID != nil && *resource.UploaderID == actorID {
		return nil
	}
	allowed, err := s.rbac.HasPermission(ctx, actorID, permission)
	if err != nil {
		logger.Error("check resource permission",
			zap.String("permission", permission), zap.Uint("actor_id", actorID), zap.Error(err))
		return err
	}
	if !allowed {
		return ErrForbiddenResource
	}
	return nil
}

// ensurePublishedTarget verifies the bound work exists and is published.
func (s *ResourceService) ensurePublishedTarget(ctx context.Context, targetType string, targetID uint) error {
	switch targetType {
	case relationModel.WorkTypeGalgame:
		galgame, err := s.galgames.FindPublishedByID(ctx, targetID)
		if err != nil {
			logger.Error("find galgame by id", zap.Uint("galgame_id", targetID), zap.Error(err))
			return err
		}
		if galgame == nil {
			return ErrTargetNotFound
		}
		return nil
	case relationModel.WorkTypeNovel:
		novel, err := s.novels.FindPublishedByID(ctx, targetID)
		if err != nil {
			logger.Error("find novel by id", zap.Uint("novel_id", targetID), zap.Error(err))
			return err
		}
		if novel == nil {
			return ErrTargetNotFound
		}
		return nil
	default:
		return ErrInvalidTargetType
	}
}

func validResourceType(value int16) bool {
	return value >= model.ResourceTypeOther && value <= model.ResourceTypeArchive
}

func validResourceStatus(value int16) bool {
	return value >= model.ResourceStatusPending && value <= model.ResourceStatusHidden
}

func normalizeLinks(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	links := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		links = append(links, value)
	}
	return links
}

func resourceLinksEqual(current []model.ResourceLink, expected []string) bool {
	if len(current) != len(expected) {
		return false
	}
	counts := make(map[string]int, len(current))
	for _, link := range current {
		counts[link.URL]++
	}
	for _, link := range expected {
		counts[link]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}
