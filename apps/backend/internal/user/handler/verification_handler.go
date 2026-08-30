package handler

import (
	"errors"

	"backend/internal/user/dto"
	"backend/internal/user/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type VerificationHandler struct {
	verificationService *service.VerificationService
}

func NewVerificationHandler(verificationService *service.VerificationService) *VerificationHandler {
	return &VerificationHandler{verificationService: verificationService}
}

func (h *VerificationHandler) SendCode(c *gin.Context) {
	var request dto.SendVerificationCodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, appErrors.ErrValidation("邮箱或验证码用途格式不正确"))
		return
	}

	err := h.verificationService.SendCode(
		c.Request.Context(),
		request.Email,
		request.Purpose,
		c.ClientIP(),
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidVerification):
			response.Error(c, appErrors.ErrValidation("邮箱或验证码用途格式不正确"))
		case errors.Is(err, service.ErrVerificationCooldown):
			response.Error(c, appErrors.ErrTooManyRequests("验证码发送过于频繁，请稍后重试"))
		case errors.Is(err, service.ErrVerificationRateLimit):
			response.Error(c, appErrors.ErrTooManyRequests("请求过于频繁，请稍后重试"))
		default:
			logger.Error("create verification email task", zap.Error(err))
			response.Error(c, appErrors.ErrInternal("验证码发送任务创建失败"))
		}
		return
	}

	response.AcceptedWithMsg(c, "验证码发送任务已创建")
}
