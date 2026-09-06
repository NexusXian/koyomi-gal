package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	levelModel "backend/internal/level/model"
	levelRepo "backend/internal/level/repository"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

func newExperienceService(t *testing.T, db *gorm.DB) *ExperienceService {
	t.Helper()
	return NewExperienceService(levelRepo.NewRepository(db), nil, nil)
}

func seedLevelConfigs(t *testing.T, db *gorm.DB) {
	t.Helper()
	configs := levelConfigs()
	for i := range configs {
		config := configs[i]
		config.IsEnabled = true
		if err := db.Create(&config).Error; err != nil {
			t.Fatalf("seed level config %d: %v", config.Level, err)
		}
	}
}

func seedRule(t *testing.T, db *gorm.DB, rule *levelModel.ExperienceRule) {
	t.Helper()
	if err := db.Create(rule).Error; err != nil {
		t.Fatalf("seed rule %s: %v", rule.EventType, err)
	}
}

func totalExp(t *testing.T, db *gorm.DB, userID uint) int64 {
	t.Helper()
	var experience levelModel.UserExperience
	if err := db.First(&experience, userID).Error; err != nil {
		t.Fatalf("load user experience: %v", err)
	}
	return experience.TotalExp
}

func currentLevel(t *testing.T, db *gorm.DB, userID uint) int {
	t.Helper()
	var experience levelModel.UserExperience
	if err := db.First(&experience, userID).Error; err != nil {
		t.Fatalf("load user experience: %v", err)
	}
	return experience.CurrentLevel
}

func countLogs(t *testing.T, db *gorm.DB, userID uint, eventType levelModel.EventType) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&levelModel.ExperienceLog{}).
		Where("user_id = ? AND event_type = ?", userID, eventType).
		Count(&count).Error; err != nil {
		t.Fatalf("count logs: %v", err)
	}
	return count
}

func TestGrantFirstExperience(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventLikeGiven, Name: "点赞内容", Exp: 1, DailyLimit: 5, Enabled: true,
	})
	userID := testutil.CreateUser(t, db, "exp-user")
	service := newExperienceService(t, db)

	result, err := service.Grant(context.Background(), userID, levelModel.EventLikeGiven, levelModel.SourcePost, 1)
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if !result.Granted || result.ExpGained != 1 {
		t.Fatalf("expected granted 1 exp, got %+v", result)
	}
	if got := totalExp(t, db, userID); got != 1 {
		t.Fatalf("expected total 1, got %d", got)
	}
	if got := countLogs(t, db, userID, levelModel.EventLikeGiven); got != 1 {
		t.Fatalf("expected 1 log, got %d", got)
	}
	if result.Level != 1 {
		t.Fatalf("expected level 1, got %d", result.Level)
	}
}

func TestGrantDailyLimit(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventLikeGiven, Name: "点赞内容", Exp: 1, DailyLimit: 2, Enabled: true,
	})
	userID := testutil.CreateUser(t, db, "limit-user")
	service := newExperienceService(t, db)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		result, err := service.Grant(ctx, userID, levelModel.EventLikeGiven, levelModel.SourcePost, uint(i+1))
		if err != nil || !result.Granted {
			t.Fatalf("grant %d should succeed: %v %+v", i+1, err, result)
		}
	}
	denied, err := service.Grant(ctx, userID, levelModel.EventLikeGiven, levelModel.SourcePost, 99)
	if err != nil {
		t.Fatalf("grant over limit: %v", err)
	}
	if denied.Granted || denied.Reason != DenyReasonDailyLimit {
		t.Fatalf("expected daily limit deny, got %+v", denied)
	}
	if got := totalExp(t, db, userID); got != 2 {
		t.Fatalf("expected total 2, got %d", got)
	}
	if got := countLogs(t, db, userID, levelModel.EventLikeGiven); got != 2 {
		t.Fatalf("expected 2 logs, got %d", got)
	}
}

