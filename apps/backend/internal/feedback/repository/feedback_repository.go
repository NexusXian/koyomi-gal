package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/feedback/model"

	"gorm.io/gorm"
)

type FeedbackFilter struct {
	Type    string
	Handled *bool
}

type FeedbackRepository struct {
	db *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

func (r *FeedbackRepository) Create(ctx context.Context, feedback *model.Feedback) error {
	if err := r.db.WithContext(ctx).Create(feedback).Error; err != nil {
		return fmt.Errorf("create feedback: %w", err)
	}
	return nil
}

func (r *FeedbackRepository) FindByID(ctx context.Context, id uint) (*model.Feedback, error) {
	var feedback model.Feedback
	err := r.db.WithContext(ctx).First(&feedback, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find feedback by id: %w", err)
	}
	return &feedback, nil
}

func (r *FeedbackRepository) ListAdmin(ctx context.Context, page, limit int, filter FeedbackFilter) ([]model.Feedback, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Feedback{})
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Handled != nil {
		if *filter.Handled {
			query = query.Where("handled_at IS NOT NULL")
		} else {
			query = query.Where("handled_at IS NULL")
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count feedback: %w", err)
	}
	items := make([]model.Feedback, 0)
	err := query.Order("id DESC").
		Offset((page - 1) * limit).Limit(limit).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list admin feedback: %w", err)
	}
	return items, total, nil
}

func (r *FeedbackRepository) UpdateHandled(ctx context.Context, id uint, handled bool, handlerID uint) (*model.Feedback, error) {
	updates := map[string]any{
		"handled_by": nil,
		"handled_at": nil,
		"updated_at": time.Now(),
	}
	if handled {
		updates["handled_by"] = handlerID
		updates["handled_at"] = time.Now()
	}
	result := r.db.WithContext(ctx).Model(&model.Feedback{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, fmt.Errorf("update feedback handled: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return r.FindByID(ctx, id)
}
