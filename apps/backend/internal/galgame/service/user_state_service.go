package service

import (
	"context"
	"errors"

	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

var (
	ErrInvalidUserState  = errors.New("invalid galgame play state")
	ErrInvalidPlayTime   = errors.New("play time minutes must not be negative")
	ErrUserStateNotFound = errors.New("galgame user state not found")
)

type UserStateService struct {
	galgames  *repository.GalgameRepository
	relations *repository.UserRelationRepository
}

func NewUserStateService(
	galgames *repository.GalgameRepository,
	relations *repository.UserRelationRepository,
) *UserStateService {
	return &UserStateService{galgames: galgames, relations: relations}
}

// UpsertState creates or replaces the user's play state for a galgame.
func (s *UserStateService) UpsertState(
	ctx context.Context,
	galgameID, userID uint,
	state int16,
	playTimeMinutes int64,
) (*model.UserState, error) {
	if !validUserState(state) {
		return nil, ErrInvalidUserState
	}
	if playTimeMinutes < 0 {
		return nil, ErrInvalidPlayTime
	}
	if err := ensurePublishedGalgame(ctx, s.galgames, galgameID); err != nil {
		return nil, err
	}

	err := s.relations.UpsertUserState(ctx, galgameID, userID, state, playTimeMinutes)
	if err != nil {
		logger.Error("upsert galgame user state",
			zap.Uint("galgame_id", galgameID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	userState, err := s.relations.FindUserState(ctx, galgameID, userID)
	if err != nil {
		logger.Error("find galgame user state",
			zap.Uint("galgame_id", galgameID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	return userState, nil
}

func (s *UserStateService) DeleteState(ctx context.Context, galgameID, userID uint) error {
	if err := ensurePublishedGalgame(ctx, s.galgames, galgameID); err != nil {
		return err
	}
	removed, err := s.relations.DeleteUserState(ctx, galgameID, userID)
	if err != nil {
		logger.Error("delete galgame user state",
			zap.Uint("galgame_id", galgameID), zap.Uint("user_id", userID), zap.Error(err))
		return err
	}
	if !removed {
		return ErrUserStateNotFound
	}
	return nil
}

func validUserState(value int16) bool {
	return value >= model.UserStateWish && value <= model.UserStateDropped
}
