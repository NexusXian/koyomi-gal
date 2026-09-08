package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"backend/internal/changelog/dto"
	"backend/internal/changelog/model"
	"backend/internal/changelog/repository"

	"gorm.io/gorm"
)

var (
	ErrChangelogNotFound = errors.New("site changelog not found")
	ErrInvalidChangelog  = errors.New("invalid site changelog")
	ErrChangelogExists   = errors.New("site changelog version already exists")
)

type ChangelogService struct {
	changelogs *repository.ChangelogRepository
}

func NewChangelogService(changelogs *repository.ChangelogRepository) *ChangelogService {
	return &ChangelogService{changelogs: changelogs}
}

func (s *ChangelogService) List(ctx context.Context) ([]model.SiteChangelog, error) {
	values, _, err := s.changelogs.List(ctx, 1, 500)
	return values, err
}

func (s *ChangelogService) ListAdmin(ctx context.Context, page, limit int) ([]model.SiteChangelog, int64, int, int, error) {
	page, limit = pagination(page, limit)
	values, total, err := s.changelogs.List(ctx, page, limit)
	return values, total, page, limit, err
}

func (s *ChangelogService) Get(ctx context.Context, id uint) (*model.SiteChangelog, error) {
	value, err := s.changelogs.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, ErrChangelogNotFound
	}
	return value, nil
}

func (s *ChangelogService) Create(ctx context.Context, req *dto.ChangelogRequest) (*model.SiteChangelog, error) {
	value, err := changelogFromRequest(req, nil)
	if err != nil {
		return nil, err
	}
	existing, err := s.changelogs.FindByVersion(ctx, value.Version)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrChangelogExists
	}
	if err := s.changelogs.Create(ctx, value); err != nil {
		return nil, normalizeWriteError(err)
	}
	return value, nil
}

func (s *ChangelogService) Update(ctx context.Context, id uint, req *dto.ChangelogRequest) (*model.SiteChangelog, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	value, err := changelogFromRequest(req, current)
	if err != nil {
		return nil, err
	}
	if value.Version != current.Version {
		existing, err := s.changelogs.FindByVersion(ctx, value.Version)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, ErrChangelogExists
		}
	}
	value.ID, value.CreatedAt = current.ID, current.CreatedAt
	if err := s.changelogs.Update(ctx, value); err != nil {
		return nil, normalizeWriteError(err)
	}
	return value, nil
}

func (s *ChangelogService) Delete(ctx context.Context, id uint) error {
	deleted, err := s.changelogs.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrChangelogNotFound
	}
	return nil
}

func changelogFromRequest(req *dto.ChangelogRequest, current *model.SiteChangelog) (*model.SiteChangelog, error) {
	publishedAt := req.PublishedAt
	if publishedAt == nil && current != nil {
		publishedAt = &current.PublishedAt
	}
	if publishedAt == nil {
		now := time.Now()
		publishedAt = &now
	}
	value := &model.SiteChangelog{
		Version:     strings.TrimSpace(req.Version),
		Title:       strings.TrimSpace(req.Title),
		PublishedAt: *publishedAt,
	}
	for _, item := range req.Items {
		value.Items = append(value.Items, model.ChangelogItem{
			Type: item.Type, Text: strings.TrimSpace(item.Text),
		})
	}
	if err := validateChangelog(value); err != nil {
		return nil, err
	}
	return value, nil
}

func validateChangelog(value *model.SiteChangelog) error {
	if value.Version == "" || value.Title == "" || len(value.Items) == 0 {
		return ErrInvalidChangelog
	}
	for _, item := range value.Items {
		if item.Text == "" || !validItemType(item.Type) {
			return ErrInvalidChangelog
		}
	}
	return nil
}

func validItemType(value string) bool {
	switch value {
	case model.ItemTypeNew, model.ItemTypeImprove, model.ItemTypeFix:
		return true
	default:
		return false
	}
}

func normalizeWriteError(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) ||
		strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return ErrChangelogExists
	}
	return err
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
