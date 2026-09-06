package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	leveldto "backend/internal/level/dto"
	levelModel "backend/internal/level/model"
	"backend/internal/level/repository"
	notificationModel "backend/internal/notification/model"
	notificationService "backend/internal/notification/service"
	"backend/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	ErrInvalidAdjustment     = errors.New("invalid experience adjustment")
	ErrInvalidGrantInput     = errors.New("invalid experience grant input")
	ErrUserNotFound          = errors.New("user not found")
)

// Deny reasons attached to GrantResult when a grant is legitimately skipped.
// They never fail the calling business flow.
const (
	DenyReasonRuleDisabled   = "rule_disabled"
	DenyReasonRuleMissing    = "rule_missing"
	DenyReasonDailyLimit     = "daily_limit_reached"
	DenyReasonDailyExpLimit  = "daily_exp_limit_reached"
	DenyReasonCooldown       = "cooldown"
	DenyReasonDuplicate      = "duplicate"
	DenyReasonExperienceFloor = "experience_floor_reached"
)

// Cache keys for low-frequency-write, high-frequency-read configuration.
const (
	CacheKeyLevelConfigs    = "level:configs"
	CacheKeyExperienceRules = "experience:rules"
	configCacheTTL          = 5 * time.Minute
)

// GrantResult reports the outcome of a Grant or Adjust call. Granted=false
// with a Reason means the change was legitimately skipped, not failed.
type GrantResult struct {
	Granted    bool
	Reason     string
	ExpGained  int64
	TotalExp   int64
	Level      int
	LevelName  string
	LeveledUp  bool
	OldLevel   int
	NewLevel   int
}

type grantOptions struct {
	actorID     *uint
	description string
}

// GrantOption customizes optional grant metadata.
type GrantOption func(*grantOptions)

// WithActor records the triggering actor in the idempotency key so per-actor
// events (e.g. comment_liked) reward once per actor and target.
func WithActor(actorID uint) GrantOption {
	return func(options *grantOptions) { options.actorID = &actorID }
}

// WithDescription overrides the experience log description.
func WithDescription(description string) GrantOption {
	return func(options *grantOptions) { options.description = description }
}

// ExperienceService is the single entry point for every experience change.
// Business modules report events; this service alone decides whether, how
// much, and with which limits experience is granted.
type ExperienceService struct {
	repository    *repository.Repository
	cache         *redis.Client
	notifications *notificationService.NotificationService
}

func NewExperienceService(
	repository *repository.Repository,
	cache *redis.Client,
	notifications *notificationService.NotificationService,
) *ExperienceService {
	return &ExperienceService{repository: repository, cache: cache, notifications: notifications}
}

