package service

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	announcementModel "backend/internal/announcement/model"
	announcementRepo "backend/internal/announcement/repository"
	"backend/internal/apprelease/dto"
	"backend/internal/apprelease/model"
	"backend/internal/apprelease/repository"
	rbacService "backend/internal/rbac/service"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var (
	ErrAppReleaseNotFound      = errors.New("app release not found")
	ErrInvalidAppRelease       = errors.New("invalid app release")
	ErrInvalidDownloadURL      = errors.New("invalid download url")
	ErrInvalidChecksum         = errors.New("invalid file checksum")
	ErrAppReleaseForbidden     = errors.New("app release permission denied")
	ErrVersionExists           = errors.New("app release version already exists")
	ErrInvalidAnnouncementLink = errors.New("invalid announcement link")
	ErrAnnouncementLinked      = errors.New("announcement already linked")
)

const PermissionAppReleasePublish = "app_release:publish"

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type AppReleaseService struct {
	releases *repository.AppReleaseRepository
	rbac     *rbacService.RBACService
}

func NewAppReleaseService(releases *repository.AppReleaseRepository, rbac *rbacService.RBACService) *AppReleaseService {
	return &AppReleaseService{releases: releases, rbac: rbac}
}

func (s *AppReleaseService) Latest(ctx context.Context, platform string, versionCode int64) (*model.AppRelease, error) {
	if !validPlatform(platform) || versionCode < 0 {
		return nil, ErrInvalidAppRelease
	}
	return s.releases.FindLatestPublished(ctx, platform, versionCode)
}

func (s *AppReleaseService) ListAdmin(ctx context.Context, page, limit int) ([]model.AppRelease, int64, int, int, error) {
	page, limit = pagination(page, limit)
	values, total, err := s.releases.ListAdmin(ctx, page, limit)
	return values, total, page, limit, err
}

func (s *AppReleaseService) Get(ctx context.Context, id uint) (*model.AppRelease, error) {
	value, err := s.releases.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, ErrAppReleaseNotFound
	}
	return value, nil
}

func (s *AppReleaseService) Create(ctx context.Context, actorID uint, req *dto.AppReleaseRequest) (*model.AppRelease, error) {
	if req.VersionCode == nil {
		return nil, ErrInvalidAppRelease
	}
	value := releaseFromRequest(req, nil)
	if err := validateAppRelease(value); err != nil {
		return nil, err
	}
	if value.Status == model.StatusPublished {
		if err := s.requirePublish(ctx, actorID); err != nil {
			return nil, err
		}
	}
	err := s.releases.Transaction(ctx, func(db *gorm.DB) error {
		releases := repository.NewAppReleaseRepository(db)
		if err := releases.Create(ctx, value); err != nil {
			return err
		}
		if req.CreateAnnouncement && value.Status == model.StatusPublished && value.AnnouncementID == nil {
			return createLinkedAnnouncement(ctx, db, releases, value, actorID)
		}
		return nil
	})
	if err != nil {
		return nil, normalizeWriteError(err)
	}
	return value, nil
}

func (s *AppReleaseService) Update(ctx context.Context, actorID, id uint, req *dto.AppReleaseRequest) (*model.AppRelease, error) {
	if req.VersionCode == nil {
		return nil, ErrInvalidAppRelease
	}
	var desired *model.AppRelease
	err := s.releases.Transaction(ctx, func(db *gorm.DB) error {
		releases := repository.NewAppReleaseRepository(db)
		current, err := releases.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return ErrAppReleaseNotFound
		}
		desired = releaseFromRequest(req, current)
		desired.ID, desired.CreatedAt = current.ID, current.CreatedAt
		if err := validateAppRelease(desired); err != nil {
			return err
		}
		if current.Status != desired.Status || !sameTime(current.PublishedAt, desired.PublishedAt) {
			if err := s.requirePublish(ctx, actorID); err != nil {
				return err
			}
		}
		if current.AnnouncementManaged && current.AnnouncementID != nil &&
			!sameUint(current.AnnouncementID, desired.AnnouncementID) {
			if err := setManagedAnnouncementPublished(ctx, db, current, false); err != nil {
				return err
			}
		}
		if err := releases.Update(ctx, desired); err != nil {
			return err
		}
		if desired.AnnouncementManaged && desired.AnnouncementID != nil {
			if desired.Status == model.StatusPublished {
				if err := syncManagedAnnouncement(ctx, db, desired, true); err != nil {
					return err
				}
			} else if err := setManagedAnnouncementPublished(ctx, db, desired, false); err != nil {
				return err
			}
		}
		if req.CreateAnnouncement && desired.Status == model.StatusPublished && desired.AnnouncementID == nil {
			return createLinkedAnnouncement(ctx, db, releases, desired, actorID)
		}
		return nil
	})
	if err != nil {
		return nil, normalizeWriteError(err)
	}
	return desired, nil
}