func TestGrantDailyExpLimitPartialCredit(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventCommentLiked, Name: "评论被点赞", Exp: 3, DailyExpLimit: 7, Enabled: true,
	})
	authorID := testutil.CreateUser(t, db, "liked-author")
	service := newExperienceService(t, db)
	ctx := context.Background()

	// Three grants award 3 + 3 + partial 1; the fourth is denied.
	gains := []int64{3, 3, 1}
	for i, expected := range gains {
		result, err := service.Grant(ctx, authorID, levelModel.EventCommentLiked, levelModel.SourceComment, uint(i+1))
		if err != nil || !result.Granted {
			t.Fatalf("grant %d should succeed: %v %+v", i+1, err, result)
		}
		if result.ExpGained != expected {
			t.Fatalf("grant %d expected %d exp, got %d", i+1, expected, result.ExpGained)
		}
	}
	denied, err := service.Grant(ctx, authorID, levelModel.EventCommentLiked, levelModel.SourceComment, 99)
	if err != nil {
		t.Fatalf("grant over cap: %v", err)
	}
	if denied.Granted || denied.Reason != DenyReasonDailyExpLimit {
		t.Fatalf("expected daily exp cap deny, got %+v", denied)
	}
	if got := totalExp(t, db, authorID); got != 7 {
		t.Fatalf("expected total 7, got %d", got)
	}
}

func TestGrantDisabledRule(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventShare, Name: "分享内容", Exp: 3, Enabled: false,
	})
	userID := testutil.CreateUser(t, db, "share-user")
	service := newExperienceService(t, db)

	result, err := service.Grant(context.Background(), userID, levelModel.EventShare, levelModel.SourcePost, 1)
	if err != nil {
		t.Fatalf("grant disabled rule: %v", err)
	}
	if result.Granted || result.Reason != DenyReasonRuleDisabled {
		t.Fatalf("expected rule disabled deny, got %+v", result)
	}
	if got := countLogs(t, db, userID, levelModel.EventShare); got != 0 {
		t.Fatalf("expected no logs, got %d", got)
	}
}

func TestGrantUnknownRule(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	userID := testutil.CreateUser(t, db, "norule-user")
	service := newExperienceService(t, db)

	result, err := service.Grant(context.Background(), userID, levelModel.EventShare, levelModel.SourcePost, 1)
	if err != nil {
		t.Fatalf("grant unknown rule: %v", err)
	}
	if result.Granted || result.Reason != DenyReasonRuleMissing {
		t.Fatalf("expected rule missing deny, got %+v", result)
	}
}

func TestGrantIdempotentDuplicate(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventResourceApproved, Name: "资源贡献审核通过", Exp: 50, Enabled: true,
	})
	userID := testutil.CreateUser(t, db, "resource-user")
	service := newExperienceService(t, db)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		result, err := service.Grant(ctx, userID, levelModel.EventResourceApproved, levelModel.SourceResource, 123)
		if err != nil {
			t.Fatalf("grant %d: %v", i+1, err)
		}
		if i == 0 && !result.Granted {
			t.Fatalf("first grant should succeed: %+v", result)
		}
		if i == 1 && (result.Granted || result.Reason != DenyReasonDuplicate) {
			t.Fatalf("second grant should be duplicate denied: %+v", result)
		}
	}
	if got := totalExp(t, db, userID); got != 50 {
		t.Fatalf("expected total 50, got %d", got)
	}
	if got := countLogs(t, db, userID, levelModel.EventResourceApproved); got != 1 {
		t.Fatalf("expected 1 log, got %d", got)
	}
}

