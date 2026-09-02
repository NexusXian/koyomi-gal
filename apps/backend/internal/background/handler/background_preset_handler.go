package handler

import (
	"errors"
	"strconv"

	"backend/internal/background/dto"
	"backend/internal/background/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type BackgroundPresetHandler struct {
	presetService *service.BackgroundPresetService
}

func NewBackgroundPresetHandler(presetService *service.BackgroundPresetService) *BackgroundPresetHandler {
	return &BackgroundPresetHandler{presetService: presetService}
}

// ListBackgroundPresets godoc
// @Summary      查看启用的背景预设
// @Description  返回当前启用的背景预设列表，按排序值倒序
// @ID           listBackgroundPresets
// @Tags         backgrounds
// @Produce      json
// @Success      200 {object} dto.BackgroundPresetListResponse "背景预设列表"
// @Failure      500 {object} response.ErrorResponse "查询背景预设失败"
// @Router       /api/v1/background-presets [get]
func (h *BackgroundPresetHandler) ListBackgroundPresets(c *gin.Context) {
	presets, err := h.presetService.ListPublic(c.Request.Context())
	if err != nil {
		h.respondError(c, err, "list background presets")
		return
	}
	response.Ok(c, dto.NewBackgroundPresetList(presets))
}

// ListAdminBackgroundPresets godoc
// @Summary      管理端查询背景预设
// @Description  分页返回全部背景预设；需要 background_preset:read 权限
// @ID           listAdminBackgroundPresets
// @Tags         admin
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.AdminBackgroundPresetListResponse "背景预设列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询背景预设失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/background-presets [get]
func (h *BackgroundPresetHandler) ListAdminBackgroundPresets(c *gin.Context) {
	var query dto.AdminBackgroundPresetQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	presets, total, page, limit, err := h.presetService.ListAdmin(c.Request.Context(), query.Page, query.Limit)
	if err != nil {
		h.respondError(c, err, "list admin background presets")
		return
	}
	response.Ok(c, dto.BackgroundPresetListData{Items: dto.NewBackgroundPresetList(presets), Total: total, Page: page, Limit: limit})
}

// CreateAdminBackgroundPreset godoc
// @Summary      创建背景预设
// @Description  创建背景预设；需要 background_preset:create 权限
// @ID           createAdminBackgroundPreset
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateBackgroundPresetRequest true "创建背景预设请求"
// @Success      200 {object} dto.BackgroundPresetDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Security     BearerAuth
// @Router       /api/v1/admin/background-presets [post]
func (h *BackgroundPresetHandler) CreateAdminBackgroundPreset(c *gin.Context) {
	var req dto.CreateBackgroundPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	preset, err := h.presetService.Create(c.Request.Context(), &req)
	if err != nil {
		h.respondError(c, err, "create background preset")
		return
	}
	response.Ok(c, dto.NewBackgroundPresetData(preset))
}

// UpdateAdminBackgroundPreset godoc
// @Summary      更新背景预设
// @Description  全量更新背景预设；需要 background_preset:update 权限
// @ID           updateAdminBackgroundPreset
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "背景预设 ID"
// @Param        request body dto.UpdateBackgroundPresetRequest true "更新背景预设请求"
// @Success      200 {object} dto.BackgroundPresetDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      404 {object} response.ErrorResponse "背景预设不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/background-presets/{id} [put]
func (h *BackgroundPresetHandler) UpdateAdminBackgroundPreset(c *gin.Context) {
	id, ok := parsePresetID(c)
	if !ok {
		return
	}
	var req dto.UpdateBackgroundPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	preset, err := h.presetService.Update(c.Request.Context(), id, &req)
	if err != nil {
		h.respondError(c, err, "update background preset")
		return
	}
	response.Ok(c, dto.NewBackgroundPresetData(preset))
}

// DeleteAdminBackgroundPreset godoc
// @Summary      删除背景预设
// @Description  删除背景预设；需要 background_preset:delete 权限
// @ID           deleteAdminBackgroundPreset
// @Tags         admin
// @Produce      json
// @Param        id path int true "背景预设 ID"
// @Success      200 {object} response.MessageResponse "背景预设已删除"
// @Failure      400 {object} response.ErrorResponse "背景预设 ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "背景预设不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/background-presets/{id} [delete]
func (h *BackgroundPresetHandler) DeleteAdminBackgroundPreset(c *gin.Context) {
	id, ok := parsePresetID(c)
	if !ok {
		return
	}
	if err := h.presetService.Delete(c.Request.Context(), id); err != nil {
		h.respondError(c, err, "delete background preset")
		return
	}
	response.OkWithMsg(c, "背景预设已删除")
}

func (h *BackgroundPresetHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrBackgroundPresetNotFound):
		response.Error(c, appErrors.ErrNotFound("背景预设不存在"))
	case errors.Is(err, service.ErrInvalidPresetInput):
		response.Error(c, appErrors.ErrValidation("背景预设名称和图片地址不能为空"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("背景预设操作失败"))
	}
}

func parsePresetID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("背景预设 ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}