func (s *AppReleaseService) Delete(ctx context.Context, id uint) error {
	return s.releases.Transaction(ctx, func(db *gorm.DB) error {
		releases := repository.NewAppReleaseRepository(db)
		current, err := releases.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return ErrAppReleaseNotFound
		}
		if err := setManagedAnnouncementPublished(ctx, db, current, false); err != nil {
			return err
		}
		_, err = releases.Delete(ctx, id)
		return err
	})
}

func (s *AppReleaseService) Publish(ctx context.Context, id, actorID uint, createAnnouncement bool) (*model.AppRelease, error) {
	var value *model.AppRelease
	err := s.releases.Transaction(ctx, func(db *gorm.DB) error {
		releases := repository.NewAppReleaseRepository(db)
		current, err := releases.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return ErrAppReleaseNotFound
		}
		if current.AnnouncementID == nil {
			current.AnnouncementManaged = false
		}
		now := time.Now()
		current.Status, current.PublishedAt = model.StatusPublished, &now
		if err := validateAppRelease(current); err != nil {
			return err
		}
		if err := releases.Update(ctx, current); err != nil {
			return err
		}
		if current.AnnouncementManaged && current.AnnouncementID != nil {
			if err := syncManagedAnnouncement(ctx, db, current, true); err != nil {
				return err
			}
		} else if createAnnouncement && current.AnnouncementID == nil {
			if err := createLinkedAnnouncement(ctx, db, releases, current, actorID); err != nil {
				return err
			}
		}
		value = current
		return nil
	})
	if err != nil {
		return nil, normalizeWriteError(err)
	}
	return value, nil
}

func (s *AppReleaseService) Disable(ctx context.Context, id uint) (*model.AppRelease, error) {
	var value *model.AppRelease
	err := s.releases.Transaction(ctx, func(db *gorm.DB) error {
		releases := repository.NewAppReleaseRepository(db)
		current, err := releases.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return ErrAppReleaseNotFound
		}
		if current.AnnouncementID == nil {
			current.AnnouncementManaged = false
		}
		current.Status = model.StatusDisabled
		if err := releases.Update(ctx, current); err != nil {
			return err
		}
		if err := setManagedAnnouncementPublished(ctx, db, current, false); err != nil {
			return err
		}
		value = current
		return nil
	})
	if err != nil {
		return nil, normalizeWriteError(err)
	}
	return value, nil
}

func (s *AppReleaseService) requirePublish(ctx context.Context, actorID uint) error {
	allowed, err := s.rbac.HasPermission(ctx, actorID, PermissionAppReleasePublish)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrAppReleaseForbidden
	}
	return nil
}

func releaseFromRequest(req *dto.AppReleaseRequest, current *model.AppRelease) *model.AppRelease {
	status := req.Status
	if current != nil && status == "" {
		status = current.Status
	}
	if status == "" {
		status = model.StatusDraft
	}
	publishedAt := req.PublishedAt
	if current != nil && publishedAt == nil {
		publishedAt = current.PublishedAt
	}
	if status == model.StatusPublished && publishedAt == nil {
		if current != nil && current.PublishedAt != nil {
			publishedAt = current.PublishedAt
		} else {
			now := time.Now()
			publishedAt = &now
		}
	}
	var checksum *string
	if req.SHA256 != nil {
		trimmed := strings.TrimSpace(*req.SHA256)
		if trimmed != "" {
			checksum = &trimmed
		}
	}
	value := &model.AppRelease{
		Platform: req.Platform, VersionName: strings.TrimSpace(req.VersionName), VersionCode: *req.VersionCode,
		Title: strings.TrimSpace(req.Title), Changelog: strings.TrimSpace(req.Changelog),
		DownloadURL: strings.TrimSpace(req.DownloadURL), FileSize: req.FileSize, FileSHA256: checksum,
		MinimumVersionCode: req.MinimumVersionCode, ForceUpdate: req.ForceUpdate,
		Status: status, PublishedAt: publishedAt, AnnouncementID: req.AnnouncementID,
	}
	if current != nil && req.AnnouncementID == nil {
		value.AnnouncementID = current.AnnouncementID
	}
	if current != nil && value.AnnouncementID != nil && sameUint(current.AnnouncementID, value.AnnouncementID) {
		value.AnnouncementManaged = current.AnnouncementManaged
	}
	return value
}

