package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	imageModel "backend/internal/image/model"
	"backend/internal/notification/model"

	"gorm.io/gorm"
)

var ErrNotificationNotFound = errors.New("notification not found")

type ListOptions struct {
	RecipientID uint
	Page        int
	Limit       int
	Category    model.NotificationCategory
	Unread      *bool
}

type NotificationRepository struct {
	db            *gorm.DB
	avatarBaseURL string
}

func NewNotificationRepository(db *gorm.DB, avatarBaseURL string) *NotificationRepository {
	return &NotificationRepository{db: db, avatarBaseURL: strings.TrimRight(avatarBaseURL, "/")}
}

func (r *NotificationRepository) Create(ctx context.Context, notification *model.Notification) error {
	if err := r.db.WithContext(ctx).Create(notification).Error; err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

func (r *NotificationRepository) CreateMany(ctx context.Context, notifications []model.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&notifications).Error; err != nil {
		return fmt.Errorf("create notifications: %w", err)
	}
	return nil
}

func (r *NotificationRepository) List(ctx context.Context, options ListOptions) ([]model.Notification, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&model.Notification{}).
			Where("notifications.recipient_id = ?", options.RecipientID)
		if options.Category != "" {
			query = query.Where("notifications.category = ?", options.Category)
		}
		if options.Unread != nil {
			if *options.Unread {
				query = query.Where("notifications.is_read = FALSE")
			} else {
				query = query.Where("notifications.is_read = TRUE")
			}
		}
		return query
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	notifications := make([]model.Notification, 0)
	err := base().
		Select(`notifications.*, actors.username AS actor_name,
CASE WHEN actor_avatars.object_key IS NOT NULL
THEN CAST(? AS text) || '/' || actor_avatars.object_key
ELSE actors.avatar END AS actor_avatar`, r.avatarBaseURL).
		Joins("LEFT JOIN users AS actors ON actors.id = notifications.actor_id").
		Joins(fmt.Sprintf(`LEFT JOIN image_assets AS actor_avatars
ON actor_avatars.id = actors.avatar_asset_id
AND actor_avatars.user_id = actors.id
AND actor_avatars.status = %d`, imageModel.ImageStatusActive)).
		Order("notifications.created_at DESC").Order("notifications.id DESC").
		Offset((options.Page - 1) * options.Limit).Limit(options.Limit).
		Find(&notifications).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	return notifications, total, nil
}

func (r *NotificationRepository) UnreadCount(ctx context.Context, recipientID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("recipient_id = ? AND is_read = FALSE", recipientID).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, recipientID, id uint) (*model.Notification, error) {
	result := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ? AND recipient_id = ?", id, recipientID).
		Updates(map[string]any{"is_read": true, "read_at": gorm.Expr("NOW()")})
	if result.Error != nil {
		return nil, fmt.Errorf("mark notification read: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotificationNotFound
	}

	var notification model.Notification
	if err := r.db.WithContext(ctx).Where("id = ? AND recipient_id = ?", id, recipientID).
		First(&notification).Error; err != nil {
		return nil, fmt.Errorf("load marked notification: %w", err)
	}
	return &notification, nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, recipientID uint) (int64, error) {
	result := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("recipient_id = ? AND is_read = FALSE", recipientID).
		Updates(map[string]any{"is_read": true, "read_at": gorm.Expr("NOW()")})
	if result.Error != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", result.Error)
	}
	return result.RowsAffected, nil
}
