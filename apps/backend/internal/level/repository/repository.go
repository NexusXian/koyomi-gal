package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	levelModel "backend/internal/level/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrLevelConfigNotFound = errors.New("level config not found")
var ErrExperienceRuleNotFound = errors.New("experience rule not found")

// Repository owns every level-system table. All experience mutations run
// inside Transaction so limits, logs, and the atomic total update commit
// together.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Transaction(ctx context.Context, fn func(tx *Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Repository{db: tx})
	})
}

// ---------------------------------------------------------------- level configs

func (r *Repository) ListLevelConfigs(ctx context.Context, enabledOnly bool) ([]levelModel.LevelConfig, error) {
	query := r.db.WithContext(ctx).Model(&levelModel.LevelConfig{})
	if enabledOnly {
		query = query.Where("is_enabled = ?", true)
	}
	var configs []levelModel.LevelConfig
	if err := query.Order("level ASC").Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("list level configs: %w", err)
	}
	return configs, nil
}

func (r *Repository) FindLevelConfigByID(ctx context.Context, id uint) (*levelModel.LevelConfig, error) {
	var config levelModel.LevelConfig
	err := r.db.WithContext(ctx).First(&config, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find level config by id: %w", err)
	}
	return &config, nil
}

func (r *Repository) FindLevelConfigByLevel(ctx context.Context, level int) (*levelModel.LevelConfig, error) {
	var config levelModel.LevelConfig
	err := r.db.WithContext(ctx).Where("level = ?", level).First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find level config by level: %w", err)
	}
	return &config, nil
}

func (r *Repository) CountUsersAtLevel(ctx context.Context, level int) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&levelModel.UserExperience{}).
		Where("current_level = ?", level).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count users at level: %w", err)
	}
	return count, nil
}

func (r *Repository) CreateLevelConfig(ctx context.Context, config *levelModel.LevelConfig) error {
	if err := r.db.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("create level config: %w", err)
	}
	return nil
}

func (r *Repository) UpdateLevelConfig(ctx context.Context, config *levelModel.LevelConfig) error {
	if err := r.db.WithContext(ctx).
		Model(&levelModel.LevelConfig{}).
		Where("id = ?", config.ID).
		Updates(map[string]any{
			"name":        config.Name,
			"min_exp":     config.MinExp,
			"icon_url":    config.IconURL,
			"color":       config.Color,
			"description": config.Description,
			"is_enabled":  config.IsEnabled,
			"updated_at":  time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("update level config: %w", err)
	}
	return nil
}

func (r *Repository) DisableLevelConfig(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).
		Model(&levelModel.LevelConfig{}).
		Where("id = ?", id).
		Updates(map[string]any{"is_enabled": false, "updated_at": time.Now()}).Error; err != nil {
		return fmt.Errorf("disable level config: %w", err)
	}
	return nil
}

func (r *Repository) DeleteLevelConfig(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&levelModel.LevelConfig{}, id)
	if result.Error != nil {
		return false, fmt.Errorf("delete level config: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// ------------------------------------------------------------- experience rules

func (r *Repository) ListExperienceRules(ctx context.Context) ([]levelModel.ExperienceRule, error) {
	var rules []levelModel.ExperienceRule
	err := r.db.WithContext(ctx).
		Model(&levelModel.ExperienceRule{}).
		Order("id ASC").
		Find(&rules).Error
	if err != nil {
		return nil, fmt.Errorf("list experience rules: %w", err)
	}
	return rules, nil
}

func (r *Repository) FindExperienceRuleByID(ctx context.Context, id uint) (*levelModel.ExperienceRule, error) {
	var rule levelModel.ExperienceRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find experience rule by id: %w", err)
	}
	return &rule, nil
}

func (r *Repository) FindExperienceRuleByEvent(
	ctx context.Context,
	eventType levelModel.EventType,
) (*levelModel.ExperienceRule, error) {
	var rule levelModel.ExperienceRule
	err := r.db.WithContext(ctx).Where("event_type = ?", eventType).First(&rule).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find experience rule by event: %w", err)
	}
	return &rule, nil
}

func (r *Repository) UpdateExperienceRule(ctx context.Context, rule *levelModel.ExperienceRule) error {
	if err := r.db.WithContext(ctx).
		Model(&levelModel.ExperienceRule{}).
		Where("id = ?", rule.ID).
		Updates(map[string]any{
			"name":             rule.Name,
			"exp":              rule.Exp,
			"daily_limit":      rule.DailyLimit,
			"daily_exp_limit":  rule.DailyExpLimit,
			"cooldown_seconds": rule.CooldownSeconds,
			"enabled":          rule.Enabled,
			"description":      rule.Description,
			"updated_at":       time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("update experience rule: %w", err)
	}
	return nil
}

// ------------------------------------------------------------- user experience

// FindExperienceForUpdate lazily creates the user_experience row and locks it
// FOR UPDATE so concurrent grants for one user serialize their limit checks.
func (r *Repository) FindExperienceForUpdate(ctx context.Context, userID uint) (*levelModel.UserExperience, error) {
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&levelModel.UserExperience{UserID: userID, TotalExp: 0, CurrentLevel: 1}).Error; err != nil {
		return nil, fmt.Errorf("lazy create user experience: %w", err)
	}
	var experience levelModel.UserExperience
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&experience, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lock user experience: %w", err)
	}
	return &experience, nil
}