func TestGrantCommentLikedPerActorIdempotency(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventCommentLiked, Name: "评论被点赞", Exp: 2, Enabled: true,
	})
	authorID := testutil.CreateUser(t, db, "comment-author")
	likerA := testutil.CreateUser(t, db, "liker-a")
	likerB := testutil.CreateUser(t, db, "liker-b")
	service := newExperienceService(t, db)
	ctx := context.Background()

	// Same liker twice rewards once; a different liker rewards again.
	if _, err := service.Grant(ctx, authorID, levelModel.EventCommentLiked, levelModel.SourceComment, 42, WithActor(likerA)); err != nil {
		t.Fatalf("grant liker A: %v", err)
	}
	duplicate, err := service.Grant(ctx, authorID, levelModel.EventCommentLiked, levelModel.SourceComment, 42, WithActor(likerA))
	if err != nil {
		t.Fatalf("grant liker A again: %v", err)
	}
	if duplicate.Granted || duplicate.Reason != DenyReasonDuplicate {
		t.Fatalf("expected duplicate deny for same liker, got %+v", duplicate)
	}
	other, err := service.Grant(ctx, authorID, levelModel.EventCommentLiked, levelModel.SourceComment, 42, WithActor(likerB))
	if err != nil || !other.Granted {
		t.Fatalf("different liker should reward: %v %+v", err, other)
	}
	if got := totalExp(t, db, authorID); got != 4 {
		t.Fatalf("expected total 4, got %d", got)
	}
}

func TestGrantCooldown(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventShare, Name: "分享内容", Exp: 3, CooldownSeconds: 3600, Enabled: true,
	})
	userID := testutil.CreateUser(t, db, "cooldown-user")
	service := newExperienceService(t, db)
	ctx := context.Background()

	if result, err := service.Grant(ctx, userID, levelModel.EventShare, levelModel.SourcePost, 1); err != nil || !result.Granted {
		t.Fatalf("first share should reward: %v %+v", err, result)
	}
	denied, err := service.Grant(ctx, userID, levelModel.EventShare, levelModel.SourcePost, 2)
	if err != nil {
		t.Fatalf("cooldown grant: %v", err)
	}
	if denied.Granted || denied.Reason != DenyReasonCooldown {
		t.Fatalf("expected cooldown deny, got %+v", denied)
	}
}

func TestGrantLevelUp(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventResourceApproved, Name: "资源贡献审核通过", Exp: 60, Enabled: true,
	})
	userID := testutil.CreateUser(t, db, "levelup-user")
	service := newExperienceService(t, db)
	ctx := context.Background()

	// 60 exp stays LV1; +60 more crosses 100 into LV2.
	first, err := service.Grant(ctx, userID, levelModel.EventResourceApproved, levelModel.SourceResource, 1)
	if err != nil || first.LeveledUp {
		t.Fatalf("first grant should not level up: %v %+v", err, first)
	}
	second, err := service.Grant(ctx, userID, levelModel.EventResourceApproved, levelModel.SourceResource, 2)
	if err != nil {
		t.Fatalf("second grant: %v", err)
	}
	if !second.LeveledUp || second.OldLevel != 1 || second.NewLevel != 2 {
		t.Fatalf("expected LV1 -> LV2, got %+v", second)
	}
	if got := currentLevel(t, db, userID); got != 2 {
		t.Fatalf("expected cached level 2, got %d", got)
	}
}

func TestGrantMultiLevelUp(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventAdminAdjustment, Name: "管理员调整", Exp: 0, Enabled: true,
	})
	userID := testutil.CreateUser(t, db, "multilevel-user")
	adminID := testutil.CreateUser(t, db, "multilevel-admin")
	service := newExperienceService(t, db)

	result, err := service.Adjust(context.Background(), userID, 600, "batch reward", adminID)
	if err != nil {
		t.Fatalf("adjust: %v", err)
	}
	if !result.LeveledUp || result.OldLevel != 1 || result.NewLevel != 3 {
		t.Fatalf("expected LV1 -> LV3, got %+v", result)
	}
	if got := currentLevel(t, db, userID); got != 3 {
		t.Fatalf("expected cached level 3, got %d", got)
	}
}

