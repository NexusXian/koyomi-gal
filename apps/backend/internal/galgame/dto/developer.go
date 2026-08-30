package dto

import (
	"time"

	"backend/internal/galgame/model"
)

type CreateDeveloperRequest struct {
	Name         string `json:"name" binding:"required,max=255" example:"YUZUSOFT"`
	OriginalName string `json:"original_name" binding:"max=255" example:"ゆずソフト"`
	Slug         string `json:"slug" binding:"required,max=255" example:"yuzusoft"`
	Description  string `json:"description" example:"游戏开发商"`
	LogoURL      string `json:"logo_url" example:"https://example.com/logo.png"`
	Website      string `json:"website" example:"https://www.yuzu-soft.com"`
}

type UpdateDeveloperRequest struct {
	Name         string `json:"name" binding:"required,max=255" example:"YUZUSOFT"`
	OriginalName string `json:"original_name" binding:"max=255" example:"ゆずソフト"`
	Slug         string `json:"slug" binding:"required,max=255" example:"yuzusoft"`
	Description  string `json:"description" example:"游戏开发商"`
	LogoURL      string `json:"logo_url" example:"https://example.com/logo.png"`
	Website      string `json:"website" example:"https://www.yuzu-soft.com"`
}

type DeveloperResponse struct {
	ID           uint      `json:"id" example:"1"`
	Name         string    `json:"name" example:"YUZUSOFT"`
	OriginalName string    `json:"original_name" example:"ゆずソフト"`
	Slug         string    `json:"slug" example:"yuzusoft"`
	Description  string    `json:"description" example:"游戏开发商"`
	LogoURL      string    `json:"logo_url" example:"https://example.com/logo.png"`
	Website      string    `json:"website" example:"https://www.yuzu-soft.com"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewDeveloperResponse(developer *model.Developer) DeveloperResponse {
	return DeveloperResponse{
		ID:           developer.ID,
		Name:         developer.Name,
		OriginalName: developer.OriginalName,
		Slug:         developer.Slug,
		Description:  developer.Description,
		LogoURL:      developer.LogoURL,
		Website:      developer.Website,
		CreatedAt:    developer.CreatedAt,
		UpdatedAt:    developer.UpdatedAt,
	}
}

func NewDeveloperResponses(developers []model.Developer) []DeveloperResponse {
	responses := make([]DeveloperResponse, 0, len(developers))
	for i := range developers {
		responses = append(responses, NewDeveloperResponse(&developers[i]))
	}
	return responses
}

type DeveloperListResponse struct {
	Code int                 `json:"code" example:"0"`
	Data []DeveloperResponse `json:"data"`
	Msg  string              `json:"msg" example:"success"`
}

type DeveloperDataResponse struct {
	Code int               `json:"code" example:"0"`
	Data DeveloperResponse `json:"data"`
	Msg  string            `json:"msg" example:"success"`
}
