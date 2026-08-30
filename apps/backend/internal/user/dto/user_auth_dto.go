package dto

type UserRegisterRequest struct {
	Username         string `json:"username" binding:"required,max=50"`
	Email            string `json:"email" binding:"required,email,max=254"`
	Password         string `json:"password" binding:"required,min=8,max=255"`
	ConfirmPassword  string `json:"confirm_password" binding:"required,min=8,max=255"`
	VerificationCode string `json:"verification_code" binding:"required"`
}

type UserLoginRequest struct {
	Account  string `json:"account" binding:"required,max=254"`
	Password string `json:"password" binding:"required,min=8,max=255"`
}
