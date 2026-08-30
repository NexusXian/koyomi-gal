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

// SendCode godoc
// @Summary      发送邮箱验证码
// @Description  创建验证码邮件任务，202 表示任务已入队，不代表邮件已送达
// @ID           sendVerificationCode
// @Tags         verification
// @Accept       json
// @Produce      json
// @Param        request body dto.SendVerificationCodeRequest true "发送验证码请求"
// @Success      202 {object} response.MessageResponse "验证码发送任务已创建"
// @Failure      400 {object} response.ErrorResponse "邮箱或验证码用途格式不正确"
// @Failure      429 {object} response.ErrorResponse "请求过于频繁"
// @Failure      500 {object} response.ErrorResponse "验证码发送任务创建失败"
// @Router       /api/v1/auth/verification-codes [post]
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
