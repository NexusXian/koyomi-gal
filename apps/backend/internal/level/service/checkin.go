package service

import (
	"context"
	"errors"
	"time"

	levelModel "backend/internal/level/model"
	"backend/internal/level/repository"
)

var ErrAlreadyCheckedIn = errors.New("already checked in today")

// CheckinService records the daily checkin and reports the daily_checkin
// experience event. The unique (user_id, checkin_date) index makes repeat
// checkins impossible even under concurrency.
type CheckinService struct {
	repository   *repository.Repository
	experiences  *ExperienceService
}

func NewCheckinService(
	repository *repository.Repository,
	experiences *ExperienceService,
) *CheckinService {
	return &CheckinService{repository: repository, experiences: experiences}
}

// Checkin performs today's checkin and grants the daily_checkin experience.
// The experience grant never fails the checkin itself.
func (s *CheckinService) Checkin(ctx context.Context, userID uint) (*CheckinResult, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if existing, err := s.repository.FindCheckin(ctx, userID, today); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, ErrAlreadyCheckedIn
	}

	consecutiveDays := 1
	yesterday, err := s.repository.FindCheckin(ctx, userID, today.AddDate(0, 0, -1))
	if err != nil {
		return nil, err
	}
	if yesterday != nil {
		consecutiveDays = yesterday.ConsecutiveDays + 1
	}

	inserted, err := s.repository.CreateCheckin(ctx, &levelModel.UserCheckin{
		UserID:          userID,
		CheckinDate:     today,
		ConsecutiveDays: consecutiveDays,
	})
	if err != nil {
		return nil, err
	}
	if !inserted {
		return nil, ErrAlreadyCheckedIn
	}

	result, err := s.experiences.Grant(
		ctx, userID, levelModel.EventDailyCheckin, levelModel.SourceCheckin, userID,
		WithDescription("每日签到"),
	)
	if err != nil {
		// The checkin itself already succeeded; keep exp at zero on failure.
		result = &GrantResult{}
	}
	return &CheckinResult{
		ExpGained:       result.ExpGained,
		ConsecutiveDays: consecutiveDays,
		TotalExp:        result.TotalExp,
		Level:           result.Level,
		LevelName:       result.LevelName,
	}, nil
}

// Status reports whether the user checked in today and the visible streak:
// today's streak when checked in, otherwise yesterday's streak if any.
func (s *CheckinService) Status(ctx context.Context, userID uint) (checkedInToday bool, consecutiveDays int, err error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	checkin, err := s.repository.FindCheckin(ctx, userID, today)
	if err != nil {
		return false, 0, err
	}
	if checkin != nil {
		return true, checkin.ConsecutiveDays, nil
	}
	yesterday, err := s.repository.FindCheckin(ctx, userID, today.AddDate(0, 0, -1))
	if err != nil {
		return false, 0, err
	}
	if yesterday != nil {
		return false, yesterday.ConsecutiveDays, nil
	}
	return false, 0, nil
}

type CheckinResult struct {
	ExpGained       int64
	ConsecutiveDays int
	TotalExp        int64
	Level           int
	LevelName       string
}
