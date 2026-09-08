package handler

import (
	"errors"
	"strconv"

	"backend/internal/message/dto"
	"backend/internal/message/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type MessageHandler struct {
	messages *service.MessageService
}

func NewMessageHandler(messages *service.MessageService) *MessageHandler {
	return &MessageHandler{messages: messages}
}

// CreateConversation godoc
// @Summary      获取或创建私信会话
// @Description  获取当前用户与目标用户的私信会话，不存在时创建
// @ID           createMessageConversation
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateConversationRequest true "目标用户"
// @Success      200 {object} dto.ConversationResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      403 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Failure      429 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Failure      503 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/messages/conversations [post]
func (h *MessageHandler) CreateConversation(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	var req dto.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("私信会话参数格式不正确"))
		return
	}
	data, err := h.messages.CreateConversation(c.Request.Context(), actorID, &req)
	if err != nil {
		h.respondError(c, err, "create message conversation")
		return
	}
	response.Ok(c, data)
}

// ListConversations godoc
// @Summary      查询私信会话
// @Description  按最后消息时间和会话 ID 稳定分页返回当前用户的会话
// @ID           listMessageConversations
// @Tags         messages
// @Produce      json
// @Param        cursor query string false "不透明游标"
// @Param        limit query int false "每页数量" default(20)
// @Success      200 {object} dto.ConversationListResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/messages/conversations [get]
func (h *MessageHandler) ListConversations(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	var query dto.ConversationListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("会话查询参数格式不正确"))
		return
	}
	data, err := h.messages.ListConversations(c.Request.Context(), actorID, query.Cursor, query.Limit)
	if err != nil {
		h.respondError(c, err, "list message conversations")
		return
	}
	response.Ok(c, data)
}

// ListMessages godoc
// @Summary      查询会话消息
// @Description  返回指定会话消息，数据库倒序查询后按从旧到新输出
// @ID           listConversationMessages
// @Tags         messages
// @Produce      json
// @Param        id path int true "会话 ID"
// @Param        before_id query int false "上一页最早消息 ID"
// @Param        limit query int false "每页数量" default(30)
// @Success      200 {object} dto.MessageListResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/messages/conversations/{id}/messages [get]
func (h *MessageHandler) ListMessages(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	conversationID, ok := pathID(c, "id", "会话 ID 格式不正确")
	if !ok {
		return
	}
	var query dto.MessageListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("消息查询参数格式不正确"))
		return
	}
	data, err := h.messages.ListMessages(
		c.Request.Context(), actorID, conversationID, query.BeforeID, query.Limit,
	)
	if err != nil {
		h.respondError(c, err, "list conversation messages")
		return
	}
	response.Ok(c, data)
}

// SendMessage godoc
// @Summary      发送私信
// @Description  在指定私信会话中发送文本消息
// @ID           sendConversationMessage
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        id path int true "会话 ID"
// @Param        request body dto.SendMessageRequest true "消息内容"
// @Success      200 {object} dto.MessageResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      403 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Failure      429 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Failure      503 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/messages/conversations/{id}/messages [post]
func (h *MessageHandler) SendMessage(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	conversationID, ok := pathID(c, "id", "会话 ID 格式不正确")
	if !ok {
		return
	}
	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("消息参数格式不正确"))
		return
	}
	data, err := h.messages.SendMessage(c.Request.Context(), actorID, conversationID, &req)
	if err != nil {
		h.respondError(c, err, "send message")
		return
	}
	response.Ok(c, data)
}

// MarkRead godoc
// @Summary      标记会话已读
// @Description  将会话读取进度推进到指定消息，读取进度不会倒退
// @ID           markConversationRead
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        id path int true "会话 ID"
// @Param        request body dto.MarkReadRequest true "读取到的消息"
// @Success      200 {object} response.MessageResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/messages/conversations/{id}/read [post]
func (h *MessageHandler) MarkRead(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	conversationID, ok := pathID(c, "id", "会话 ID 格式不正确")
	if !ok {
		return
	}
	var req dto.MarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("已读参数格式不正确"))
		return
	}
	if err := h.messages.MarkRead(c.Request.Context(), actorID, conversationID, req.MessageID); err != nil {
		h.respondError(c, err, "mark conversation read")
		return
	}
	response.OkWithMsg(c, "会话已标记为已读")
}

// DeleteConversation godoc
// @Summary      删除私信会话
// @Description  仅隐藏当前用户的会话视图，不删除对方视图或历史消息
// @ID           deleteMessageConversation
// @Tags         messages
// @Produce      json
// @Param        id path int true "会话 ID"
// @Success      200 {object} response.MessageResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/messages/conversations/{id} [delete]
func (h *MessageHandler) DeleteConversation(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	conversationID, ok := pathID(c, "id", "会话 ID 格式不正确")
	if !ok {
		return
	}
	if err := h.messages.DeleteConversation(c.Request.Context(), actorID, conversationID); err != nil {
		h.respondError(c, err, "delete message conversation")
		return
	}
	response.OkWithMsg(c, "会话已删除")
}

// DeleteMessage godoc
// @Summary      删除私信消息
// @Description  仅允许发送者软删除自己的消息
// @ID           deleteMessage
// @Tags         messages
// @Produce      json
// @Param        message_id path int true "消息 ID"
// @Success      200 {object} response.MessageResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      403 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/messages/{message_id} [delete]
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	messageID, ok := pathID(c, "message_id", "消息 ID 格式不正确")
	if !ok {
		return
	}
	if err := h.messages.DeleteMessage(c.Request.Context(), actorID, messageID); err != nil {
		h.respondError(c, err, "delete message")
		return
	}
	response.OkWithMsg(c, "消息已删除")
}

