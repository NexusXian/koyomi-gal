package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	notificationModel "backend/internal/notification/model"
	notificationService "backend/internal/notification/service"
	userModel "backend/internal/user/model"
	userService "backend/internal/user/service"
	"backend/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	ErrInvalidScore   = errors.New("invalid rating score")
	ErrInvalidRating  = errors.New("invalid rating fields")
	ErrRatingNotFound = errors.New("galgame rating not found")
)

const ratingSummaryTTL = 10 * time.Minute

type RatingService struct {
	galgames      *repository.GalgameRepository
	relations     *repository.UserRelationRepository
	activities    userService.ActivityRecorder
	notifications *notificationService.NotificationService
	cache         *redis.Client
}

func (s *RatingService) SetActivityRecorder(recorder userService.ActivityRecorder) {
	s.activities = recorder
}

func (s *RatingService) SetNotificationService(notifications *notificationService.NotificationService) {
	s.notifications = notifications
}

func NewRatingService(
	galgames *repository.GalgameRepository,
	relations *repository.UserRelationRepository,
	cache ...*redis.Client,
) *RatingService {
	service := &RatingService{galgames: galgames, relations: relations}
	if len(cache) > 0 {
		service.cache = cache[0]
	}
	return service
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
	s.invalidateSummary(ctx, galgameID)
	s.recordActivity(ctx, galgameID, userID)
	return rating, nil
}

