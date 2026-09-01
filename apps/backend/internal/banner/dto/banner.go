package dto

import (
	"time"

	"backend/internal/banner/model"
)

type CreateBannerRequest struct {
	Title     string     `json:"title" binding:"required,max=255" example:"夏日新作专题"`
	Subtitle  string     `json:"subtitle" binding:"max=500" example:"发现本月值得关注的新作"`
	ImageURL  string     `json:"image_url" binding:"required,max=2048" example:"https://example.com/banner.jpg"`
	LinkType  string     `json:"link_type" binding:"required,oneof=none url galgame post news" example:"galgame"`
	LinkValue string     `json:"link_value" binding:"max=2048" example:"1"`
	SortOrder int        `json:"sort_order" example:"100"`
	IsActive  *bool      `json:"is_active" example:"true"`
	StartAt   *time.Time `json:"start_at" example:"2026-08-01T00:00:00Z"`
	EndAt     *time.Time `json:"end_at" example:"2026-09-01T00:00:00Z"`
}

type UpdateBannerRequest struct {
	Title     string     `json:"title" binding:"required,max=255" example:"夏日新作专题"`
	Subtitle  string     `json:"subtitle" binding:"max=500" example:"发现本月值得关注的新作"`
	ImageURL  string     `json:"image_url" binding:"required,max=2048" example:"https://example.com/banner.jpg"`
	LinkType  string     `json:"link_type" binding:"required,oneof=none url galgame post news" example:"galgame"`
	LinkValue string     `json:"link_value" binding:"max=2048" example:"1"`
	SortOrder int        `json:"sort_order" example:"100"`
	IsActive  *bool      `json:"is_active" binding:"required" example:"true"`
	StartAt   *time.Time `json:"start_at" example:"2026-08-01T00:00:00Z"`
	EndAt     *time.Time `json:"end_at" example:"2026-09-01T00:00:00Z"`
}

type AdminBannerQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type BannerData struct {
	ID        uint       `json:"id" example:"1"`
	Title     string     `json:"title" example:"夏日新作专题"`
	Subtitle  string     `json:"subtitle" example:"发现本月值得关注的新作"`
	ImageURL  string     `json:"image_url" example:"https://example.com/banner.jpg"`
	LinkType  string     `json:"link_type" example:"galgame"`
	LinkValue string     `json:"link_value" example:"1"`
	SortOrder int        `json:"sort_order" example:"100"`
	IsActive  bool       `json:"is_active" example:"true"`
	StartAt   *time.Time `json:"start_at"`
	EndAt     *time.Time `json:"end_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type BannerListData struct {
	Items []BannerData `json:"items"`
	Total int64        `json:"total" example:"10"`
	Page  int          `json:"page" example:"1"`
	Limit int          `json:"limit" example:"20"`
}

type BannerListResponse struct {
	Code int          `json:"code" example:"0"`
	Data []BannerData `json:"data"`
	Msg  string       `json:"msg" example:"success"`
}

type AdminBannerListResponse struct {
	Code int            `json:"code" example:"0"`
	Data BannerListData `json:"data"`
	Msg  string         `json:"msg" example:"success"`
}

type BannerDataResponse struct {
	Code int        `json:"code" example:"0"`
	Data BannerData `json:"data"`
	Msg  string     `json:"msg" example:"success"`
}

func NewBannerData(banner *model.Banner) BannerData {
	return BannerData{
		ID: banner.ID, Title: banner.Title, Subtitle: banner.Subtitle, ImageURL: banner.ImageURL,
		LinkType: banner.LinkType, LinkValue: banner.LinkValue,
		SortOrder: banner.SortOrder, IsActive: banner.IsActive,
		StartAt: banner.StartAt, EndAt: banner.EndAt,
		CreatedAt: banner.CreatedAt, UpdatedAt: banner.UpdatedAt,
	}
}

func NewBannerList(banners []model.Banner) []BannerData {
	items := make([]BannerData, 0, len(banners))
	for i := range banners {
		items = append(items, NewBannerData(&banners[i]))
	}
	return items
}
