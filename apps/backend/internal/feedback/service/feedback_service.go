package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/feedback/dto"
	"backend/internal/feedback/model"
	"backend/internal/feedback/repository"

	"github.com/redis/go-redis/v9"
)

var (
	ErrFeedbackNotFound  = errors.New("feedback not found")
	ErrFeedbackRateLimit = errors.New("feedback rate limit exceeded")
	ErrInvalidFeedback   = errors.New("invalid feedback input")
)

const (
	feedbackIPWindow = time.Hour
	feedbackIPLimit  = 5
)

type FeedbackService struct {
	feedbacks *repository.FeedbackRepository
	cache     *redis.Client
}

func NewFeedbackService(
	feedbacks *repository.FeedbackRepository,
	cache *redis.Client,
) *FeedbackService {
	return &FeedbackService{feedbacks: feedbacks, cache: cache}
}

func (s *FeedbackService) Submit(ctx context.Context, req *dto.CreateFeedbackRequest, ip string, userAgent string) (*model.Feedback, error) {
	content := strings.TrimSpace(req.Content)
	if len(content) < 5 {
		return nil, ErrInvalidFeedback
	}
	if err := s.enforceIPLimit(ctx, ip); err != nil {
		return nil, err
	}
	feedback := &model.Feedback{
		Type:      req.Type,
		Content:   content,
		Contact:   strings.TrimSpace(req.Contact),
		IP:        ip,
		UserAgent: userAgent,
	}
	if err := s.feedbacks.Create(ctx, feedback); err != nil {
		return nil, err
	}
	return feedback, nil
}

func (s *FeedbackService) ListAdmin(ctx context.Context, page, limit int, filter repository.FeedbackFilter) ([]model.Feedback, int64, int, int, error) {
	page, limit = pagination(page, limit)
	items, total, err := s.feedbacks.ListAdmin(ctx, page, limit, filter)
	return items, total, page, limit, err
}

func (s *FeedbackService) Handle(ctx context.Context, id uint, handled bool, handlerID uint) (*model.Feedback, error) {
	feedback, err := s.feedbacks.UpdateHandled(ctx, id, handled, handlerID)
	if err != nil {
		return nil, err
	}
	if feedback == nil {
		return nil, ErrFeedbackNotFound
	}
	return feedback, nil
}

// enforceIPLimit applies a fixed-window counter per IP; limits are skipped
// when Redis is unavailable.
func (s *FeedbackService) enforceIPLimit(ctx context.Context, ip string) error {
	if s.cache == nil {
		return nil
	}
	key := fmt.Sprintf("feedback:ip:%s", ip)
	count, err := s.cache.Incr(ctx, key).Result()
	if err != nil {
		return nil
	}
	if count == 1 {
		s.cache.Expire(ctx, key, feedbackIPWindow)
	}
	if count > feedbackIPLimit {
		return ErrFeedbackRateLimit
	}
	return nil
}

func pagination(page, limit int) (int, int) {
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	return page, limit
}