func (s *RatingService) PutRating(ctx context.Context, galgameID, userID uint, req *dto.PutRatingRequest) (*model.RatingView, error) {
	if req == nil || !validRatingValue(req.Overall) || !validRecommendation(req.Recommendation) || req.SpoilerLevel < 0 || req.SpoilerLevel > 2 ||
		!validOptionalRatingValue(req.Visual) || !validOptionalRatingValue(req.Story) ||
		!validOptionalRatingValue(req.Music) || !validOptionalRatingValue(req.Character) ||
		!validOptionalRatingValue(req.Branch) || !validOptionalRatingValue(req.System) ||
		!validOptionalRatingValue(req.Voice) || !validOptionalRatingValue(req.Replay) {
		return nil, ErrInvalidRating
	}
	if err := ensurePublishedGalgame(ctx, s.galgames, galgameID); err != nil {
		return nil, err
	}

	rating := &model.Rating{
		GalgameID: galgameID, UserID: userID, Score: req.Overall,
		Visual: req.Visual, Story: req.Story, Music: req.Music,
		Character: req.Character, Branch: req.Branch, System: req.System,
		Voice: req.Voice, Replay: req.Replay, Recommendation: req.Recommendation,
		ReviewText: req.ReviewText, SpoilerLevel: req.SpoilerLevel,
	}
	err := s.relations.Transaction(ctx, func(tx *repository.UserRelationRepository) error {
		if err := tx.UpsertDetailedRating(ctx, rating); err != nil {
			return err
		}
		return tx.RecalculateGalgameRating(ctx, galgameID)
	})
	if err != nil {
		logger.Error("put galgame rating", zap.Uint("galgame_id", galgameID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	s.invalidateSummary(ctx, galgameID)
	s.recordActivity(ctx, galgameID, userID)
	return s.relations.FindRatingView(ctx, galgameID, userID, userID)
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
	s.invalidateSummary(ctx, galgameID)
	return nil
}

func (s *RatingService) GetRating(ctx context.Context, galgameID, userID uint) (*model.Rating, error) {
	return s.relations.FindRating(ctx, galgameID, userID)
}

func (s *RatingService) GetMyRating(ctx context.Context, galgameID, userID uint) (*model.RatingView, error) {
	if err := ensurePublishedGalgame(ctx, s.galgames, galgameID); err != nil {
		return nil, err
	}
	rating, err := s.relations.FindRatingView(ctx, galgameID, userID, userID)
	if err != nil {
		return nil, err
	}
	return rating, nil
}

func (s *RatingService) ListRatings(ctx context.Context, galgameID uint, viewerID *uint, page, pageSize int, sort string) ([]model.RatingView, int64, error) {
	if err := ensurePublishedGalgame(ctx, s.galgames, galgameID); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if sort == "" {
		sort = "newest"
	}
	if sort != "newest" && sort != "highest" && sort != "lowest" && sort != "popular" {
		return nil, 0, ErrInvalidRating
	}
	var viewer uint
	if viewerID != nil {
		viewer = *viewerID
	}
	return s.relations.ListRatings(ctx, galgameID, repository.RatingListOptions{
		Page: page, PageSize: pageSize, Sort: sort, ViewerID: viewer,
	})
}

func (s *RatingService) Summary(ctx context.Context, galgameID uint) (*model.RatingSummary, error) {
	if err := ensurePublishedGalgame(ctx, s.galgames, galgameID); err != nil {
		return nil, err
	}
	key := ratingSummaryCacheKey(galgameID)
	if s.cache != nil {
		cached, err := s.cache.Get(ctx, key).Bytes()
		if err == nil {
			var summary model.RatingSummary
			if json.Unmarshal(cached, &summary) == nil {
				return &summary, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			logger.Warn("read galgame rating summary cache", zap.Uint("galgame_id", galgameID), zap.Error(err))
		}
	}
	summary, err := s.relations.RatingSummary(ctx, galgameID)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		if payload, marshalErr := json.Marshal(summary); marshalErr == nil {
			if cacheErr := s.cache.Set(ctx, key, payload, ratingSummaryTTL).Err(); cacheErr != nil {
				logger.Warn("cache galgame rating summary", zap.Uint("galgame_id", galgameID), zap.Error(cacheErr))
			}
		}
	}
	return summary, nil
}

func (s *RatingService) LikeRating(ctx context.Context, ratingID, userID uint) (*model.RatingView, error) {
	inserted := false
	err := s.relations.Transaction(ctx, func(tx *repository.UserRelationRepository) error {
		exists, err := tx.LockPublishedRating(ctx, ratingID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrRatingNotFound
		}
		inserted, err = tx.AddRatingLike(ctx, ratingID, userID)
		if err != nil || !inserted {
			return err
		}
		return tx.IncrementRatingLikeCount(ctx, ratingID)
	})
	if err != nil {
		logger.Error("like galgame rating", zap.Uint("rating_id", ratingID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	rating, err := s.relations.FindPublishedRatingByID(ctx, ratingID, userID)
	if err != nil {
		return nil, err
	}
	if inserted && rating != nil {
		s.notifyRatingLiked(ctx, rating, userID)
	}
	return rating, nil
}

func (s *RatingService) notifyRatingLiked(ctx context.Context, rating *model.RatingView, likerID uint) {
	if s.notifications == nil {
		return
	}
	content := "点赞了你的游戏评价"
	if galgame, err := s.galgames.FindPublishedByID(ctx, rating.GalgameID); err == nil && galgame != nil {
		content = fmt.Sprintf("点赞了你在「%s」下的评价", galgame.Title)
	}
	if _, err := s.notifications.Create(ctx, notificationService.CreateInput{
		RecipientID: rating.UserID, ActorID: &likerID,
		Category: notificationModel.CategoryInteraction, Type: notificationModel.TypeRatingLiked,
		EntityType: "galgame_rating", EntityID: rating.ID, Title: "评价收到点赞",
		Content: content, TargetURL: fmt.Sprintf("/galgames/%d", rating.GalgameID),
	}); err != nil {
		logger.Error("create rating like notification", zap.Uint("rating_id", rating.ID), zap.Error(err))
	}
}

func (s *RatingService) UnlikeRating(ctx context.Context, ratingID, userID uint) (*model.RatingView, error) {
	err := s.relations.Transaction(ctx, func(tx *repository.UserRelationRepository) error {
		exists, err := tx.LockPublishedRating(ctx, ratingID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrRatingNotFound
		}
		removed, err := tx.RemoveRatingLike(ctx, ratingID, userID)
		if err != nil || !removed {
			return err
		}
		return tx.DecrementRatingLikeCount(ctx, ratingID)
	})
	if err != nil {
		logger.Error("unlike galgame rating", zap.Uint("rating_id", ratingID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	return s.relations.FindPublishedRatingByID(ctx, ratingID, userID)
}

func validRatingValue(value int16) bool { return value >= 1 && value <= 10 }

func validOptionalRatingValue(value *int16) bool { return value == nil || validRatingValue(*value) }

func validRecommendation(value *int16) bool {
	return value == nil || *value == -1 || *value == 0 || *value == 1 || *value == 2
}

func ratingSummaryCacheKey(galgameID uint) string {
	return fmt.Sprintf("game:rating:summary:%d", galgameID)
}

func (s *RatingService) invalidateSummary(ctx context.Context, galgameID uint) {
	if s.cache == nil {
		return
	}
	if err := s.cache.Del(ctx, ratingSummaryCacheKey(galgameID)).Err(); err != nil {
		logger.Warn("invalidate galgame rating summary cache", zap.Uint("galgame_id", galgameID), zap.Error(err))
	}
}

func (s *RatingService) recordActivity(ctx context.Context, galgameID, userID uint) {
	if s.activities == nil {
		return
	}
	metadata := map[string]any{}
	if galgame, err := s.galgames.FindPublishedByID(ctx, galgameID); err == nil && galgame != nil {
		metadata["title"] = galgame.Title
	}
	if err := s.activities.Record(ctx, userID, userModel.ActivityRatingCreated, &galgameID, metadata); err != nil {
		logger.Error("record rating activity", zap.Uint("galgame_id", galgameID), zap.Error(err))
	}
}
