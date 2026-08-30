package dto

import (
	"time"

	"backend/internal/rbac/model"
)

type CreatePermissionRequest struct {
	Name        string `json:"name" binding:"required,max=64" example:"删除用户"`
	Code        string `json:"code" binding:"required,max=64" example:"user:delete"`
	Description string `json:"description" binding:"max=255" example:"允许删除用户"`
}

type UpdatePermissionRequest struct {
	Name        string `json:"name" binding:"required,max=64" example:"删除用户"`
	Description string `json:"description" binding:"max=255" example:"允许删除用户"`
}

type PermissionResponse struct {
	ID          int64     `json:"id" example:"1"`
	Name        string    `json:"name" example:"删除用户"`
	Code        string    `json:"code" example:"user:delete"`
	Resource    string    `json:"resource" example:"user"`
	Action      string    `json:"action" example:"delete"`
	Description string    `json:"description" example:"允许删除用户"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewPermissionResponse(permission *model.Permission) PermissionResponse {
	return PermissionResponse{
		ID:          permission.ID,
		Name:        permission.Name,
		Code:        permission.Code,
		Resource:    permission.Resource,
		Action:      permission.Action,
		Description: permission.Description,
		CreatedAt:   permission.CreatedAt,
		UpdatedAt:   permission.UpdatedAt,
	}
}

func NewPermissionResponses(permissions []model.Permission) []PermissionResponse {
	responses := make([]PermissionResponse, 0, len(permissions))
	for i := range permissions {
		responses = append(responses, NewPermissionResponse(&permissions[i]))
	}
	return responses
}

type PermissionListResponse struct {
	Code int                  `json:"code" example:"0"`
	Data []PermissionResponse `json:"data"`
	Msg  string               `json:"msg" example:"success"`
}