func TestGrantMaxLevel(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	userID := testutil.CreateUser(t, db, "maxlevel-user")
	adminID := testutil.CreateUser(t, db, "maxlevel-admin")
	service := newExperienceService(t, db)

	if _, err := service.Adjust(context.Background(), userID, 5000, "max out", adminID); err != nil {
		t.Fatalf("adjust: %v", err)
	}
	profile, err := service.GetProfile(context.Background(), userID)
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if !profile.IsMaxLevel || profile.NextLevel != nil || profile.Progress != 1 || profile.RemainingExp != 0 {
		t.Fatalf("expected max level profile, got %+v", profile)
	}
}

func TestAdjustDeductAndDowngrade(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	userID := testutil.CreateUser(t, db, "downgrade-user")
	adminID := testutil.CreateUser(t, db, "downgrade-admin")
	service := newExperienceService(t, db)
	ctx := context.Background()

	if _, err := service.Adjust(ctx, userID, 600, "grant", adminID); err != nil {
		t.Fatalf("adjust up: %v", err)
	}
	result, err := service.Adjust(ctx, userID, -550, "penalty", adminID)
	if err != nil {
		t.Fatalf("adjust down: %v", err)
	}
	if !result.Granted || result.ExpGained != -550 {
		t.Fatalf("expected -550 delta, got %+v", result)
	}
	if got := currentLevel(t, db, userID); got != 1 {
		t.Fatalf("expected level downgraded to 1, got %d", got)
	}
	if got := totalExp(t, db, userID); got != 50 {
		t.Fatalf("expected total 50, got %d", got)
	}
	if got := countLogs(t, db, userID, levelModel.EventAdminAdjustment); got != 2 {
		t.Fatalf("expected 2 adjustment logs, got %d", got)
	}
}

func TestAdjustClampsAtZero(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	userID := testutil.CreateUser(t, db, "clamp-user")
	adminID := testutil.CreateUser(t, db, "clamp-admin")
	service := newExperienceService(t, db)
	ctx := context.Background()

	if _, err := service.Adjust(ctx, userID, 30, "grant", adminID); err != nil {
		t.Fatalf("adjust up: %v", err)
	}
	result, err := service.Adjust(ctx, userID, -100, "over-deduct", adminID)
	if err != nil {
		t.Fatalf("adjust down: %v", err)
	}
	if !result.Granted || result.ExpGained != -30 {
		t.Fatalf("expected clamped -30 delta, got %+v", result)
	}
	if got := totalExp(t, db, userID); got != 0 {
		t.Fatalf("expected total 0, got %d", got)
	}

	// Deducting at zero is a no-op without a log.
	before := countLogs(t, db, userID, levelModel.EventAdminAdjustment)
	floor, err := service.Adjust(ctx, userID, -10, "at floor", adminID)
	if err != nil {
		t.Fatalf("adjust at floor: %v", err)
	}
	if floor.Granted || floor.Reason != DenyReasonExperienceFloor {
		t.Fatalf("expected floor deny, got %+v", floor)
	}
	if after := countLogs(t, db, userID, levelModel.EventAdminAdjustment); after != before {
		t.Fatalf("expected no extra log at floor")
	}
}

func TestAdjustValidation(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	userID := testutil.CreateUser(t, db, "adjust-user")
	adminID := testutil.CreateUser(t, db, "adjust-admin")
	service := newExperienceService(t, db)

	if _, err := service.Adjust(context.Background(), userID, 0, "zero", adminID); err != ErrInvalidAdjustment {
		t.Fatalf("expected invalid adjustment for zero exp, got %v", err)
	}
	if _, err := service.Adjust(context.Background(), userID, 10, "  ", adminID); err != ErrInvalidAdjustment {
		t.Fatalf("expected invalid adjustment for empty reason, got %v", err)
	}
	if _, err := service.Adjust(context.Background(), 999999, 10, "missing user", adminID); err != ErrUserNotFound {
		t.Fatalf("expected user not found, got %v", err)
	}
}