// AddExperience atomically adds delta and clamps the total at zero; it returns
// the resulting total.
func (r *Repository) AddExperience(ctx context.Context, userID uint, delta int64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&levelModel.UserExperience{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"total_exp":  gorm.Expr("GREATEST(total_exp + ?, 0)", delta),
			"updated_at": time.Now(),
		}).Error
	if err != nil {
		return 0, fmt.Errorf("add experience: %w", err)
	}
	if err := r.db.WithContext(ctx).
		Model(&levelModel.UserExperience{}).
		Where("user_id = ?", userID).
		Pluck("total_exp", &total).Error; err != nil {
		return 0, fmt.Errorf("reload experience total: %w", err)
	}
	return total, nil
}

func (r *Repository) UpdateCurrentLevel(ctx context.Context, userID uint, level int) error {
	if err := r.db.WithContext(ctx).
		Model(&levelModel.UserExperience{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{"current_level": level, "updated_at": time.Now()}).Error; err != nil {
		return fmt.Errorf("update current level: %w", err)
	}
	return nil
}

func (r *Repository) FindExperience(ctx context.Context, userID uint) (*levelModel.UserExperience, error) {
	var experience levelModel.UserExperience
	err := r.db.WithContext(ctx).First(&experience, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user experience: %w", err)
	}
	return &experience, nil
}

// ------------------------------------------------------------- experience logs

func (r *Repository) CreateExperienceLog(ctx context.Context, log *levelModel.ExperienceLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("create experience log: %w", err)
	}
	return nil
}

func (r *Repository) FindExperienceLogByIdempotencyKey(
	ctx context.Context,
	key string,
) (*levelModel.ExperienceLog, error) {
	var log levelModel.ExperienceLog
	err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find experience log by idempotency key: %w", err)
	}
	return &log, nil
}

func (r *Repository) CountExperienceLogsSince(
	ctx context.Context,
	userID uint,
	eventType levelModel.EventType,
	since time.Time,
) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&levelModel.ExperienceLog{}).
		Where("user_id = ? AND event_type = ? AND created_at >= ?", userID, eventType, since).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count experience logs since: %w", err)
	}
	return count, nil
}

func (r *Repository) SumExperienceSince(
	ctx context.Context,
	userID uint,
	eventType levelModel.EventType,
	since time.Time,
) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&levelModel.ExperienceLog{}).
		Select("COALESCE(SUM(exp_delta), 0)").
		Where("user_id = ? AND event_type = ? AND created_at >= ?", userID, eventType, since).
		Scan(&total).Error
	if err != nil {
		return 0, fmt.Errorf("sum experience since: %w", err)
	}
	return total, nil
}

func (r *Repository) LastExperienceLogAt(
	ctx context.Context,
	userID uint,
	eventType levelModel.EventType,
) (*time.Time, error) {
	var log levelModel.ExperienceLog
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND event_type = ?", userID, eventType).
		Order("created_at DESC").
		First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find last experience log: %w", err)
	}
	return &log.CreatedAt, nil
}

func (r *Repository) ListExperienceLogs(
	ctx context.Context,
	userID uint,
	page, limit int,
) ([]levelModel.ExperienceLog, int64, error) {
	base := r.db.WithContext(ctx).
		Model(&levelModel.ExperienceLog{}).
		Where("user_id = ?", userID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count experience logs: %w", err)
	}
	var logs []levelModel.ExperienceLog
	err := base.
		Order("created_at DESC").Order("id DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list experience logs: %w", err)
	}
	return logs, total, nil
}

// --------------------------------------------------------------- level summaries

type LevelSummaryRow struct {
	UserID      uint
	Level       int
	Name        string
	IconURL     string
	Color       string
}

// ListLevelSummaries returns level display data for the given users, joined
// through user_experience -> level_configs. Users without an experience row
// resolve to the default level (LV1).
func (r *Repository) ListLevelSummaries(ctx context.Context, userIDs []uint) ([]LevelSummaryRow, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var rows []LevelSummaryRow
	err := r.db.WithContext(ctx).
		Table("users").
		Select("users.id AS user_id, level_configs.level, level_configs.name, level_configs.icon_url, level_configs.color").
		Joins("LEFT JOIN user_experience ON user_experience.user_id = users.id").
		Joins("JOIN level_configs ON level_configs.level = COALESCE(user_experience.current_level, 1) AND level_configs.is_enabled = TRUE").
		Where("users.id IN ?", userIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list level summaries: %w", err)
	}
	return rows, nil
}

// ------------------------------------------------------------------- checkins

func (r *Repository) FindCheckin(
	ctx context.Context,
	userID uint,
	date time.Time,
) (*levelModel.UserCheckin, error) {
	var checkin levelModel.UserCheckin
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND checkin_date = ?", userID, date.Format("2006-01-02")).
		First(&checkin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find checkin: %w", err)
	}
	return &checkin, nil
}

// CreateCheckin inserts the checkin and reports whether a new row was created;
// false means the user already checked in on that date.
func (r *Repository) CreateCheckin(ctx context.Context, checkin *levelModel.UserCheckin) (bool, error) {
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(checkin).Error
	if err != nil {
		return false, fmt.Errorf("create checkin: %w", err)
	}
	return checkin.ID != 0, nil
}

// ----------------------------------------------------------------------- users

func (r *Repository) FindUserIDByUsername(ctx context.Context, username string) (uint, bool, error) {
	var id uint
	err := r.db.WithContext(ctx).
		Table("users").
		Where("username = ?", username).
		Pluck("id", &id).Error
	if err != nil {
		return 0, false, fmt.Errorf("find user id by username: %w", err)
	}
	if id == 0 {
		return 0, false, nil
	}
	return id, true, nil
}

func (r *Repository) UserExists(ctx context.Context, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("users").
		Where("id = ?", userID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}
	return count > 0, nil
}
