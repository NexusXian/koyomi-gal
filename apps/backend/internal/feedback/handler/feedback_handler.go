package handler

import (
	"errors"
	"strconv"

	"backend/internal/feedback/dto"
	"backend/internal/feedback/repository"
	"backend/internal/feedback/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FeedbackHandler struct {
	feedbackService *service.FeedbackService
}

func NewFeedbackHandler(feedbackService *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{feedbackService: feedbackService}
}

// CreateFeedback godoc
// @Summary      提交意见反馈或版权投诉
// @Description  匿名提交，按 IP 每小时限 5 次
// @ID           createFeedback
// @Tags         feedback
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateFeedbackRequest true "提交反馈请求"
// @Success      200 {object} response.MessageResponse "提交成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      429 {object} response.ErrorResponse "提交过于频繁"
// @Failure      500 {object} response.ErrorResponse "提交失败"
// @Router       /api/v1/feedback [post]
func (h *FeedbackHandler) CreateFeedback(c *gin.Context) {
	var req dto.CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("反馈内容格式不正确"))
		return
	}
	userAgent := c.Request.UserAgent()
	if len(userAgent) > 512 {
		userAgent = userAgent[:512]
	}
	if _, err := h.feedbackService.Submit(c.Request.Context(), &req, c.ClientIP(), userAgent); err != nil {
		switch {
		case errors.Is(err, service.ErrFeedbackRateLimit):
			response.Error(c, appErrors.ErrTooManyRequests("提交过于频繁，请稍后重试"))
		case errors.Is(err, service.ErrInvalidFeedback):
			response.Error(c, appErrors.ErrValidation("反馈内容至少 5 个字符"))
		default:
			logger.Error("create feedback", zap.Error(err))
			response.Error(c, appErrors.ErrInternal("反馈提交失败"))
		}
		return
	}
	response.OkWithMsg(c, "提交成功，感谢你的反馈")
}

// ListAdminFeedback godoc
// @Summary      管理端查询反馈
// @Description  分页返回意见反馈与版权投诉；需要 feedback:read 权限
// @ID           listAdminFeedback
// @Tags         admin
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Param        type query string false "类型" Enums(feedback,copyright)
// @Param        handled query bool false "是否已处理"
// @Success      200 {object} dto.AdminFeedbackListResponse "反馈列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询反馈失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/feedback [get]
func (h *FeedbackHandler) ListAdminFeedback(c *gin.Context) {
	var query dto.AdminFeedbackQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	filter := repository.FeedbackFilter{Type: query.Type, Handled: query.Handled}
	items, total, page, limit, err := h.feedbackService.ListAdmin(c.Request.Context(), query.Page, query.Limit, filter)
	if err != nil {
		h.respondError(c, err, "list admin feedback")
		return
	}
	response.Ok(c, dto.FeedbackListData{Items: dto.NewFeedbackList(items), Total: total, Page: page, Limit: limit})
}

// HandleFeedback godoc
// @Summary      处理反馈
// @Description  标记反馈为已处理或待处理；需要 feedback:handle 权限
// @ID           handleFeedback
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "反馈 ID"
// @Param        request body dto.HandleFeedbackRequest true "处理反馈请求"
// @Success      200 {object} dto.FeedbackDataResponse "处理成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      404 {object} response.ErrorResponse "反馈不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/feedback/{id}/handle [put]
func (h *FeedbackHandler) HandleFeedback(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("反馈 ID 格式不正确"))
		return
	}
	var req dto.HandleFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	adminID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	feedback, err := h.feedbackService.Handle(c.Request.Context(), uint(id), req.Handled, adminID)
	if err != nil {
		h.respondError(c, err, "handle feedback")
		return
	}
	response.Ok(c, dto.NewFeedbackData(feedback))
}

func (h *FeedbackHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrFeedbackNotFound):
		response.Error(c, appErrors.ErrNotFound("反馈不存在"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("反馈操作失败"))
	}
}
