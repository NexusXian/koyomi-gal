package dto

import (
	"time"

	"backend/internal/background/model"
)

type CreateBackgroundPresetRequest struct {
	Name      string `json:"name" binding:"required,max=255" example:"暮色花海"`
	ImageURL  string `json:"image_url" binding:"required,max=2048" example:"presets/backgrounds/default-01.webp"`
	SortOrder int    `json:"sort_order" example:"50"`
	IsActive  *bool  `json:"is_active" example:"true"`
}

type UpdateBackgroundPresetRequest struct {
	Name      string `json:"name" binding:"required,max=255" example:"暮色花海"`
	ImageURL  string `json:"image_url" binding:"required,max=2048" example:"presets/backgrounds/default-01.webp"`
	SortOrder int    `json:"sort_order" example:"50"`
	IsActive  *bool  `json:"is_active" binding:"required" example:"true"`
}

type AdminBackgroundPresetQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type BackgroundPresetData struct {
	ID        uint      `json:"id" example:"1"`
	Key       string    `json:"key" example:"default-01"`
	Name      string    `json:"name" example:"暮色花海"`
	ImageURL  string    `json:"image_url" example:"https://img.example.com/presets/backgrounds/default-01.webp"`
	SortOrder int       `json:"sort_order" example:"50"`
	IsActive  bool      `json:"is_active" example:"true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BackgroundPresetListData struct {
	Items []BackgroundPresetData `json:"items"`
	Total int64                  `json:"total" example:"5"`
	Page  int                    `json:"page" example:"1"`
	Limit int                    `json:"limit" example:"20"`
}

type BackgroundPresetListResponse struct {
	Code int                    `json:"code" example:"0"`
	Data []BackgroundPresetData `json:"data"`
	Msg  string                 `json:"msg" example:"success"`
}

type AdminBackgroundPresetListResponse struct {
	Code int                        `json:"code" example:"0"`
	Data BackgroundPresetListData   `json:"data"`
	Msg  string                     `json:"msg" example:"success"`
}

type BackgroundPresetDataResponse struct {
	Code int                    `json:"code" example:"0"`
	Data BackgroundPresetData   `json:"data"`
	Msg  string                 `json:"msg" example:"success"`
}

func NewBackgroundPresetData(preset *model.BackgroundPreset) BackgroundPresetData {
	return BackgroundPresetData{
		ID: preset.ID, Key: preset.Key, Name: preset.Name, ImageURL: preset.ImageURL,
		SortOrder: preset.SortOrder, IsActive: preset.IsActive,
		CreatedAt: preset.CreatedAt, UpdatedAt: preset.UpdatedAt,
	}
}

func NewBackgroundPresetList(presets []model.BackgroundPreset) []BackgroundPresetData {
	items := make([]BackgroundPresetData, 0, len(presets))
	for i := range presets {
		items = append(items, NewBackgroundPresetData(&presets[i]))
	}
	return items
}
