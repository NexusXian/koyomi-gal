package service

import (
	"context"

	"backend/internal/galgame/repository"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

// UserRelationService reads the current user's rating, favorite, and play
// state relations for a single galgame.
type UserRelationService struct {
	galgames  *repository.GalgameRepository
	relations *repository.UserRelationRepository
}

func NewUserRelationService(
	galgames *repository.GalgameRepository,
	relations *repository.UserRelationRepository,
) *UserRelationService {
	return &UserRelationService{galgames: galgames, relations: relations}
}

func (s *UserRelationService) GetGalgameRelation(
	ctx context.Context,
	galgameID, userID uint,
) (*repository.UserRelationSummary, error) {
	if err := ensurePublishedGalgame(ctx, s.galgames, galgameID); err != nil {
		return nil, err
	}
	summary, err := s.relations.FindUserRelation(ctx, galgameID, userID)
	if err != nil {
		logger.Error("find galgame user relation",
			zap.Uint("galgame_id", galgameID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	return summary, nil
}

func ensurePublishedGalgame(ctx context.Context, galgames *repository.GalgameRepository, id uint) error {
	galgame, err := galgames.FindPublishedByID(ctx, id)
	if err != nil {
		logger.Error("find galgame by id", zap.Uint("galgame_id", id), zap.Error(err))
		return err
	}
	if galgame == nil {
		return ErrGalgameNotFound
	}
	return nil
}
