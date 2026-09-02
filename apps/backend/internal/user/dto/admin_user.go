package dto

import (
	"time"

	"backend/internal/user/model"
)

type AdminUserQuery struct {
	Keyword string `form:"keyword" binding:"omitempty,max=255"`
	Page    int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit   int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

type CreateAdminUserRequest struct {
	Username string `json:"username" binding:"required,max=50" example:"koyomi"`
	Email    string `json:"email" binding:"required,email,max=254" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=8,max=255" example:"password123"`
	IsBanned *bool  `json:"is_banned" example:"false"`
}

type UpdateAdminUserRequest struct {
	Username *string `json:"username" binding:"omitempty,max=50" example:"koyomi"`
	Email    *string `json:"email" binding:"omitempty,email,max=254" example:"user@example.com"`
	Password *string `json:"password" binding:"omitempty,min=8,max=255" example:"new-password123"`
	IsBanned *bool   `json:"is_banned" example:"false"`
}

type AdminUserData struct {
	ID        uint      `json:"id" example:"1"`
	Username  string    `json:"username" example:"koyomi"`
	Email     string    `json:"email" example:"user@example.com"`
	IsBanned  bool      `json:"is_banned" example:"false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminUserListData struct {
	Items []AdminUserData `json:"items"`
	Total int64           `json:"total" example:"100"`
	Page  int             `json:"page" example:"1"`
	Limit int             `json:"limit" example:"20"`
}

type AdminUserListResponse struct {
	Code int               `json:"code" example:"0"`
	Data AdminUserListData `json:"data"`
	Msg  string            `json:"msg" example:"success"`
}

type AdminUserDataResponse struct {
	Code int           `json:"code" example:"0"`
	Data AdminUserData `json:"data"`
	Msg  string        `json:"msg" example:"success"`
}

func NewAdminUserData(user *model.User) AdminUserData {
	return AdminUserData{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		IsBanned:  user.IsBanned,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func NewAdminUserList(users []model.User) []AdminUserData {
	items := make([]AdminUserData, 0, len(users))
	for i := range users {
		items = append(items, NewAdminUserData(&users[i]))
	}
	return items
}
