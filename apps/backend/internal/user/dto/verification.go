package dto

type SendVerificationCodeRequest struct {
	Email   string `json:"email" binding:"required,email,max=254"`
	Purpose string `json:"purpose" binding:"required,oneof=register password_reset"`
}
