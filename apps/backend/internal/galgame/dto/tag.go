package dto

import (
	"time"

	"backend/internal/galgame/model"
)

type CreateTagRequest struct {
	Name        string `json:"name" binding:"required,max=100" example:"纯爱"`
	Slug        string `json:"slug" binding:"required,max=100" example:"pure-love"`
	Description string `json:"description" example:"纯爱题材"`
}

type UpdateTagRequest struct {
	Name        string `json:"name" binding:"required,max=100" example:"纯爱"`
	Slug        string `json:"slug" binding:"required,max=100" example:"pure-love"`
	Description string `json:"description" example:"纯爱题材"`
}

type TagResponse struct {
	ID          uint      `json:"id" example:"1"`
	Name        string    `json:"name" example:"纯爱"`
	Slug        string    `json:"slug" example:"pure-love"`
	Description string    `json:"description" example:"纯爱题材"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewTagResponse(tag *model.Tag) TagResponse {
	return TagResponse{
		ID:          tag.ID,
		Name:        tag.Name,
		Slug:        tag.Slug,
		Description: tag.Description,
		CreatedAt:   tag.CreatedAt,
		UpdatedAt:   tag.UpdatedAt,
	}
}

func NewTagResponses(tags []model.Tag) []TagResponse {
	responses := make([]TagResponse, 0, len(tags))
	for i := range tags {
		responses = append(responses, NewTagResponse(&tags[i]))
	}
	return responses
}

type TagListResponse struct {
	Code int           `json:"code" example:"0"`
	Data []TagResponse `json:"data"`
	Msg  string        `json:"msg" example:"success"`
}

type TagDataResponse struct {
	Code int         `json:"code" example:"0"`
	Data TagResponse `json:"data"`
	Msg  string      `json:"msg" example:"success"`
}
