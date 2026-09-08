package handler

import (
	"errors"
	"strconv"

	"backend/internal/changelog/dto"
	"backend/internal/changelog/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ChangelogHandler struct {
	service *service.ChangelogService
}

func NewChangelogHandler(value *service.ChangelogService) *ChangelogHandler {
	return &ChangelogHandler{service: value}
}

// List godoc
// @Summary      获取站点更新日志
// @Description  返回全部已发布的站点版本更新日志，按发布时间倒序
// @ID           listSiteChangelogs
// @Tags         changelogs
// @Produce      json
// @Success      200 {object} dto.ChangelogPublicListResponse "更新日志列表"
// @Failure      500 {object} response.ErrorResponse "查询失败"
// @Router       /api/v1/changelogs [get]
func (h *ChangelogHandler) List(c *gin.Context) {
	values, err := h.service.List(c.Request.Context())
	if err != nil {
		h.respondError(c, err, "list site changelogs")
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	response.Ok(c, dto.ChangelogPublicListData{Items: dto.NewChangelogList(values)})
}

// ListAdmin godoc
// @Summary      管理端查询站点更新日志
// @Description  分页返回站点更新日志；需要 changelog:read 权限
// @ID           listAdminSiteChangelogs
// @Tags         admin
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量" default(20)
// @Success      200 {object} dto.ChangelogListResponse "更新日志列表"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/changelogs [get]
func (h *ChangelogHandler) ListAdmin(c *gin.Context) {
	var query dto.AdminChangelogQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	values, total, page, limit, err := h.service.ListAdmin(c.Request.Context(), query.Page, query.Limit)
	if err != nil {
		h.respondError(c, err, "list admin site changelogs")
		return
	}
	response.Ok(c, dto.ChangelogListData{Items: dto.NewChangelogList(values), Total: total, Page: page, Limit: limit})
}

// GetAdmin godoc
// @Summary      管理端查询单条更新日志
// @Description  需要 changelog:read 权限
// @ID           getAdminSiteChangelog
// @Tags         admin
// @Produce      json
// @Param        id path int true "更新日志 ID"
// @Success      200 {object} dto.ChangelogDataResponse "更新日志详情"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "更新日志不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/changelogs/{id} [get]
func (h *ChangelogHandler) GetAdmin(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	value, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "get site changelog")
		return
	}
	response.Ok(c, dto.NewChangelogData(value))
}

// Create godoc
// @Summary      创建更新日志
// @Description  新增一条站点版本更新日志；需要 changelog:create 权限
// @ID           createSiteChangelog
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body dto.ChangelogRequest true "更新日志请求"
// @Success      200 {object} dto.ChangelogDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      409 {object} response.ErrorResponse "版本已存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/changelogs [post]
func (h *ChangelogHandler) Create(c *gin.Context) {
	var req dto.ChangelogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	value, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		h.respondError(c, err, "create site changelog")
		return
	}
	response.Ok(c, dto.NewChangelogData(value))
}

// Update godoc
// @Summary      更新更新日志
// @Description  修改站点版本更新日志的内容；需要 changelog:update 权限
// @ID           updateSiteChangelog
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "更新日志 ID"
// @Param        request body dto.ChangelogRequest true "更新日志请求"
// @Success      200 {object} dto.ChangelogDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "更新日志不存在"
// @Failure      409 {object} response.ErrorResponse "版本已存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/changelogs/{id} [put]
func (h *ChangelogHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.ChangelogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	value, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		h.respondError(c, err, "update site changelog")
		return
	}
	response.Ok(c, dto.NewChangelogData(value))
}

// Delete godoc
// @Summary      删除更新日志
// @Description  删除站点版本更新日志；需要 changelog:delete 权限
// @ID           deleteSiteChangelog
// @Tags         admin
// @Produce      json
// @Param        id path int true "更新日志 ID"
// @Success      200 {object} response.MessageResponse "更新日志已删除"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "更新日志不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/changelogs/{id} [delete]
func (h *ChangelogHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.respondError(c, err, "delete site changelog")
		return
	}
	response.OkWithMsg(c, "更新日志已删除")
}

func (h *ChangelogHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrChangelogNotFound):
		response.Error(c, appErrors.ErrNotFound("更新日志不存在"))
	case errors.Is(err, service.ErrInvalidChangelog):
		response.Error(c, appErrors.ErrValidation("更新日志参数不正确"))
	case errors.Is(err, service.ErrChangelogExists):
		response.Error(c, appErrors.ErrConflict("该版本更新日志已存在"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("更新日志操作失败"))
	}
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("更新日志 ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}
