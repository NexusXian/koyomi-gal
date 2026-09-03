package service

import (
	"context"

	"backend/internal/user/repository"
)

type ActivityRecorder interface {
	Record(ctx context.Context, userID uint, activityType string, entityID *uint, metadata map[string]any) error
}

type UserActivityService struct {
	profiles *repository.UserProfileRepository
}

func NewUserActivityService(profiles *repository.UserProfileRepository) *UserActivityService {
	return &UserActivityService{profiles: profiles}
}

func (s *UserActivityService) Record(ctx context.Context, userID uint, activityType string, entityID *uint, metadata map[string]any) error {
	activity, err := repository.NewActivity(userID, activityType, entityID, metadata)
	if err != nil {
		return err
	}
	return s.profiles.RecordActivity(ctx, activity)
}
