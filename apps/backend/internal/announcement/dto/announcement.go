package dto

import (
	"time"

	"backend/internal/announcement/model"
)

type ActiveAnnouncementQuery struct {
	Platform string `form:"platform" binding:"required,oneof=web android ios windows macos linux desktop"`
}

type AdminAnnouncementQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type AnnouncementRequest struct {
	Title       string     `json:"title" binding:"required,max=255"`
	Content     string     `json:"content" binding:"required"`
	Type        string     `json:"type" binding:"required,oneof=normal update maintenance warning event system"`
	DisplayMode string     `json:"displayMode" binding:"required,oneof=normal banner modal startup_modal"`
	Target      string     `json:"target" binding:"required,oneof=all web android ios desktop"`
	Priority    int        `json:"priority"`
	StartsAt    *time.Time `json:"startsAt"`
	EndsAt      *time.Time `json:"endsAt"`
	Dismissible *bool      `json:"dismissible"`
	Published   *bool      `json:"published"`
}

type ActiveAnnouncementData struct {
	ID          uint       `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Type        string     `json:"type"`
	DisplayMode string     `json:"displayMode"`
	Target      string     `json:"target"`
	Priority    int        `json:"priority"`
	Dismissible bool       `json:"dismissible"`
	StartsAt    *time.Time `json:"startsAt"`
	EndsAt      *time.Time `json:"endsAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type AnnouncementData struct {
	ID          uint       `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Type        string     `json:"type"`
	DisplayMode string     `json:"displayMode"`
	Target      string     `json:"target"`
	Priority    int        `json:"priority"`
	Dismissible bool       `json:"dismissible"`
	Published   bool       `json:"published"`
	StartsAt    *time.Time `json:"startsAt"`
	EndsAt      *time.Time `json:"endsAt"`
	CreatedBy   *uint      `json:"createdBy"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type AnnouncementListData struct {
	Items []AnnouncementData `json:"items"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

type AnnouncementListResponse struct {
	Code int                      `json:"code"`
	Data []ActiveAnnouncementData `json:"data"`
	Msg  string                   `json:"msg"`
}

type AdminAnnouncementListResponse struct {
	Code int                  `json:"code"`
	Data AnnouncementListData `json:"data"`
	Msg  string               `json:"msg"`
}

type AnnouncementDataResponse struct {
	Code int              `json:"code"`
	Data AnnouncementData `json:"data"`
	Msg  string           `json:"msg"`
}

func NewAnnouncementData(value *model.Announcement) AnnouncementData {
	return AnnouncementData{
		ID: value.ID, Title: value.Title, Content: value.Content, Type: value.Type,
		DisplayMode: value.DisplayMode, Target: value.Target, Priority: value.Priority,
		Dismissible: value.Dismissible, Published: value.Published,
		StartsAt: value.StartsAt, EndsAt: value.EndsAt, CreatedBy: value.CreatedBy,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func NewAnnouncementList(values []model.Announcement) []AnnouncementData {
	items := make([]AnnouncementData, 0, len(values))
	for i := range values {
		items = append(items, NewAnnouncementData(&values[i]))
	}
	return items
}

func NewActiveAnnouncementList(values []model.Announcement) []ActiveAnnouncementData {
	items := make([]ActiveAnnouncementData, 0, len(values))
	for i := range values {
		value := &values[i]
		items = append(items, ActiveAnnouncementData{
			ID: value.ID, Title: value.Title, Content: value.Content, Type: value.Type,
			DisplayMode: value.DisplayMode, Target: value.Target, Priority: value.Priority,
			Dismissible: value.Dismissible, StartsAt: value.StartsAt, EndsAt: value.EndsAt,
			UpdatedAt: value.UpdatedAt,
		})
	}
	return items
}
