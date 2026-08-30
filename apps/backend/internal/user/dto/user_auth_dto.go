package dto

type UserRegisterRequest struct {
	Username         string `json:"username" binding:"required,max=50" example:"koyomi"`
	Email            string `json:"email" binding:"required,email,max=254" example:"user@example.com"`
	Password         string `json:"password" binding:"required,min=8,max=255" example:"password123"`
	ConfirmPassword  string `json:"confirm_password" binding:"required,min=8,max=255" example:"password123"`
	VerificationCode string `json:"verification_code" binding:"required,len=6,numeric" example:"123456"`
}

type UserLoginRequest struct {
	Account  string `json:"account" binding:"required,max=254" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=8,max=255" example:"password123"`
}

type AuthUser struct {
	ID       uint   `json:"id" example:"1"`
	Username string `json:"username" example:"koyomi"`
	Email    string `json:"email" example:"user@example.com"`
	Avatar   string `json:"avatar"`
}

type AuthSession struct {
	Token string   `json:"token"`
	User  AuthUser `json:"user"`
}

type AuthSessionResponse struct {
	Code int         `json:"code" example:"0"`
	Data AuthSession `json:"data"`
	Msg  string      `json:"msg" example:"success"`
}
