package dto

import (
	"time"

	changelogModel "backend/internal/changelog/model"
)

type ChangelogItemPayload struct {
	Type string `json:"type" binding:"required,oneof=new improve fix" example:"new"`
	Text string `json:"text" binding:"required,max=255" example:"新增站点页脚与更新日志页面"`
}

type ChangelogRequest struct {
	Version     string                 `json:"version" binding:"required,max=32" example:"0.5.0"`
	Title       string                 `json:"title" binding:"required,max=255" example:"站点功能更新"`
	PublishedAt *time.Time             `json:"publishedAt" example:"2026-09-08T12:00:00Z"`
	Items       []ChangelogItemPayload `json:"items" binding:"required,min=1,max=50,dive"`
}

type ChangelogItemData struct {
	Type string `json:"type" example:"new"`
	Text string `json:"text" example:"新增站点页脚与更新日志页面"`
}

type ChangelogData struct {
	ID          uint                `json:"id" example:"1"`
	Version     string              `json:"version" example:"0.5.0"`
	Title       string              `json:"title" example:"站点功能更新"`
	Items       []ChangelogItemData `json:"items"`
	PublishedAt time.Time           `json:"publishedAt"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

type ChangelogPublicListData struct {
	Items []ChangelogData `json:"items"`
}

type ChangelogPublicListResponse struct {
	Code int                     `json:"code" example:"0"`
	Data ChangelogPublicListData `json:"data"`
	Msg  string                  `json:"msg" example:"success"`
}

type ChangelogListData struct {
	Items []ChangelogData `json:"items"`
	Total int64           `json:"total" example:"5"`
	Page  int             `json:"page" example:"1"`
	Limit int             `json:"limit" example:"20"`
}

type ChangelogListResponse struct {
	Code int               `json:"code" example:"0"`
	Data ChangelogListData `json:"data"`
	Msg  string            `json:"msg" example:"success"`
}

type ChangelogDataResponse struct {
	Code int           `json:"code" example:"0"`
	Data ChangelogData `json:"data"`
	Msg  string        `json:"msg" example:"success"`
}

type AdminChangelogQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

func NewChangelogData(value *changelogModel.SiteChangelog) ChangelogData {
	items := make([]ChangelogItemData, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, ChangelogItemData{Type: item.Type, Text: item.Text})
	}
	return ChangelogData{
		ID: value.ID, Version: value.Version, Title: value.Title, Items: items,
		PublishedAt: value.PublishedAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func NewChangelogList(values []changelogModel.SiteChangelog) []ChangelogData {
	items := make([]ChangelogData, 0, len(values))
	for i := range values {
		items = append(items, NewChangelogData(&values[i]))
	}
	return items
}
