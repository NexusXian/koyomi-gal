package service

import (
	"context"
	"errors"
	"strings"

	leveldto "backend/internal/level/dto"
	levelModel "backend/internal/level/model"
	"backend/internal/level/repository"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrLevelConfigNotFound  = errors.New("level config not found")
	ErrLevelConfigExists    = errors.New("level config already exists")
	ErrLevelConfigInUse     = errors.New("level config is in use and can only be disabled")
	ErrLevelConfigInvalid   = errors.New("invalid level config")
	ErrExperienceRuleNotFound = repository.ErrExperienceRuleNotFound
	ErrExperienceRuleInvalid  = errors.New("invalid experience rule")
)

// LevelConfigService manages admin-editable level definitions and experience
// rules; both caches are invalidated after every write.
type LevelConfigService struct {
	repository *repository.Repository
	cache      *redis.Client
}

func NewLevelConfigService(repository *repository.Repository, cache *redis.Client) *LevelConfigService {
	return &LevelConfigService{repository: repository, cache: cache}
}

func (s *LevelConfigService) ListLevels(ctx context.Context) ([]levelModel.LevelConfig, error) {
	return s.repository.ListLevelConfigs(ctx, false)
}

func (s *LevelConfigService) CreateLevel(
	ctx context.Context,
	req *leveldto.CreateLevelConfigRequest,
) (*levelModel.LevelConfig, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrLevelConfigInvalid
	}
	existing, err := s.repository.FindLevelConfigByLevel(ctx, req.Level)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrLevelConfigExists
	}
	config := &levelModel.LevelConfig{
		Level:       req.Level,
		Name:        name,
		MinExp:      req.MinExp,
		IconURL:     strings.TrimSpace(req.IconURL),
		Color:       strings.TrimSpace(req.Color),
		Description: strings.TrimSpace(req.Description),
		IsEnabled:   true,
	}
	if req.IsEnabled != nil {
		config.IsEnabled = *req.IsEnabled
	}
	if err := s.repository.CreateLevelConfig(ctx, config); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrLevelConfigExists
		}
		return nil, err
	}
	s.invalidate(ctx)
	return config, nil
}

func (s *LevelConfigService) UpdateLevel(
	ctx context.Context,
	id uint,
	req *leveldto.UpdateLevelConfigRequest,
) (*levelModel.LevelConfig, error) {
	config, err := s.repository.FindLevelConfigByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, ErrLevelConfigNotFound
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrLevelConfigInvalid
	}
	config.Name = name
	config.MinExp = req.MinExp
	config.IconURL = strings.TrimSpace(req.IconURL)
	config.Color = strings.TrimSpace(req.Color)
	config.Description = strings.TrimSpace(req.Description)
	config.IsEnabled = *req.IsEnabled
	if err := s.repository.UpdateLevelConfig(ctx, config); err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	return config, nil
}

// DeleteLevel removes a level definition. Levels already held by users are
// only disabled so no user ends up referencing a missing level.
func (s *LevelConfigService) DeleteLevel(ctx context.Context, id uint) (disabled bool, err error) {
	config, err := s.repository.FindLevelConfigByID(ctx, id)
	if err != nil {
		return false, err
	}
	if config == nil {
		return false, ErrLevelConfigNotFound
	}
	usersAtLevel, err := s.repository.CountUsersAtLevel(ctx, config.Level)
	if err != nil {
		return false, err
	}
	if usersAtLevel > 0 {
		if err := s.repository.DisableLevelConfig(ctx, id); err != nil {
			return false, err
		}
		s.invalidate(ctx)
		return true, nil
	}
	deleted, err := s.repository.DeleteLevelConfig(ctx, id)
	if err != nil || !deleted {
		return false, err
	}
	s.invalidate(ctx)
	return false, nil
}

func (s *LevelConfigService) ListRules(ctx context.Context) ([]levelModel.ExperienceRule, error) {
	return s.repository.ListExperienceRules(ctx)
}

func (s *LevelConfigService) UpdateRule(
	ctx context.Context,
	id uint,
	req *leveldto.UpdateExperienceRuleRequest,
) (*levelModel.ExperienceRule, error) {
	rule, err := s.repository.FindExperienceRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrExperienceRuleNotFound
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, ErrExperienceRuleInvalid
		}
		rule.Name = name
	}
	if req.Exp != nil {
		if *req.Exp < 0 {
			return nil, ErrExperienceRuleInvalid
		}
		rule.Exp = *req.Exp
	}
	if req.DailyLimit != nil {
		if *req.DailyLimit < 0 {
			return nil, ErrExperienceRuleInvalid
		}
		rule.DailyLimit = *req.DailyLimit
	}
	if req.DailyExpLimit != nil {
		if *req.DailyExpLimit < 0 {
			return nil, ErrExperienceRuleInvalid
		}
		rule.DailyExpLimit = *req.DailyExpLimit
	}
	if req.CooldownSeconds != nil {
		if *req.CooldownSeconds < 0 {
			return nil, ErrExperienceRuleInvalid
		}
		rule.CooldownSeconds = *req.CooldownSeconds
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Description != nil {
		rule.Description = strings.TrimSpace(*req.Description)
	}
	if err := s.repository.UpdateExperienceRule(ctx, rule); err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	return rule, nil
}

func (s *LevelConfigService) invalidate(ctx context.Context) {
	InvalidateCaches(ctx, s.cache)
}

func isUniqueViolation(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey) ||
		strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}
