package ipgeo

import (
	"strconv"

	appErrors "backend/pkg/errors"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type LogQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type UserIPLogListData struct {
	Items []UserIPLog `json:"items"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

type UserIPLogListResponse struct {
	Code int               `json:"code"`
	Data UserIPLogListData `json:"data"`
	Msg  string            `json:"msg"`
}

type Handler struct {
	service *AuditService
}

func NewHandler(service *AuditService) *Handler {
	return &Handler{service: service}
}

// ListUserIPLogs godoc
// @Summary      查询用户 IP 历史
// @Description  返回用户发帖和评论时记录的完整 IP 与属地；需要 ip_audit:read 权限
// @ID           listUserIPLogs
// @Tags         admin
// @Produce      json
// @Param        id path int true "用户 ID"
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} UserIPLogListResponse "IP 历史"
// @Failure      400 {object} response.ErrorResponse "参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询 IP 历史失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/users/{id}/ip-logs [get]
func (h *Handler) ListUserIPLogs(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || userID == 0 {
		response.Error(c, appErrors.ErrValidation("用户 ID 格式不正确"))
		return
	}
	var query LogQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	items, total, page, limit, err := h.service.ListByUser(c.Request.Context(), uint(userID), query.Page, query.Limit)
	if err != nil {
		response.Error(c, appErrors.ErrInternal("查询 IP 历史失败"))
		return
	}
	response.Ok(c, UserIPLogListData{Items: items, Total: total, Page: page, Limit: limit})
}
