package service

import (
	"context"
	"errors"
	"strings"

	galgameRepository "backend/internal/galgame/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/resource/dto"
	"backend/internal/resource/model"
	"backend/internal/resource/repository"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

var (
	ErrGalgameNotFound       = errors.New("galgame not found")
	ErrResourceNotFound      = errors.New("resource not found")
	ErrForbiddenResource     = errors.New("not allowed to manage this resource")
	ErrInvalidResourceInput  = errors.New("invalid resource input")
	ErrInvalidResourceType   = errors.New("invalid resource type")
	ErrInvalidResourceStatus = errors.New("invalid resource status")
	ErrEmptyResourceLinks    = errors.New("resource links must not be empty")
)

// Permission codes used for managing resources owned by other users.
const (
	PermissionResourceUpdate = "resource:update"
	PermissionResourceDelete = "resource:delete"
)

type ResourceService struct {
	resources *repository.ResourceRepository
	galgames  *galgameRepository.GalgameRepository
	rbac      *rbacService.RBACService
}

func NewResourceService(
	resources *repository.ResourceRepository,
	galgames *galgameRepository.GalgameRepository,
	rbac *rbacService.RBACService,
) *ResourceService {
	return &ResourceService{resources: resources, galgames: galgames, rbac: rbac}
}

// ListPublishedByGalgame returns the galgame's published resources with links.
func (s *ResourceService) ListPublishedByGalgame(ctx context.Context, galgameID uint) ([]model.Resource, error) {
	if err := s.ensurePublishedGalgame(ctx, galgameID); err != nil {
		return nil, err
	}
	resources, err := s.resources.ListPublishedByGalgame(ctx, galgameID)
	if err != nil {
		logger.Error("list published resources", zap.Uint("galgame_id", galgameID), zap.Error(err))
		return nil, err
	}
	return resources, nil
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

// CreateResource creates the resource, its links, and increments the
// galgame's resource_count in one transaction. Any authenticated user may
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
	if err := s.ensurePublishedGalgame(ctx, req.GalgameID); err != nil {
		return nil, err
	}

	resource := &model.Resource{
		GalgameID:   req.GalgameID,
		UploaderID:  &uploaderID,
		Title:       title,
		Type:        req.Type,
		Description: strings.TrimSpace(req.Description),
		Status:      status,
	}
	err := s.resources.Transaction(ctx, func(tx *repository.ResourceRepository) error {
		if err := tx.Create(ctx, resource); err != nil {
			return err
		}
		if err := tx.CreateLinks(ctx, resource.ID, links); err != nil {
			return err
		}
		return tx.IncrementResourceCount(ctx, req.GalgameID)
	})
	if err != nil {
		logger.Error("create resource",
			zap.Uint("galgame_id", req.GalgameID), zap.Uint("uploader_id", uploaderID), zap.Error(err))
		return nil, err
	}
	return s.resources.FindByID(ctx, resource.ID)
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

	resource.Title = title
	resource.Type = req.Type
	resource.Description = strings.TrimSpace(req.Description)
	resource.Status = *req.Status
	err = s.resources.Transaction(ctx, func(tx *repository.ResourceRepository) error {
		if err := tx.Update(ctx, resource); err != nil {
			return err
		}
		return tx.ReplaceLinks(ctx, id, links)
	})
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
		return tx.DecrementResourceCount(ctx, resource.GalgameID)
	})
	if err != nil {
		logger.Error("delete resource",
			zap.Uint("resource_id", id), zap.Uint("actor_id", actorID), zap.Error(err))
		return err
	}
	return nil
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

func (s *ResourceService) ensurePublishedGalgame(ctx context.Context, id uint) error {
	galgame, err := s.galgames.FindPublishedByID(ctx, id)
	if err != nil {
		logger.Error("find galgame by id", zap.Uint("galgame_id", id), zap.Error(err))
		return err
	}
	if galgame == nil {
		return ErrGalgameNotFound
	}
	return nil
}

func validResourceType(value int16) bool {
	return value >= model.ResourceTypeOther && value <= model.ResourceTypeGuide
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
