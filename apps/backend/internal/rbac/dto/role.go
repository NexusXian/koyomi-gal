package dto

import (
	"time"

	"backend/internal/rbac/model"
)

type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required,max=64" example:"管理员"`
	Code        string `json:"code" binding:"required,max=64" example:"admin"`
	Description string `json:"description" binding:"max=255" example:"管理员角色"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name" binding:"required,max=64" example:"管理员"`
	Description string `json:"description" binding:"max=255" example:"管理员角色"`
}

type RoleResponse struct {
	ID          int64     `json:"id" example:"1"`
	Name        string    `json:"name" example:"管理员"`
	Code        string    `json:"code" example:"admin"`
	Description string    `json:"description" example:"管理员角色"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewRoleResponse(role *model.Role) RoleResponse {
	return RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Code:        role.Code,
		Description: role.Description,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

func NewRoleResponses(roles []model.Role) []RoleResponse {
	responses := make([]RoleResponse, 0, len(roles))
	for i := range roles {
		responses = append(responses, NewRoleResponse(&roles[i]))
	}
	return responses
}

type RoleListResponse struct {
	Code int            `json:"code" example:"0"`
	Data []RoleResponse `json:"data"`
	Msg  string         `json:"msg" example:"success"`
}