func TestGetProfileDefaultLevel(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	userID := testutil.CreateUser(t, db, "profile-user")
	service := newExperienceService(t, db)

	profile, err := service.GetProfile(context.Background(), userID)
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if profile.Level != 1 || profile.TotalExp != 0 || profile.LevelName != "初见" {
		t.Fatalf("expected LV1 with zero exp, got %+v", profile)
	}
	if profile.NextLevel == nil || *profile.NextLevel != 2 {
		t.Fatalf("expected next level 2, got %+v", profile.NextLevel)
	}
}

func TestGetLogsPagination(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventLikeGiven, Name: "点赞内容", Exp: 1, DailyLimit: 100, Enabled: true,
	})
	userID := testutil.CreateUser(t, db, "logs-user")
	service := newExperienceService(t, db)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := service.Grant(ctx, userID, levelModel.EventLikeGiven, levelModel.SourcePost, uint(i+1)); err != nil {
			t.Fatalf("grant %d: %v", i+1, err)
		}
	}
	logs, total, page, limit, err := service.GetLogs(ctx, userID, 1, 2)
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	if total != 3 || len(logs) != 2 || page != 1 || limit != 2 {
		t.Fatalf("unexpected page result: total=%d len=%d page=%d limit=%d", total, len(logs), page, limit)
	}
}

func TestSummaries(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	userA := testutil.CreateUser(t, db, "summary-a")
	userB := testutil.CreateUser(t, db, "summary-b")
	adminID := testutil.CreateUser(t, db, "summary-admin")
	service := newExperienceService(t, db)

	if _, err := service.Adjust(context.Background(), userA, 200, "grant", adminID); err != nil {
		t.Fatalf("adjust: %v", err)
	}
	summaries, err := service.Summaries(context.Background(), []uint{userA, userB})
	if err != nil {
		t.Fatalf("summaries: %v", err)
	}
	if summary, ok := summaries[userA]; !ok || summary.Level != 2 || summary.Name != "读者" {
		t.Fatalf("expected LV2 读者 for user A, got %+v ok=%v", summary, ok)
	}
	if summary, ok := summaries[userB]; !ok || summary.Level != 1 || summary.Name != "初见" {
		t.Fatalf("expected default LV1 初见 for user B, got %+v ok=%v", summary, ok)
	}
}

func TestConcurrentGrantSameEvent(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventResourceApproved, Name: "资源贡献审核通过", Exp: 50, Enabled: true,
	})
	userID := testutil.CreateUser(t, db, "concurrent-user")
	service := newExperienceService(t, db)

	const workers = 8
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = service.Grant(context.Background(), userID, levelModel.EventResourceApproved, levelModel.SourceResource, 7)
		}()
	}
	wg.Wait()
	if got := totalExp(t, db, userID); got != 50 {
		t.Fatalf("expected exactly one 50 exp grant, got %d", got)
	}
	if got := countLogs(t, db, userID, levelModel.EventResourceApproved); got != 1 {
		t.Fatalf("expected 1 log, got %d", got)
	}
}

func TestConcurrentGrantDailyLimit(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventLikeGiven, Name: "点赞内容", Exp: 1, DailyLimit: 5, Enabled: true,
	})
	userID := testutil.CreateUser(t, db, "concurrent-limit-user")
	service := newExperienceService(t, db)

	const workers = 20
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, _ = service.Grant(context.Background(), userID, levelModel.EventLikeGiven, levelModel.SourcePost, uint(index+1))
		}(i)
	}
	wg.Wait()
	if got := totalExp(t, db, userID); got != 5 {
		t.Fatalf("expected total capped at 5, got %d", got)
	}
	if got := countLogs(t, db, userID, levelModel.EventLikeGiven); got != 5 {
		t.Fatalf("expected 5 logs, got %d", got)
	}
}

