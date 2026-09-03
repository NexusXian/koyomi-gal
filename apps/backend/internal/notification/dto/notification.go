package dto

import (
	"time"

	"backend/internal/notification/model"
)

type ListNotificationsQuery struct {
	Page     int                        `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit    int                        `form:"limit" binding:"omitempty,min=1,max=100"`
	Category model.NotificationCategory `form:"category" binding:"omitempty,oneof=interaction review moderation system"`
	Unread   *bool                      `form:"unread"`
}

type ActorData struct {
	ID          uint   `json:"id" example:"2"`
	Username    string `json:"username" example:"koyomi"`
	DisplayName string `json:"display_name" example:"Koyomi"`
	Avatar      string `json:"avatar" example:"https://cdn.example.com/avatars/2/avatar.webp"`
	AvatarURL   string `json:"avatar_url" example:"https://cdn.example.com/avatars/2/avatar.webp"`
}

type NotificationData struct {
	ID         uint                       `json:"id" example:"1"`
	Actor      *ActorData                 `json:"actor"`
	Category   model.NotificationCategory `json:"category" example:"interaction"`
	Type       model.NotificationType     `json:"type" example:"post_liked"`
	EntityType *string                    `json:"entity_type" example:"post"`
	EntityID   *uint                      `json:"entity_id" example:"42"`
	Title      string                     `json:"title" example:"你的帖子收到了赞"`
	Content    string                     `json:"content" example:"koyomi 赞了你的帖子"`
	TargetURL  *string                    `json:"target_url" example:"/posts/42"`
	Metadata   model.Metadata             `json:"metadata" swaggertype:"object"`
	IsRead     bool                       `json:"is_read" example:"false"`
	ReadAt     *time.Time                 `json:"read_at"`
	CreatedAt  time.Time                  `json:"created_at"`
}

type NotificationListData struct {
	Items []NotificationData `json:"items"`
	Total int64              `json:"total" example:"10"`
	Page  int                `json:"page" example:"1"`
	Limit int                `json:"limit" example:"20"`
}

type UnreadCountData struct {
	Count int64 `json:"count" example:"3"`
}

type NotificationListResponse struct {
	Code int                  `json:"code" example:"0"`
	Data NotificationListData `json:"data"`
	Msg  string               `json:"msg" example:"success"`
}

type NotificationDataResponse struct {
	Code int              `json:"code" example:"0"`
	Data NotificationData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

type UnreadCountResponse struct {
	Code int             `json:"code" example:"0"`
	Data UnreadCountData `json:"data"`
	Msg  string          `json:"msg" example:"success"`
}

func NewNotificationData(notification *model.Notification) NotificationData {
	data := NotificationData{
		ID: notification.ID, Category: notification.Category, Type: notification.Type,
		EntityType: notification.EntityType, EntityID: notification.EntityID,
		Title: notification.Title, Content: notification.Content, TargetURL: notification.TargetURL,
		Metadata: notification.Metadata, IsRead: notification.IsRead,
		ReadAt: notification.ReadAt, CreatedAt: notification.CreatedAt,
	}
	if notification.ActorID != nil {
		data.Actor = &ActorData{ID: *notification.ActorID, Username: notification.ActorName, DisplayName: notification.ActorDisplayName, Avatar: notification.ActorAvatar, AvatarURL: notification.ActorAvatar}
	}
	if data.Metadata == nil {
		data.Metadata = model.Metadata{}
	}
	return data
}

func NewNotificationList(notifications []model.Notification) []NotificationData {
	items := make([]NotificationData, 0, len(notifications))
	for i := range notifications {
		items = append(items, NewNotificationData(&notifications[i]))
	}
	return items
}
