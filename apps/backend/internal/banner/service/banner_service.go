package service

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"backend/internal/banner/dto"
	"backend/internal/banner/model"
	"backend/internal/banner/repository"
	homeService "backend/internal/home/service"

	"github.com/redis/go-redis/v9"
)

var (
	ErrBannerNotFound     = errors.New("banner not found")
	ErrInvalidBannerInput = errors.New("invalid banner input")
	ErrInvalidBannerLink  = errors.New("invalid banner link")
	ErrInvalidSchedule    = errors.New("invalid banner schedule")
)

type BannerService struct {
	banners *repository.BannerRepository
	cache   *redis.Client
}

func NewBannerService(banners *repository.BannerRepository, cache *redis.Client) *BannerService {
	return &BannerService{banners: banners, cache: cache}
}

func (s *BannerService) ListPublic(ctx context.Context) ([]model.Banner, error) {
	return s.banners.ListPublic(ctx, 0)
}

func (s *BannerService) ListAdmin(ctx context.Context, page, limit int) ([]model.Banner, int64, int, int, error) {
	page, limit = pagination(page, limit)
	banners, total, err := s.banners.ListAdmin(ctx, page, limit)
	return banners, total, page, limit, err
}

func (s *BannerService) Get(ctx context.Context, id uint) (*model.Banner, error) {
	banner, err := s.banners.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if banner == nil {
		return nil, ErrBannerNotFound
	}
	return banner, nil
}

func (s *BannerService) Create(ctx context.Context, req *dto.CreateBannerRequest) (*model.Banner, error) {
	banner := &model.Banner{
		Title: strings.TrimSpace(req.Title), Subtitle: strings.TrimSpace(req.Subtitle), ImageURL: strings.TrimSpace(req.ImageURL),
		LinkType:  strings.TrimSpace(req.LinkType),
		LinkValue: strings.TrimSpace(req.LinkValue), SortOrder: req.SortOrder,
		IsActive: true, StartAt: req.StartAt, EndAt: req.EndAt,
	}
	if req.IsActive != nil {
		banner.IsActive = *req.IsActive
	}
	if err := validateBanner(banner); err != nil {
		return nil, err
	}
	if err := s.banners.Create(ctx, banner); err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	return banner, nil
}

func (s *BannerService) Update(ctx context.Context, id uint, req *dto.UpdateBannerRequest) (*model.Banner, error) {
	banner, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	banner.Title = strings.TrimSpace(req.Title)
	banner.Subtitle = strings.TrimSpace(req.Subtitle)
	banner.ImageURL = strings.TrimSpace(req.ImageURL)
	banner.LinkType = strings.TrimSpace(req.LinkType)
	banner.LinkValue = strings.TrimSpace(req.LinkValue)
	banner.SortOrder = req.SortOrder
	banner.IsActive = *req.IsActive
	banner.StartAt = req.StartAt
	banner.EndAt = req.EndAt
	if err := validateBanner(banner); err != nil {
		return nil, err
	}
	if err := s.banners.Update(ctx, banner); err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	return banner, nil
}

func (s *BannerService) Delete(ctx context.Context, id uint) error {
	deleted, err := s.banners.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrBannerNotFound
	}
	s.invalidate(ctx)
	return nil
}

func (s *BannerService) invalidate(ctx context.Context) {
	if s.cache != nil {
		_ = s.cache.Del(context.WithoutCancel(ctx), homeService.CacheKey).Err()
	}
}

func validateBanner(banner *model.Banner) error {
	if banner.Title == "" || banner.ImageURL == "" {
		return ErrInvalidBannerInput
	}
	if banner.StartAt != nil && banner.EndAt != nil && !banner.StartAt.Before(*banner.EndAt) {
		return ErrInvalidSchedule
	}
	switch banner.LinkType {
	case model.LinkTypeNone:
		if banner.LinkValue != "" {
			return ErrInvalidBannerLink
		}
	case model.LinkTypeURL:
		parsed, err := url.ParseRequestURI(banner.LinkValue)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return ErrInvalidBannerLink
		}
	case model.LinkTypeGalgame, model.LinkTypePost, model.LinkTypeNews:
		id, err := strconv.ParseUint(banner.LinkValue, 10, 64)
		if err != nil || id == 0 {
			return ErrInvalidBannerLink
		}
	default:
		return ErrInvalidBannerLink
	}
	return nil
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