func TestCheckinFlow(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventDailyCheckin, Name: "每日签到", Exp: 10, DailyLimit: 1, Enabled: true,
	})
	repository := levelRepo.NewRepository(db)
	experiences := NewExperienceService(repository, nil, nil)
	checkins := NewCheckinService(repository, experiences)
	userID := testutil.CreateUser(t, db, "checkin-user")
	ctx := context.Background()

	result, err := checkins.Checkin(ctx, userID)
	if err != nil {
		t.Fatalf("checkin: %v", err)
	}
	if result.ExpGained != 10 || result.ConsecutiveDays != 1 {
		t.Fatalf("expected 10 exp on day 1, got %+v", result)
	}
	if _, err := checkins.Checkin(ctx, userID); err != ErrAlreadyCheckedIn {
		t.Fatalf("expected already checked in, got %v", err)
	}
	if got := totalExp(t, db, userID); got != 10 {
		t.Fatalf("expected total 10, got %d", got)
	}

	// Simulate yesterday's checkin; a new day checkin continues the streak.
	now := time.Now()
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
	if err := db.Create(&levelModel.UserCheckin{
		UserID: userID, CheckinDate: yesterday, ConsecutiveDays: 6,
	}).Error; err != nil {
		t.Fatalf("seed old checkin: %v", err)
	}
	// Remove today's checkin to check in again (unique per date).
	if err := db.Where("user_id = ? AND checkin_date = ?", userID, yesterday.AddDate(0, 0, 1).Format("2006-01-02")).
		Delete(&levelModel.UserCheckin{}).Error; err != nil {
		t.Fatalf("remove today checkin: %v", err)
	}
	if err := db.Exec(
		"UPDATE experience_logs SET created_at = created_at - INTERVAL '1 day' WHERE user_id = ?",
		userID,
	).Error; err != nil {
		t.Fatalf("age checkin logs: %v", err)
	}
	second, err := checkins.Checkin(ctx, userID)
	if err != nil {
		t.Fatalf("second-day checkin: %v", err)
	}
	if second.ConsecutiveDays != 7 {
		t.Fatalf("expected streak 7, got %d", second.ConsecutiveDays)
	}
	if second.TotalExp != 20 {
		t.Fatalf("expected total 20, got %d", second.TotalExp)
	}
}

func TestCheckinStatus(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repository := levelRepo.NewRepository(db)
	checkins := NewCheckinService(repository, NewExperienceService(repository, nil, nil))
	userID := testutil.CreateUser(t, db, "status-user")

	checkedIn, consecutive, err := checkins.Status(context.Background(), userID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if checkedIn || consecutive != 0 {
		t.Fatalf("expected empty status, got %v %d", checkedIn, consecutive)
	}
}

func TestGrantWritesIdempotencyKey(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	seedLevelConfigs(t, db)
	seedRule(t, db, &levelModel.ExperienceRule{
		EventType: levelModel.EventResourceApproved, Name: "资源贡献审核通过", Exp: 50, Enabled: true,
	})
	userID := testutil.CreateUser(t, db, "key-user")
	service := newExperienceService(t, db)

	if _, err := service.Grant(context.Background(), userID, levelModel.EventResourceApproved, levelModel.SourceResource, 33); err != nil {
		t.Fatalf("grant: %v", err)
	}
	var log levelModel.ExperienceLog
	if err := db.Where("user_id = ?", userID).First(&log).Error; err != nil {
		t.Fatalf("load log: %v", err)
	}
	expected := fmt.Sprintf("%s:%s:%d", levelModel.EventResourceApproved, levelModel.SourceResource, 33)
	if log.IdempotencyKey == nil || *log.IdempotencyKey != expected {
		t.Fatalf("expected idempotency key %s, got %v", expected, log.IdempotencyKey)
	}
}
