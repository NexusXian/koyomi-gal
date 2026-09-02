package handler

import (
	"errors"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/notification/dto"
	"backend/internal/notification/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type NotificationHandler struct {
	notificationService *service.NotificationService
}

func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

// ListNotifications godoc
// @Summary      查询通知
// @Description  分页返回当前用户的站内通知
// @ID           listNotifications
// @Tags         notifications
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Param        category query string false "通知分类" Enums(interaction,review,moderation,system)
// @Param        unread query bool false "是否未读；false 仅返回已读通知"
// @Success      200 {object} dto.NotificationListResponse "通知列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      500 {object} response.ErrorResponse "查询通知失败"
// @Security     BearerAuth
// @Router       /api/v1/notifications [get]
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	var query dto.ListNotificationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	notifications, total, page, limit, err := h.notificationService.List(
		c.Request.Context(), userID, query.Page, query.Limit, query.Category, query.Unread,
	)
	if err != nil {
		h.respondError(c, err, "list notifications")
		return
	}
	response.Ok(c, dto.NotificationListData{
		Items: dto.NewNotificationList(notifications), Total: total, Page: page, Limit: limit,
	})
}

// UnreadCount godoc
// @Summary      查询未读通知数
// @Description  返回当前用户的未读通知数量
// @ID           getNotificationUnreadCount
// @Tags         notifications
// @Produce      json
// @Success      200 {object} dto.UnreadCountResponse "未读通知数量"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      500 {object} response.ErrorResponse "查询未读通知数失败"
// @Security     BearerAuth
// @Router       /api/v1/notifications/unread-count [get]
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	count, err := h.notificationService.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err, "count unread notifications")
		return
	}
	response.Ok(c, dto.UnreadCountData{Count: count})
}

// MarkRead godoc
// @Summary      标记通知已读
// @Description  将当前用户的一条通知标记为已读
// @ID           markNotificationRead
// @Tags         notifications
// @Produce      json
// @Param        id path int true "通知 ID"
// @Success      200 {object} response.MessageResponse "通知已标记为已读"
// @Failure      400 {object} response.ErrorResponse "通知 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "通知不存在"
// @Failure      500 {object} response.ErrorResponse "标记通知失败"
// @Security     BearerAuth
// @Router       /api/v1/notifications/{id}/read [patch]
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("通知 ID 格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if _, err := h.notificationService.MarkRead(c.Request.Context(), userID, uint(id)); err != nil {
		h.respondError(c, err, "mark notification read")
		return
	}
	response.OkWithMsg(c, "通知已标记为已读")
}

// MarkAllRead godoc
// @Summary      全部标记已读
// @Description  将当前用户的全部未读通知标记为已读
// @ID           markAllNotificationsRead
// @Tags         notifications
// @Produce      json
// @Success      200 {object} response.MessageResponse "通知已全部标记为已读"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      500 {object} response.ErrorResponse "标记通知失败"
// @Security     BearerAuth
// @Router       /api/v1/notifications/read-all [patch]
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if _, err := h.notificationService.MarkAllRead(c.Request.Context(), userID); err != nil {
		h.respondError(c, err, "mark all notifications read")
		return
	}
	response.OkWithMsg(c, "通知已全部标记为已读")
}

func (h *NotificationHandler) respondError(c *gin.Context, err error, operation string) {
	if errors.Is(err, service.ErrNotificationNotFound) {
		response.Error(c, appErrors.ErrNotFound("通知不存在"))
		return
	}
	logger.Error(operation, zap.Error(err))
	response.Error(c, appErrors.ErrInternal("通知操作失败"))
}
