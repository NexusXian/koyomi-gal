package service

import (
	"context"
	"errors"
	"strings"

	"backend/internal/notification/model"
	"backend/internal/notification/repository"
)

var (
	ErrNotificationNotFound = repository.ErrNotificationNotFound
	ErrInvalidNotification  = errors.New("invalid notification")
)

type CreateInput struct {
	RecipientID uint
	ActorID     *uint
	Category    model.NotificationCategory
	Type        model.NotificationType
	EntityType  string
	EntityID    uint
	Title       string
	Content     string
	TargetURL   string
	Metadata    map[string]any
}

type NotificationService struct {
	notifications *repository.NotificationRepository
}

func NewNotificationService(notifications *repository.NotificationRepository) *NotificationService {
	return &NotificationService{notifications: notifications}
}

func (s *NotificationService) Create(ctx context.Context, input CreateInput) (*model.Notification, error) {
	if input.ActorID != nil && *input.ActorID == input.RecipientID {
		return nil, nil
	}
	notification, err := notificationFromInput(input)
	if err != nil {
		return nil, err
	}
	if err := s.notifications.Create(ctx, &notification); err != nil {
		return nil, err
	}
	return &notification, nil
}

func (s *NotificationService) CreateMany(ctx context.Context, inputs []CreateInput) ([]model.Notification, error) {
	notifications := make([]model.Notification, 0, len(inputs))
	for _, input := range inputs {
		if input.ActorID != nil && *input.ActorID == input.RecipientID {
			continue
		}
		notification, err := notificationFromInput(input)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	if len(notifications) == 0 {
		return notifications, nil
	}
	if err := s.notifications.CreateMany(ctx, notifications); err != nil {
		return nil, err
	}
	return notifications, nil
}

func (s *NotificationService) List(
	ctx context.Context,
	recipientID uint,
	page int,
	limit int,
	category model.NotificationCategory,
	unread *bool,
) ([]model.Notification, int64, int, int, error) {
	page, limit = notificationPagination(page, limit)
	notifications, total, err := s.notifications.List(ctx, repository.ListOptions{
		RecipientID: recipientID, Page: page, Limit: limit, Category: category, Unread: unread,
	})
	return notifications, total, page, limit, err
}

func (s *NotificationService) UnreadCount(ctx context.Context, recipientID uint) (int64, error) {
	return s.notifications.UnreadCount(ctx, recipientID)
}

func (s *NotificationService) MarkRead(ctx context.Context, recipientID, id uint) (*model.Notification, error) {
	return s.notifications.MarkRead(ctx, recipientID, id)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, recipientID uint) (int64, error) {
	return s.notifications.MarkAllRead(ctx, recipientID)
}

func notificationFromInput(input CreateInput) (model.Notification, error) {
	input.Category = model.NotificationCategory(strings.TrimSpace(string(input.Category)))
	input.Type = model.NotificationType(strings.TrimSpace(string(input.Type)))
	input.EntityType = strings.TrimSpace(input.EntityType)
	input.Title = strings.TrimSpace(input.Title)
	input.TargetURL = strings.TrimSpace(input.TargetURL)
	if input.RecipientID == 0 || !model.IsValidCategory(input.Category) || !model.IsValidType(input.Type) || input.Title == "" {
		return model.Notification{}, ErrInvalidNotification
	}
	metadata, err := model.NewMetadata(input.Metadata)
	if err != nil {
		return model.Notification{}, err
	}
	notification := model.Notification{
		RecipientID: input.RecipientID, ActorID: input.ActorID, Category: input.Category,
		Type: input.Type, Title: input.Title, Content: input.Content,
		Metadata: metadata,
	}
	if input.EntityType != "" {
		notification.EntityType = &input.EntityType
	}
	if input.EntityID != 0 {
		notification.EntityID = &input.EntityID
	}
	if input.TargetURL != "" {
		notification.TargetURL = &input.TargetURL
	}
	return notification, nil
}

func notificationPagination(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
