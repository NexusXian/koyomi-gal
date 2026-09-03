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
	ErrInvalidScore   = errors.New("invalid rating score")
	ErrRatingNotFound = errors.New("galgame rating not found")
)

type RatingService struct {
	galgames   *repository.GalgameRepository
	relations  *repository.UserRelationRepository
	activities userService.ActivityRecorder
}

func (s *RatingService) SetActivityRecorder(recorder userService.ActivityRecorder) {
	s.activities = recorder
}

func NewRatingService(
	galgames *repository.GalgameRepository,
	relations *repository.UserRelationRepository,
) *RatingService {
	return &RatingService{galgames: galgames, relations: relations}
}

// UpsertRating creates or updates the user's score and recomputes the
// galgame's rating_average and rating_count in one transaction.
func (s *RatingService) UpsertRating(
	ctx context.Context,
	galgameID, userID uint,
	score int16,
) (*model.Rating, error) {
	if score < 1 || score > 10 {
		return nil, ErrInvalidScore
	}
	if err := ensurePublishedGalgame(ctx, s.galgames, galgameID); err != nil {
		return nil, err
	}

	var rating *model.Rating
	err := s.relations.Transaction(ctx, func(tx *repository.UserRelationRepository) error {
		if err := tx.UpsertRating(ctx, galgameID, userID, score); err != nil {
			return err
		}
		if err := tx.RecalculateGalgameRating(ctx, galgameID); err != nil {
			return err
		}
		var txErr error
		rating, txErr = tx.FindRating(ctx, galgameID, userID)
		return txErr
	})
	if err != nil {
		logger.Error("upsert galgame rating",
			zap.Uint("galgame_id", galgameID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	if s.activities != nil {
		metadata := map[string]any{}
		if galgame, findErr := s.galgames.FindPublishedByID(ctx, galgameID); findErr == nil && galgame != nil {
			metadata["title"] = galgame.Title
		}
		if recordErr := s.activities.Record(ctx, userID, userModel.ActivityRatingCreated, &galgameID, metadata); recordErr != nil {
			logger.Error("record rating activity", zap.Uint("galgame_id", galgameID), zap.Error(recordErr))
		}
	}
	return rating, nil
}

// DeleteRating removes the user's score and recomputes the galgame's
// rating_average and rating_count in one transaction.
func (s *RatingService) DeleteRating(ctx context.Context, galgameID, userID uint) error {
	if err := ensurePublishedGalgame(ctx, s.galgames, galgameID); err != nil {
		return err
	}

	err := s.relations.Transaction(ctx, func(tx *repository.UserRelationRepository) error {
		removed, err := tx.DeleteRating(ctx, galgameID, userID)
		if err != nil {
			return err
		}
		if !removed {
			return ErrRatingNotFound
		}
		return tx.RecalculateGalgameRating(ctx, galgameID)
	})
	if err != nil {
		if errors.Is(err, ErrRatingNotFound) {
			return err
		}
		logger.Error("delete galgame rating",
			zap.Uint("galgame_id", galgameID), zap.Uint("user_id", userID), zap.Error(err))
		return err
	}
	return nil
}

func (s *RatingService) GetRating(ctx context.Context, galgameID, userID uint) (*model.Rating, error) {
	return s.relations.FindRating(ctx, galgameID, userID)
}
