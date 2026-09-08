package dto

type UserRegisterRequest struct {
	Username         string `json:"username" binding:"required,max=50" example:"koyomi"`
	Email            string `json:"email" binding:"required,email,max=254" example:"user@example.com"`
	Password         string `json:"password" binding:"required,min=8,max=72" example:"password123"`
	ConfirmPassword  string `json:"confirm_password" binding:"required,min=8,max=72" example:"password123"`
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

type PasswordForgotCodeRequest struct {
	Email string `json:"email" binding:"required,email,max=254" example:"user@example.com"`
}

type PasswordForgotVerifyRequest struct {
	Email string `json:"email" binding:"required,email,max=254" example:"user@example.com"`
	Code  string `json:"code" binding:"required,len=6,numeric" example:"123456"`
}

type PasswordForgotResetRequest struct {
	ResetToken      string `json:"reset_token" binding:"required"`
	Password        string `json:"password" binding:"required" example:"newpassword123"`
	ConfirmPassword string `json:"confirm_password" binding:"required" example:"newpassword123"`
}

type ChangePasswordRequest struct {
	Code            string `json:"code" binding:"required,len=6,numeric" example:"123456"`
	NewPassword     string `json:"new_password" binding:"required" example:"newpassword123"`
	ConfirmPassword string `json:"confirm_password" binding:"required" example:"newpassword123"`
}

type PasswordResetTokenData struct {
	ResetToken string `json:"reset_token"`
}

type PasswordResetTokenResponse struct {
	Code int                    `json:"code" example:"0"`
	Data PasswordResetTokenData `json:"data"`
	Msg  string                 `json:"msg" example:"success"`
}

type PasswordCodeData struct {
	Email string `json:"email" example:"us***@example.com"`
}

type PasswordCodeResponse struct {
	Code int              `json:"code" example:"0"`
	Data PasswordCodeData `json:"data"`
	Msg  string           `json:"msg" example:"验证码已发送"`
}