// Grant awards experience for an event after checking the rule, daily limits,
// cooldown, and idempotency. All writes happen in one transaction; the level
// cache column is refreshed and an upgrade notification is sent post-commit.
func (s *ExperienceService) Grant(
	ctx context.Context,
	userID uint,
	eventType levelModel.EventType,
	sourceType string,
	sourceID uint,
	opts ...GrantOption,
) (*GrantResult, error) {
	if userID == 0 || !levelModel.ValidEventType(eventType) {
		return nil, ErrInvalidGrantInput
	}
	options := &grantOptions{}
	for _, opt := range opts {
		opt(options)
	}

	result := &GrantResult{}
	rule, err := s.rule(ctx, eventType)
	if err != nil {
		return nil, err
	}
	switch {
	case rule == nil:
		result.Reason = DenyReasonRuleMissing
		return s.denyWithProfile(ctx, userID, result)
	case !rule.Enabled:
		result.Reason = DenyReasonRuleDisabled
		return s.denyWithProfile(ctx, userID, result)
	}

	idempotencyKey := idempotencyKey(eventType, sourceType, sourceID, options.actorID)
	description := options.description
	if description == "" {
		description = rule.Name
	}

	denied := ""
	var (
		expGain       int64
		newTotal      int64
		oldLevel      int
		newLevel      int
		leveledUp     bool
		newLevelFound bool
	)
	err = s.repository.Transaction(ctx, func(tx *repository.Repository) error {
		// Serialize concurrent grants per user before reading counters.
		experience, err := tx.FindExperienceForUpdate(ctx, userID)
		if err != nil {
			return err
		}
		if experience == nil {
			return ErrUserNotFound
		}
		if idempotencyKey != "" {
			existing, err := tx.FindExperienceLogByIdempotencyKey(ctx, idempotencyKey)
			if err != nil {
				return err
			}
			if existing != nil {
				denied = DenyReasonDuplicate
				return nil
			}
		}
		now := time.Now()
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if rule.DailyLimit > 0 {
			count, err := tx.CountExperienceLogsSince(ctx, userID, eventType, dayStart)
			if err != nil {
				return err
			}
			if count >= int64(rule.DailyLimit) {
				denied = DenyReasonDailyLimit
				return nil
			}
		}
		if rule.CooldownSeconds > 0 {
			lastAt, err := tx.LastExperienceLogAt(ctx, userID, eventType)
			if err != nil {
				return err
			}
			if lastAt != nil && lastAt.Add(time.Duration(rule.CooldownSeconds)*time.Second).After(now) {
				denied = DenyReasonCooldown
				return nil
			}
		}
		expGain = int64(rule.Exp)
		if rule.DailyExpLimit > 0 {
			gained, err := tx.SumExperienceSince(ctx, userID, eventType, dayStart)
			if err != nil {
				return err
			}
			remaining := int64(rule.DailyExpLimit) - gained
			if remaining <= 0 {
				denied = DenyReasonDailyExpLimit
				return nil
			}
			if expGain > remaining {
				expGain = remaining
			}
		}

		log := &levelModel.ExperienceLog{
			UserID:         userID,
			EventType:      eventType,
			ExpDelta:       expGain,
			SourceType:     sourceType,
			IdempotencyKey: idempotencyKeyOrNil(idempotencyKey),
			Description:    description,
		}
		if sourceID != 0 {
			sourceID := sourceID
			log.SourceID = &sourceID
		}
		if options.actorID != nil {
			log.OperatorID = options.actorID
		}
		if err := tx.CreateExperienceLog(ctx, log); err != nil {
			return err
		}
		newTotal, err = tx.AddExperience(ctx, userID, expGain)
		if err != nil {
			return err
		}

		configs := s.levelConfigs(ctx)
		current, _ := resolveLevel(configs, newTotal)
		if current == nil {
			return ErrLevelConfigNotFound
		}
		newLevelFound = true
		if current.Level != experience.CurrentLevel {
			if err := tx.UpdateCurrentLevel(ctx, userID, current.Level); err != nil {
				return err
			}
			oldLevel, newLevel = experience.CurrentLevel, current.Level
			leveledUp = newLevel > oldLevel
		} else {
			newLevel = current.Level
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if denied != "" {
		result.Reason = denied
		return s.denyWithProfile(ctx, userID, result)
	}

	result.Granted = true
	result.ExpGained = expGain
	result.TotalExp = newTotal
	result.LeveledUp = leveledUp
	result.OldLevel = oldLevel
	result.NewLevel = newLevel
	if newLevelFound {
		result.Level = newLevel
		if configs := s.levelConfigs(ctx); configs != nil {
			if current, _ := resolveLevel(configs, newTotal); current != nil {
				result.LevelName = current.Name
			}
		}
	}
	if leveledUp {
		s.notifyLevelUp(ctx, userID, oldLevel, newLevel)
	}
	return result, nil
}

// Adjust manually adds or removes experience. The total clamps at zero, the
// change is logged with the operator, and the level is recomputed so the user
// may level up or down.
func (s *ExperienceService) Adjust(
	ctx context.Context,
	userID uint,
	exp int64,
	reason string,
	operatorID uint,
) (*GrantResult, error) {
	if exp == 0 || strings.TrimSpace(reason) == "" || userID == 0 || operatorID == 0 {
		return nil, ErrInvalidAdjustment
	}
	exists, err := s.repository.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	result := &GrantResult{}
	var (
		actualDelta  int64
		newTotal     int64
		oldLevel     int
		newLevel     int
		leveledUp    bool
		newLevelSeen bool
	)
	err = s.repository.Transaction(ctx, func(tx *repository.Repository) error {
		experience, err := tx.FindExperienceForUpdate(ctx, userID)
		if err != nil {
			return err
		}
		if experience == nil {
			return ErrUserNotFound
		}
		target := experience.TotalExp + exp
		if target < 0 {
			target = 0
		}
		actualDelta = target - experience.TotalExp
		if actualDelta == 0 {
			return nil
		}
		log := &levelModel.ExperienceLog{
			UserID:      userID,
			EventType:   levelModel.EventAdminAdjustment,
			ExpDelta:    actualDelta,
			SourceType:  levelModel.SourceAdmin,
			Description: strings.TrimSpace(reason),
			OperatorID:  &operatorID,
		}
		if err := tx.CreateExperienceLog(ctx, log); err != nil {
			return err
		}
		newTotal, err = tx.AddExperience(ctx, userID, actualDelta)
		if err != nil {
			return err
		}
		configs := s.levelConfigs(ctx)
		current, _ := resolveLevel(configs, newTotal)
		if current == nil {
			return ErrLevelConfigNotFound
		}
		newLevelSeen = true
		if current.Level != experience.CurrentLevel {
			if err := tx.UpdateCurrentLevel(ctx, userID, current.Level); err != nil {
				return err
			}
			oldLevel, newLevel = experience.CurrentLevel, current.Level
			leveledUp = newLevel > oldLevel
		} else {
			newLevel = current.Level
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if actualDelta == 0 {
		result.Reason = DenyReasonExperienceFloor
		return s.denyWithProfile(ctx, userID, result)
	}

	result.Granted = true
	result.ExpGained = actualDelta
	result.TotalExp = newTotal
	result.LeveledUp = leveledUp
	result.OldLevel = oldLevel
	result.NewLevel = newLevel
	if newLevelSeen {
		result.Level = newLevel
		if configs := s.levelConfigs(ctx); configs != nil {
			if current, _ := resolveLevel(configs, newTotal); current != nil {
				result.LevelName = current.Name
			}
		}
	}
	if leveledUp {
		s.notifyLevelUp(ctx, userID, oldLevel, newLevel)
	}
	return result, nil
}

// GetProfile returns the user's level profile; users without an experience
// row still report LV1 with zero exp.
func (s *ExperienceService) GetProfile(ctx context.Context, userID uint) (*leveldto.UserLevelData, error) {
	configs := s.levelConfigs(ctx)
	if len(configs) == 0 {
		return nil, ErrLevelConfigNotFound
	}
	experience, err := s.repository.FindExperience(ctx, userID)
	if err != nil {
		return nil, err
	}
	totalExp := int64(0)
	if experience != nil {
		totalExp = experience.TotalExp
	}
	current, next := resolveLevel(configs, totalExp)
	if current == nil {
		return nil, ErrLevelConfigNotFound
	}
	progress, remaining := levelProgress(current, next, totalExp)
	data := &leveldto.UserLevelData{
		Level:           current.Level,
		LevelName:       current.Name,
		TotalExp:        totalExp,
		CurrentLevelExp: current.MinExp,
		IconURL:         current.IconURL,
		Color:           current.Color,
		RemainingExp:    remaining,
		Progress:        progress,
		IsMaxLevel:      next == nil,
	}
	if next != nil {
		nextLevel, nextLevelExp, nextLevelName := next.Level, next.MinExp, next.Name
		data.NextLevel = &nextLevel
		data.NextLevelExp = &nextLevelExp
		data.NextLevelName = &nextLevelName
	}
	return data, nil
}

// GetLogs returns one page of the user's experience history.
func (s *ExperienceService) GetLogs(
	ctx context.Context,
	userID uint,
	page, limit int,
) ([]levelModel.ExperienceLog, int64, int, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	logs, total, err := s.repository.ListExperienceLogs(ctx, userID, page, limit)
	return logs, total, page, limit, err
}

// Summaries returns level badge data for the given users, used by modules that
// display user cards (profiles, comments, posts, contributor lists).
func (s *ExperienceService) Summaries(
	ctx context.Context,
	userIDs []uint,
) (map[uint]leveldto.UserLevelSummary, error) {
	unique := make([]uint, 0, len(userIDs))
	seen := make(map[uint]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	summaries := make(map[uint]leveldto.UserLevelSummary, len(unique))
	if len(unique) == 0 {
		return summaries, nil
	}
	rows, err := s.repository.ListLevelSummaries(ctx, unique)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		summaries[row.UserID] = leveldto.UserLevelSummary{
			Level: row.Level, Name: row.Name, IconURL: row.IconURL, Color: row.Color,
		}
	}
	return summaries, nil
}

// FindUserIDByUsername resolves a username to a user id for public level lookups.
func (s *ExperienceService) FindUserIDByUsername(ctx context.Context, username string) (uint, bool, error) {
	return s.repository.FindUserIDByUsername(ctx, username)
}

func (s *ExperienceService) denyWithProfile(
	ctx context.Context,
	userID uint,
	result *GrantResult,
) (*GrantResult, error) {
	profile, err := s.GetProfile(ctx, userID)
	if err == nil && profile != nil {
		result.TotalExp = profile.TotalExp
		result.Level = profile.Level
		result.LevelName = profile.LevelName
	}
	return result, nil
}

func (s *ExperienceService) notifyLevelUp(ctx context.Context, userID uint, oldLevel, newLevel int) {
	if s.notifications == nil {
		return
	}
	configs := s.levelConfigs(ctx)
	oldName, newName := "", ""
	for i := range configs {
		switch configs[i].Level {
		case oldLevel:
			oldName = configs[i].Name
		case newLevel:
			newName = configs[i].Name
		}
	}
	_, err := s.notifications.Create(ctx, notificationService.CreateInput{
		RecipientID: userID,
		Category:    notificationModel.CategorySystem,
		Type:        notificationModel.TypeLevelUp,
		EntityType:  "user",
		EntityID:    userID,
		Title:       "等级提升",
		Content:     fmt.Sprintf("恭喜你升级了！LV%d %s → LV%d %s", oldLevel, oldName, newLevel, newName),
		TargetURL:   "/settings/experience",
		Metadata:    map[string]any{"old_level": oldLevel, "new_level": newLevel},
	})
	if err != nil {
		logger.Error("create level up notification",
			zap.Uint("user_id", userID), zap.Int("new_level", newLevel), zap.Error(err))
	}
}

// ------------------------------------------------------------------- caching

func (s *ExperienceService) levelConfigs(ctx context.Context) []levelModel.LevelConfig {
	if s.cache != nil {
		if encoded, err := s.cache.Get(ctx, CacheKeyLevelConfigs).Result(); err == nil {
			var configs []levelModel.LevelConfig
			if json.Unmarshal([]byte(encoded), &configs) == nil {
				return configs
			}
		}
	}
	configs, err := s.repository.ListLevelConfigs(ctx, true)
	if err != nil {
		logger.Error("load level configs", zap.Error(err))
		return nil
	}
	if s.cache != nil && configs != nil {
		if encoded, err := json.Marshal(configs); err == nil {
			s.cache.Set(context.WithoutCancel(ctx), CacheKeyLevelConfigs, string(encoded), configCacheTTL)
		}
	}
	return configs
}

func (s *ExperienceService) rule(
	ctx context.Context,
	eventType levelModel.EventType,
) (*levelModel.ExperienceRule, error) {
	rules, err := s.rules(ctx)
	if err != nil {
		return nil, err
	}
	rule, ok := rules[eventType]
	if !ok {
		return nil, nil
	}
	return &rule, nil
}

func (s *ExperienceService) rules(
	ctx context.Context,
) (map[levelModel.EventType]levelModel.ExperienceRule, error) {
	if s.cache != nil {
		if encoded, err := s.cache.Get(ctx, CacheKeyExperienceRules).Result(); err == nil {
			var rules map[levelModel.EventType]levelModel.ExperienceRule
			if json.Unmarshal([]byte(encoded), &rules) == nil {
				return rules, nil
			}
		}
	}
	list, err := s.repository.ListExperienceRules(ctx)
	if err != nil {
		return nil, err
	}
	rules := make(map[levelModel.EventType]levelModel.ExperienceRule, len(list))
	for i := range list {
		rules[list[i].EventType] = list[i]
	}
	if s.cache != nil {
		if encoded, err := json.Marshal(rules); err == nil {
			s.cache.Set(context.WithoutCancel(ctx), CacheKeyExperienceRules, string(encoded), configCacheTTL)
		}
	}
	return rules, nil
}

// InvalidateCaches drops cached level configs and experience rules; called
// after any admin configuration change.
func InvalidateCaches(ctx context.Context, cache *redis.Client) {
	if cache == nil {
		return
	}
	cache.Del(context.WithoutCancel(ctx), CacheKeyLevelConfigs, CacheKeyExperienceRules)
}

// idempotencyKey builds the dedup key for idempotent events, e.g.
// "resource_approved:resource:12" or "comment_liked:comment:42:7".
func idempotencyKey(
	eventType levelModel.EventType,
	sourceType string,
	sourceID uint,
	actorID *uint,
) string {
	if !levelModel.IsIdempotentEvent(eventType) || sourceID == 0 || sourceType == "" {
		return ""
	}
	key := fmt.Sprintf("%s:%s:%d", eventType, sourceType, sourceID)
	if actorID != nil {
		key = fmt.Sprintf("%s:%d", key, *actorID)
	}
	return key
}

func idempotencyKeyOrNil(key string) *string {
	if key == "" {
		return nil
	}
	return &key
}