func createLinkedAnnouncement(ctx context.Context, db *gorm.DB, releases *repository.AppReleaseRepository, release *model.AppRelease, actorID uint) error {
	announcement := &announcementModel.Announcement{
		Title: release.Title, Content: release.Changelog, Type: announcementModel.TypeUpdate,
		DisplayMode: announcementModel.DisplayModeStartupModal, Target: announcementModel.TargetAndroid,
		Priority: 0, StartsAt: release.PublishedAt, Dismissible: !release.ForceUpdate,
		Published: true, CreatedBy: &actorID,
	}
	if err := announcementRepo.NewAnnouncementRepository(db).Create(ctx, announcement); err != nil {
		return err
	}
	release.AnnouncementID = &announcement.ID
	release.AnnouncementManaged = true
	return releases.Update(ctx, release)
}

func syncManagedAnnouncement(ctx context.Context, db *gorm.DB, release *model.AppRelease, published bool) error {
	if !release.AnnouncementManaged || release.AnnouncementID == nil {
		return nil
	}
	return announcementRepo.NewAnnouncementRepository(db).SyncGenerated(
		ctx,
		*release.AnnouncementID,
		release.Title,
		release.Changelog,
		release.PublishedAt,
		!release.ForceUpdate,
		published,
	)
}

func setManagedAnnouncementPublished(ctx context.Context, db *gorm.DB, release *model.AppRelease, published bool) error {
	if !release.AnnouncementManaged || release.AnnouncementID == nil {
		return nil
	}
	return announcementRepo.NewAnnouncementRepository(db).SetPublished(ctx, *release.AnnouncementID, published)
}

func validateAppRelease(value *model.AppRelease) error {
	if !validPlatform(value.Platform) || value.VersionName == "" || value.VersionCode < 0 ||
		value.Title == "" || value.Changelog == "" || value.MinimumVersionCode < 0 ||
		value.MinimumVersionCode > value.VersionCode || !validStatus(value.Status) ||
		(value.FileSize != nil && *value.FileSize < 0) ||
		(value.Status == model.StatusPublished && value.PublishedAt == nil) {
		return ErrInvalidAppRelease
	}
	parsed, err := url.ParseRequestURI(value.DownloadURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ErrInvalidDownloadURL
	}
	if value.FileSHA256 != nil && !sha256Pattern.MatchString(*value.FileSHA256) {
		return ErrInvalidChecksum
	}
	return nil
}

func validPlatform(value string) bool {
	switch value {
	case model.PlatformAndroid, model.PlatformIOS, model.PlatformWindows, model.PlatformMacOS, model.PlatformLinux:
		return true
	default:
		return false
	}
}

func validStatus(value string) bool {
	switch value {
	case model.StatusDraft, model.StatusPublished, model.StatusDisabled:
		return true
	default:
		return false
	}
}

func normalizeWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23505" && pgErr.ConstraintName == "uk_app_releases_platform_version_code":
			return ErrVersionExists
		case pgErr.Code == "23505" && pgErr.ConstraintName == "uk_app_releases_announcement_id":
			return ErrAnnouncementLinked
		case pgErr.Code == "23503" && pgErr.ConstraintName == "app_releases_announcement_id_fkey":
			return ErrInvalidAnnouncementLink
		}
	}
	return err
}

func sameTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func sameUint(left, right *uint) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
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
