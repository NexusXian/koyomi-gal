package dto

import (
	"time"

	"backend/internal/article/model"
)

type CreateArticleRequest struct {
	Title       string     `json:"title" binding:"required,max=255" example:"站点更新公告"`
	Summary     string     `json:"summary" binding:"max=500" example:"本次更新内容概览"`
	Content     string     `json:"content" binding:"required" example:"完整公告内容"`
	CoverURL    string     `json:"cover_url" binding:"max=2048" example:"https://example.com/cover.jpg"`
	Type        string     `json:"type" binding:"required,oneof=announcement news event update" example:"announcement"`
	IsPinned    bool       `json:"is_pinned" example:"true"`
	IsPublished *bool      `json:"is_published" binding:"required" example:"true"`
	PublishedAt *time.Time `json:"published_at" example:"2026-08-31T00:00:00Z"`
}

type UpdateArticleRequest struct {
	Title       string     `json:"title" binding:"required,max=255" example:"站点更新公告"`
	Summary     string     `json:"summary" binding:"max=500" example:"本次更新内容概览"`
	Content     string     `json:"content" binding:"required" example:"完整公告内容"`
	CoverURL    string     `json:"cover_url" binding:"max=2048" example:"https://example.com/cover.jpg"`
	Type        string     `json:"type" binding:"required,oneof=announcement news event update" example:"announcement"`
	IsPinned    bool       `json:"is_pinned" example:"true"`
	IsPublished *bool      `json:"is_published" binding:"required" example:"true"`
	PublishedAt *time.Time `json:"published_at" example:"2026-08-31T00:00:00Z"`
}

type ArticleQuery struct {
	Type  string `form:"type" binding:"omitempty,oneof=announcement news event update"`
	Page  int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

type AdminArticleQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type ArticleListItem struct {
	ID          uint       `json:"id" example:"1"`
	Title       string     `json:"title" example:"站点更新公告"`
	Summary     string     `json:"summary" example:"本次更新内容概览"`
	CoverURL    string     `json:"cover_url" example:"https://example.com/cover.jpg"`
	Type        string     `json:"type" example:"announcement"`
	IsPinned    bool       `json:"is_pinned" example:"true"`
	IsPublished bool       `json:"is_published" example:"true"`
	PublishedAt *time.Time `json:"published_at"`
	ViewCount   int64      `json:"view_count" example:"100"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ArticleData struct {
	ArticleListItem
	Content string `json:"content" example:"完整公告内容"`
}

type ArticleListData struct {
	Items []ArticleListItem `json:"items"`
	Total int64             `json:"total" example:"10"`
	Page  int               `json:"page" example:"1"`
	Limit int               `json:"limit" example:"20"`
}

type ArticleListResponse struct {
	Code int             `json:"code" example:"0"`
	Data ArticleListData `json:"data"`
	Msg  string          `json:"msg" example:"success"`
}

type AdminArticleListData struct {
	Items []ArticleData `json:"items"`
	Total int64         `json:"total" example:"10"`
	Page  int           `json:"page" example:"1"`
	Limit int           `json:"limit" example:"20"`
}

type AdminArticleListResponse struct {
	Code int                  `json:"code" example:"0"`
	Data AdminArticleListData `json:"data"`
	Msg  string               `json:"msg" example:"success"`
}

type ArticleDataResponse struct {
	Code int         `json:"code" example:"0"`
	Data ArticleData `json:"data"`
	Msg  string      `json:"msg" example:"success"`
}

func NewArticleListItem(article *model.Article) ArticleListItem {
	return ArticleListItem{
		ID: article.ID, Title: article.Title, Summary: article.Summary,
		CoverURL: article.CoverURL, Type: article.Type, IsPinned: article.IsPinned,
		IsPublished: article.IsPublished,
		PublishedAt: article.PublishedAt, ViewCount: article.ViewCount,
		CreatedAt: article.CreatedAt, UpdatedAt: article.UpdatedAt,
	}
}

func NewArticleData(article *model.Article) ArticleData {
	return ArticleData{ArticleListItem: NewArticleListItem(article), Content: article.Content}
}

func NewArticleList(articles []model.Article) []ArticleListItem {
	items := make([]ArticleListItem, 0, len(articles))
	for i := range articles {
		items = append(items, NewArticleListItem(&articles[i]))
	}
	return items
}

func NewAdminArticleList(articles []model.Article) []ArticleData {
	items := make([]ArticleData, 0, len(articles))
	for i := range articles {
		items = append(items, NewArticleData(&articles[i]))
	}
	return items
}
