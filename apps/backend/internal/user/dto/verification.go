package dto

type SendVerificationCodeRequest struct {
	Email   string `json:"email" binding:"required,email,max=254" example:"user@example.com"`
	Purpose string `json:"purpose" binding:"required,eq=register" example:"register"`
}