// UnreadCount godoc
// @Summary      查询未读私信数
// @ID           getMessageUnreadCount
// @Tags         messages
// @Produce      json
// @Success      200 {object} dto.MessageUnreadCountResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/messages/unread-count [get]
func (h *MessageHandler) UnreadCount(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	count, err := h.messages.UnreadCount(c.Request.Context(), actorID)
	if err != nil {
		h.respondError(c, err, "count unread messages")
		return
	}
	response.Ok(c, dto.UnreadCountData{Count: count})
}

// GetSettings godoc
// @Summary      查询私信设置
// @ID           getMessageSettings
// @Tags         messages
// @Produce      json
// @Success      200 {object} dto.MessageSettingsResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/me/message-settings [get]
func (h *MessageHandler) GetSettings(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	data, err := h.messages.GetSettings(c.Request.Context(), actorID)
	if err != nil {
		h.respondError(c, err, "get message settings")
		return
	}
	response.Ok(c, data)
}

// UpdateSettings godoc
// @Summary      更新私信设置
// @ID           updateMessageSettings
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        request body dto.UpdateMessageSettingsRequest true "私信设置"
// @Success      200 {object} dto.MessageSettingsResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/me/message-settings [put]
func (h *MessageHandler) UpdateSettings(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	var req dto.UpdateMessageSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("私信设置参数格式不正确"))
		return
	}
	data, err := h.messages.UpdateSettings(c.Request.Context(), actorID, &req)
	if err != nil {
		h.respondError(c, err, "update message settings")
		return
	}
	response.Ok(c, data)
}

// BlockUser godoc
// @Summary      屏蔽用户私信
// @ID           blockMessageUser
// @Tags         messages
// @Produce      json
// @Param        id path int true "用户 ID"
// @Success      200 {object} response.MessageResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/block [post]
func (h *MessageHandler) BlockUser(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	blockedID, ok := pathID(c, "id", "用户 ID 格式不正确")
	if !ok {
		return
	}
	if err := h.messages.BlockUser(c.Request.Context(), actorID, blockedID); err != nil {
		h.respondError(c, err, "block message user")
		return
	}
	response.OkWithMsg(c, "用户已屏蔽")
}

// UnblockUser godoc
// @Summary      取消屏蔽用户私信
// @ID           unblockMessageUser
// @Tags         messages
// @Produce      json
// @Param        id path int true "用户 ID"
// @Success      200 {object} response.MessageResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/block [delete]
func (h *MessageHandler) UnblockUser(c *gin.Context) {
	actorID, ok := currentUser(c)
	if !ok {
		return
	}
	blockedID, ok := pathID(c, "id", "用户 ID 格式不正确")
	if !ok {
		return
	}
	if err := h.messages.UnblockUser(c.Request.Context(), actorID, blockedID); err != nil {
		h.respondError(c, err, "unblock message user")
		return
	}
	response.OkWithMsg(c, "用户屏蔽已取消")
}

func currentUser(c *gin.Context) (uint, bool) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
	}
	return userID, ok
}

func pathID(c *gin.Context, name, message string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation(message))
		return 0, false
	}
	return uint(id), true
}

func (h *MessageHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrMessageEmpty):
		response.Error(c, appErrors.ErrValidation("消息内容不能为空"))
	case errors.Is(err, service.ErrMessageTooLong):
		response.Error(c, appErrors.ErrValidation("消息内容不能超过 2000 个字符"))
	case errors.Is(err, service.ErrInvalidMessageType):
		response.Error(c, appErrors.ErrValidation("消息类型不支持"))
	case errors.Is(err, service.ErrInvalidMessagePermission):
		response.Error(c, appErrors.ErrValidation("私信权限设置无效"))
	case errors.Is(err, service.ErrInvalidMessage), errors.Is(err, service.ErrInvalidUser):
		response.Error(c, appErrors.ErrValidation("私信参数格式不正确"))
	case errors.Is(err, service.ErrInvalidCursor):
		response.Error(c, appErrors.ErrValidation("会话游标格式不正确"))
	case errors.Is(err, service.ErrCannotMessageSelf):
		response.Error(c, appErrors.ErrBadRequest("不能向自己发起私信操作"))
	case errors.Is(err, service.ErrUserNotFound):
		response.Error(c, appErrors.ErrNotFound("用户不存在"))
	case errors.Is(err, service.ErrConversationNotFound):
		response.Error(c, appErrors.ErrNotFound("会话不存在"))
	case errors.Is(err, service.ErrConversationAccessDenied):
		// Conceal conversation membership to avoid exposing private conversation IDs.
		response.Error(c, appErrors.ErrNotFound("会话不存在"))
	case errors.Is(err, service.ErrMessageNotFound):
		response.Error(c, appErrors.ErrNotFound("消息不存在"))
	case errors.Is(err, service.ErrMessagePermissionDenied):
		response.Error(c, appErrors.ErrForbidden("MessagePermissionDenied"))
	case errors.Is(err, service.ErrUserBlocked):
		response.Error(c, appErrors.ErrForbidden("UserBlocked"))
	case errors.Is(err, service.ErrMessageAccessDenied):
		response.Error(c, appErrors.ErrForbidden("MessageAccessDenied"))
	case errors.Is(err, service.ErrMessageRateLimited):
		response.Error(c, appErrors.ErrTooManyRequests("消息发送过于频繁"))
	case errors.Is(err, service.ErrConversationRateLimit):
		response.Error(c, appErrors.ErrTooManyRequests("新建会话过于频繁"))
	case errors.Is(err, service.ErrRateLimitUnavailable):
		response.Error(c, appErrors.New(appErrors.CodeBiz, "私信服务暂时不可用", 503))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("私信操作失败"))
	}
}
