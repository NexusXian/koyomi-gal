package service

import (
	"context"
	"errors"
	"strings"

	"backend/internal/announcement/dto"
	"backend/internal/announcement/model"
	"backend/internal/announcement/repository"
	rbacService "backend/internal/rbac/service"
)

var (
	ErrAnnouncementNotFound  = errors.New("announcement not found")
	ErrInvalidAnnouncement   = errors.New("invalid announcement")
	ErrInvalidSchedule       = errors.New("invalid announcement schedule")
	ErrInvalidPlatform       = errors.New("invalid announcement platform")
	ErrAnnouncementForbidden = errors.New("announcement permission denied")
)

const PermissionAnnouncementPublish = "announcement:publish"

type AnnouncementService struct {
	announcements *repository.AnnouncementRepository
	rbac          *rbacService.RBACService
}

func NewAnnouncementService(announcements *repository.AnnouncementRepository, rbac *rbacService.RBACService) *AnnouncementService {
	return &AnnouncementService{announcements: announcements, rbac: rbac}
}

func (s *AnnouncementService) ListActive(ctx context.Context, platform string) ([]model.Announcement, error) {
	if !validQueryPlatform(platform) {
		return nil, ErrInvalidPlatform
	}
	return s.announcements.ListActive(ctx, platform)
}

func (s *AnnouncementService) ListAdmin(ctx context.Context, page, limit int) ([]model.Announcement, int64, int, int, error) {
	page, limit = pagination(page, limit)
	values, total, err := s.announcements.ListAdmin(ctx, page, limit)
	return values, total, page, limit, err
}

func (s *AnnouncementService) Get(ctx context.Context, id uint) (*model.Announcement, error) {
	value, err := s.announcements.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, ErrAnnouncementNotFound
	}
	return value, nil
}

func (s *AnnouncementService) Create(ctx context.Context, actorID uint, req *dto.AnnouncementRequest) (*model.Announcement, error) {
	value := announcementFromRequest(req, nil)
	value.CreatedBy = &actorID
	if err := validateAnnouncement(value); err != nil {
		return nil, err
	}
	if value.Published {
		if err := s.requirePublish(ctx, actorID); err != nil {
			return nil, err
		}
	}
	if err := s.announcements.Create(ctx, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *AnnouncementService) Update(ctx context.Context, actorID, id uint, req *dto.AnnouncementRequest) (*model.Announcement, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	desired := announcementFromRequest(req, current)
	desired.ID, desired.CreatedBy, desired.CreatedAt = current.ID, current.CreatedBy, current.CreatedAt
	if err := validateAnnouncement(desired); err != nil {
		return nil, err
	}
	if current.Published != desired.Published {
		if err := s.requirePublish(ctx, actorID); err != nil {
			return nil, err
		}
	}
	if err := s.announcements.Update(ctx, desired); err != nil {
		return nil, err
	}
	return desired, nil
}

func (s *AnnouncementService) Delete(ctx context.Context, id uint) error {
	deleted, err := s.announcements.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrAnnouncementNotFound
	}
	return nil
}

func (s *AnnouncementService) Publish(ctx context.Context, id uint) (*model.Announcement, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	if err := s.announcements.SetPublished(ctx, id, true); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *AnnouncementService) Withdraw(ctx context.Context, id uint) (*model.Announcement, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	if err := s.announcements.SetPublished(ctx, id, false); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *AnnouncementService) requirePublish(ctx context.Context, actorID uint) error {
	allowed, err := s.rbac.HasPermission(ctx, actorID, PermissionAnnouncementPublish)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrAnnouncementForbidden
	}
	return nil
}

func announcementFromRequest(req *dto.AnnouncementRequest, current *model.Announcement) *model.Announcement {
	dismissible := true
	published := false
	if current != nil {
		dismissible = current.Dismissible
		published = current.Published
	}
	if req.Dismissible != nil {
		dismissible = *req.Dismissible
	}
	if req.Published != nil {
		published = *req.Published
	}
	return &model.Announcement{
		Title: strings.TrimSpace(req.Title), Content: strings.TrimSpace(req.Content),
		Type: req.Type, DisplayMode: req.DisplayMode, Target: req.Target,
		Priority: req.Priority, StartsAt: req.StartsAt, EndsAt: req.EndsAt,
		Dismissible: dismissible, Published: published,
	}
}

func validateAnnouncement(value *model.Announcement) error {
	if value.Title == "" || value.Content == "" || !validType(value.Type) ||
		!validDisplayMode(value.DisplayMode) || !validTarget(value.Target) {
		return ErrInvalidAnnouncement
	}
	if value.StartsAt != nil && value.EndsAt != nil && value.EndsAt.Before(*value.StartsAt) {
		return ErrInvalidSchedule
	}
	return nil
}

func validType(value string) bool {
	switch value {
	case model.TypeNormal, model.TypeUpdate, model.TypeMaintenance, model.TypeWarning, model.TypeEvent, model.TypeSystem:
		return true
	default:
		return false
	}
}

func validDisplayMode(value string) bool {
	switch value {
	case model.DisplayModeNormal, model.DisplayModeBanner, model.DisplayModeModal, model.DisplayModeStartupModal:
		return true
	default:
		return false
	}
}

func validTarget(value string) bool {
	switch value {
	case model.TargetAll, model.TargetWeb, model.TargetAndroid, model.TargetIOS, model.TargetDesktop:
		return true
	default:
		return false
	}
}

func validQueryPlatform(value string) bool {
	switch value {
	case "web", "android", "ios", "windows", "macos", "linux", "desktop":
		return true
	default:
		return false
	}
}

func pagination(page, limit int) (int, int) {
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	return page, limit
}
