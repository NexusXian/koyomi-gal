package service

import (
	"context"
	"errors"

	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	userModel "backend/internal/user/model"
	userService "backend/internal/user/service"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

var (
	ErrAlreadyFavorited = errors.New("galgame already favorited")
	ErrFavoriteNotFound = errors.New("galgame favorite not found")
)

type FavoriteService struct {
	galgames   *repository.GalgameRepository
	relations  *repository.UserRelationRepository
	activities userService.ActivityRecorder
}

func (s *FavoriteService) SetActivityRecorder(recorder userService.ActivityRecorder) {
	s.activities = recorder
}

func NewFavoriteService(
	galgames *repository.GalgameRepository,
	relations *repository.UserRelationRepository,
) *FavoriteService {
	return &FavoriteService{galgames: galgames, relations: relations}
}

// AddFavorite creates the favorite and atomically increments favorite_count
// in one transaction; a duplicate request aborts with ErrAlreadyFavorited.
func (s *FavoriteService) AddFavorite(ctx context.Context, galgameID, userID uint) (*model.Favorite, error) {
	if err := ensurePublishedGalgame(ctx, s.galgames, galgameID); err != nil {
		return nil, err
	}

	var favorite *model.Favorite
	err := s.relations.Transaction(ctx, func(tx *repository.UserRelationRepository) error {
		inserted, err := tx.AddFavorite(ctx, galgameID, userID)
		if err != nil {
			return err
		}
		if !inserted {
			return ErrAlreadyFavorited
		}
		if err := tx.IncrementFavoriteCount(ctx, galgameID); err != nil {
			return err
		}
		favorite, err = tx.FindFavorite(ctx, galgameID, userID)
		return err
	})
	if err != nil {
		if errors.Is(err, ErrAlreadyFavorited) {
			return nil, err
		}
		logger.Error("add galgame favorite",
			zap.Uint("galgame_id", galgameID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	if s.activities != nil {
		metadata := map[string]any{}
		if galgame, findErr := s.galgames.FindPublishedByID(ctx, galgameID); findErr == nil && galgame != nil {
			metadata["title"] = galgame.Title
		}
		if recordErr := s.activities.Record(ctx, userID, userModel.ActivityFavoriteCreated, &galgameID, metadata); recordErr != nil {
			logger.Error("record favorite activity", zap.Uint("galgame_id", galgameID), zap.Error(recordErr))
		}
	}
	return favorite, nil
}

// RemoveFavorite deletes the favorite and atomically decrements
// favorite_count in one transaction.
func (s *FavoriteService) RemoveFavorite(ctx context.Context, galgameID, userID uint) error {
	if err := ensurePublishedGalgame(ctx, s.galgames, galgameID); err != nil {
		return err
	}

	err := s.relations.Transaction(ctx, func(tx *repository.UserRelationRepository) error {
		removed, err := tx.RemoveFavorite(ctx, galgameID, userID)
		if err != nil {
			return err
		}
		if !removed {
			return ErrFavoriteNotFound
		}
		return tx.DecrementFavoriteCount(ctx, galgameID)
	})
	if err != nil {
		if errors.Is(err, ErrFavoriteNotFound) {
			return err
		}
		logger.Error("remove galgame favorite",
			zap.Uint("galgame_id", galgameID), zap.Uint("user_id", userID), zap.Error(err))
		return err
	}
	return nil
}
