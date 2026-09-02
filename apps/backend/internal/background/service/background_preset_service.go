package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"backend/internal/background/dto"
	"backend/internal/background/model"
	"backend/internal/background/repository"
)

var (
	ErrBackgroundPresetNotFound = errors.New("background preset not found")
	ErrInvalidPresetInput       = errors.New("invalid background preset input")
)

type BackgroundPresetService struct {
	presets   *repository.BackgroundPresetRepository
	publicURL string
}

func NewBackgroundPresetService(
	presets *repository.BackgroundPresetRepository,
	publicURL string,
) *BackgroundPresetService {
	return &BackgroundPresetService{presets: presets, publicURL: publicURL}
}

func (s *BackgroundPresetService) ListPublic(ctx context.Context) ([]model.BackgroundPreset, error) {
	items, err := s.presets.ListPublic(ctx)
	if err != nil {
		return nil, err
	}
	s.resolveAll(items)
	return items, nil
}

func (s *BackgroundPresetService) ListAdmin(ctx context.Context, page, limit int) ([]model.BackgroundPreset, int64, int, int, error) {
	page, limit = pagination(page, limit)
	presets, total, err := s.presets.ListAdmin(ctx, page, limit)
	if err != nil {
		return nil, 0, page, limit, err
	}
	s.resolveAll(presets)
	return presets, total, page, limit, nil
}

func (s *BackgroundPresetService) Get(ctx context.Context, id uint) (*model.BackgroundPreset, error) {
	preset, err := s.presets.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if preset == nil {
		return nil, ErrBackgroundPresetNotFound
	}
	preset.ImageURL = s.resolveImageURL(preset.ImageURL)
	return preset, nil
}

func (s *BackgroundPresetService) Create(ctx context.Context, req *dto.CreateBackgroundPresetRequest) (*model.BackgroundPreset, error) {
	key, err := generatePresetKey()
	if err != nil {
		return nil, err
	}
	preset := &model.BackgroundPreset{
		Key:       key,
		Name:      strings.TrimSpace(req.Name),
		ImageURL:  strings.TrimSpace(req.ImageURL),
		SortOrder: req.SortOrder,
		IsActive:  true,
	}
	if req.IsActive != nil {
		preset.IsActive = *req.IsActive
	}
	if err := validatePreset(preset); err != nil {
		return nil, err
	}
	if err := s.presets.Create(ctx, preset); err != nil {
		return nil, err
	}
	preset.ImageURL = s.resolveImageURL(preset.ImageURL)
	return preset, nil
}

func (s *BackgroundPresetService) Update(ctx context.Context, id uint, req *dto.UpdateBackgroundPresetRequest) (*model.BackgroundPreset, error) {
	preset, err := s.presets.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if preset == nil {
		return nil, ErrBackgroundPresetNotFound
	}
	preset.Name = strings.TrimSpace(req.Name)
	preset.ImageURL = strings.TrimSpace(req.ImageURL)
	preset.SortOrder = req.SortOrder
	preset.IsActive = *req.IsActive
	if err := validatePreset(preset); err != nil {
		return nil, err
	}
	if err := s.presets.Update(ctx, preset); err != nil {
		return nil, err
	}
	preset.ImageURL = s.resolveImageURL(preset.ImageURL)
	return preset, nil
}

func (s *BackgroundPresetService) Delete(ctx context.Context, id uint) error {
	deleted, err := s.presets.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrBackgroundPresetNotFound
	}
	return nil
}

// resolveImageURL keeps absolute URLs and joins object keys with the R2
// public origin so seeded rows work across environments.
func (s *BackgroundPresetService) resolveImageURL(value string) string {
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return strings.TrimRight(s.publicURL, "/") + "/" + strings.TrimLeft(value, "/")
}

func (s *BackgroundPresetService) resolveAll(presets []model.BackgroundPreset) {
	for i := range presets {
		presets[i].ImageURL = s.resolveImageURL(presets[i].ImageURL)
	}
}

func generatePresetKey() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate preset key: %w", err)
	}
	return "preset-" + hex.EncodeToString(buf), nil
}

func validatePreset(preset *model.BackgroundPreset) error {
	if preset.Name == "" || preset.ImageURL == "" {
		return ErrInvalidPresetInput
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
