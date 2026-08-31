package dto

import (
	"time"

	"backend/internal/resource/model"
)

type CreateResourceRequest struct {
	GalgameID   uint     `json:"galgame_id" binding:"required,gt=0" example:"1"`
	Title       string   `json:"title" binding:"required,max=255" example:"千恋＊万花 官方整合包"`
	Type        int16    `json:"type" binding:"oneof=0 1 2 3 4 5 6" example:"1"`
	Description string   `json:"description" example:"官方汉化整合包"`
	Status      *int16   `json:"status" binding:"omitempty,oneof=0 1 2 3" example:"1"`
	Links       []string `json:"links" binding:"required,min=1,max=50,dive,required,min=1,max=2048" example:"https://example.com/dl,https://example.com/dl2"`
}

type UpdateResourceRequest struct {
	Title       string   `json:"title" binding:"required,max=255" example:"千恋＊万花 官方整合包"`
	Type        int16    `json:"type" binding:"oneof=0 1 2 3 4 5 6" example:"1"`
	Description string   `json:"description" example:"官方汉化整合包"`
	Status      *int16   `json:"status" binding:"required,oneof=0 1 2 3" example:"1"`
	Links       []string `json:"links" binding:"required,min=1,max=50,dive,required,min=1,max=2048" example:"https://example.com/dl"`
}

type ReviewResourceRequest struct {
	Status int16 `json:"status" binding:"required,oneof=1 2 3" example:"1"`
}

type ResourceQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type ResourceLinkData struct {
	ID        uint      `json:"id" example:"1"`
	URL       string    `json:"url" example:"https://example.com/dl"`
	CreatedAt time.Time `json:"created_at"`
}

type ResourceData struct {
	ID          uint               `json:"id" example:"1"`
	GalgameID   uint               `json:"galgame_id" example:"1"`
	UploaderID  *uint              `json:"uploader_id" example:"1"`
	Title       string             `json:"title" example:"千恋＊万花 官方整合包"`
	Type        int16              `json:"type" example:"1"`
	Description string             `json:"description" example:"官方汉化整合包"`
	Status      int16              `json:"status" example:"1"`
	Links       []ResourceLinkData `json:"links"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type ResourceListData struct {
	Items []ResourceData `json:"items"`
	Total int64          `json:"total" example:"10"`
	Page  int            `json:"page" example:"1"`
	Limit int            `json:"limit" example:"20"`
}

type ResourceListResponse struct {
	Code int              `json:"code" example:"0"`
	Data ResourceListData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

type ResourceDataResponse struct {
	Code int          `json:"code" example:"0"`
	Data ResourceData `json:"data"`
	Msg  string       `json:"msg" example:"success"`
}

type AdminResourceListData struct {
	Items []ResourceData `json:"items"`
	Total int64          `json:"total" example:"10"`
	Page  int            `json:"page" example:"1"`
	Limit int            `json:"limit" example:"20"`
}

type AdminResourceListResponse struct {
	Code int                   `json:"code" example:"0"`
	Data AdminResourceListData `json:"data"`
	Msg  string                `json:"msg" example:"success"`
}

func NewResourceData(resource *model.Resource) ResourceData {
	links := make([]ResourceLinkData, 0, len(resource.Links))
	for _, link := range resource.Links {
		links = append(links, ResourceLinkData{
			ID:        link.ID,
			URL:       link.URL,
			CreatedAt: link.CreatedAt,
		})
	}
	return ResourceData{
		ID:          resource.ID,
		GalgameID:   resource.GalgameID,
		UploaderID:  resource.UploaderID,
		Title:       resource.Title,
		Type:        resource.Type,
		Description: resource.Description,
		Status:      resource.Status,
		Links:       links,
		CreatedAt:   resource.CreatedAt,
		UpdatedAt:   resource.UpdatedAt,
	}
}

func NewResourceListData(resources []model.Resource) []ResourceData {
	items := make([]ResourceData, 0, len(resources))
	for i := range resources {
		items = append(items, NewResourceData(&resources[i]))
	}
	return items
}
