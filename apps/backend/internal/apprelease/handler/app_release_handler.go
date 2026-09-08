package handler

import (
	"errors"
	"io"
	"strconv"

	"backend/internal/apprelease/dto"
	"backend/internal/apprelease/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AppReleaseHandler struct {
	service *service.AppReleaseService
}

func NewAppReleaseHandler(value *service.AppReleaseService) *AppReleaseHandler {
	return &AppReleaseHandler{service: value}
}

// Latest godoc
// @Summary      Check for an app update
// @Description  Returns the greatest currently published version newer than versionCode
// @ID           getLatestAppRelease
// @Tags         app-releases
// @Produce      json
// @Param        platform query string true "Client platform" Enums(android,ios,windows,macos,linux)
// @Param        versionCode query int true "Numeric client version code"
// @Param        versionName query string false "Display-only client version"
// @Success      200 {object} dto.LatestReleaseResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /api/v1/app/releases/latest [get]
func (h *AppReleaseHandler) Latest(c *gin.Context) {
	var query dto.LatestReleaseQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("版本查询参数不正确"))
		return
	}
	value, err := h.service.Latest(c.Request.Context(), query.Platform, *query.VersionCode)
	if err != nil {
		h.respondError(c, err, "get latest app release")
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	if value == nil {
		response.Ok(c, dto.NoUpdateData{HasUpdate: false})
		return
	}
	response.Ok(c, dto.NewUpdateData(value, *query.VersionCode))
}

// ListAdmin godoc
// @Summary      List app releases for administration
// @Description  Requires app_release:read
// @ID           listAdminAppReleases
// @Tags         admin
// @Produce      json
// @Param        page query int false "Page" default(1)
// @Param        limit query int false "Page size" default(20)
// @Success      200 {object} dto.AdminReleaseListResponse
// @Failure      400 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/app/releases [get]
func (h *AppReleaseHandler) ListAdmin(c *gin.Context) {
	var query dto.AdminReleaseQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	values, total, page, limit, err := h.service.ListAdmin(c.Request.Context(), query.Page, query.Limit)
	if err != nil {
		h.respondError(c, err, "list app releases")
		return
	}
	response.Ok(c, dto.AppReleaseListData{Items: dto.NewAppReleaseList(values), Total: total, Page: page, Limit: limit})
}

// GetAdmin godoc
// @Summary      Get an app release
// @Description  Requires app_release:read
// @ID           getAdminAppRelease
// @Tags         admin
// @Produce      json
// @Param        id path int true "App release ID"
// @Success      200 {object} dto.AppReleaseDataResponse
// @Failure      404 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/app/releases/{id} [get]
func (h *AppReleaseHandler) GetAdmin(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	value, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "get app release")
		return
	}
	response.Ok(c, dto.NewAppReleaseData(value))
}

// CreateAdmin godoc
// @Summary      Create an app release
// @Description  Requires app_release:create; publishing also requires app_release:publish
// @ID           createAdminAppRelease
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body dto.AppReleaseRequest true "App release"
// @Success      200 {object} dto.AppReleaseDataResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      403 {object} response.ErrorResponse
// @Failure      409 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/app/releases [post]
func (h *AppReleaseHandler) CreateAdmin(c *gin.Context) {
	var req dto.AppReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	value, err := h.service.Create(c.Request.Context(), actorID, &req)
	if err != nil {
		h.respondError(c, err, "create app release")
		return
	}
	response.Ok(c, dto.NewAppReleaseData(value))
}

// UpdateAdmin godoc
// @Summary      Update an app release
// @Description  Requires app_release:update; publication changes also require app_release:publish
// @ID           updateAdminAppRelease
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "App release ID"
// @Param        request body dto.AppReleaseRequest true "App release"
// @Success      200 {object} dto.AppReleaseDataResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      403 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Failure      409 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/app/releases/{id} [put]
func (h *AppReleaseHandler) UpdateAdmin(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.AppReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	value, err := h.service.Update(c.Request.Context(), actorID, id, &req)
	if err != nil {
		h.respondError(c, err, "update app release")
		return
	}
	response.Ok(c, dto.NewAppReleaseData(value))
}

// DeleteAdmin godoc
// @Summary      Delete an app release
// @Description  Requires app_release:delete
// @ID           deleteAdminAppRelease
// @Tags         admin
// @Produce      json
// @Param        id path int true "App release ID"
// @Success      200 {object} response.MessageResponse
// @Failure      404 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/app/releases/{id} [delete]
func (h *AppReleaseHandler) DeleteAdmin(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.respondError(c, err, "delete app release")
		return
	}
	response.OkWithMsg(c, "应用版本已删除")
}

// Publish godoc
// @Summary      Publish an app release
// @Description  Requires app_release:publish
// @ID           publishAppRelease
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "App release ID"
// @Param        request body dto.PublishReleaseRequest false "Publication options"
// @Success      200 {object} dto.AppReleaseDataResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/app/releases/{id}/publish [patch]
func (h *AppReleaseHandler) Publish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.PublishReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	value, err := h.service.Publish(c.Request.Context(), id, actorID, req.CreateAnnouncement)
	if err != nil {
		h.respondError(c, err, "publish app release")
		return
	}
	response.Ok(c, dto.NewAppReleaseData(value))
}

// Disable godoc
// @Summary      Disable an app release
// @Description  Requires app_release:publish
// @ID           disableAppRelease
// @Tags         admin
// @Produce      json
// @Param        id path int true "App release ID"
// @Success      200 {object} dto.AppReleaseDataResponse
// @Failure      404 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/app/releases/{id}/disable [patch]
func (h *AppReleaseHandler) Disable(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	value, err := h.service.Disable(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "disable app release")
		return
	}
	response.Ok(c, dto.NewAppReleaseData(value))
}

func (h *AppReleaseHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrAppReleaseNotFound):
		response.Error(c, appErrors.ErrNotFound("应用版本不存在"))
	case errors.Is(err, service.ErrInvalidAppRelease):
		response.Error(c, appErrors.ErrValidation("应用版本参数不正确"))
	case errors.Is(err, service.ErrInvalidDownloadURL):
		response.Error(c, appErrors.ErrValidation("下载地址必须是 HTTP 或 HTTPS URL"))
	case errors.Is(err, service.ErrInvalidChecksum):
		response.Error(c, appErrors.ErrValidation("文件校验和必须是 64 位小写十六进制"))
	case errors.Is(err, service.ErrAppReleaseForbidden):
		response.Error(c, appErrors.ErrForbidden("没有执行该操作的权限"))
	case errors.Is(err, service.ErrVersionExists):
		response.Error(c, appErrors.ErrConflict("该平台版本号已存在"))
	case errors.Is(err, service.ErrInvalidAnnouncementLink):
		response.Error(c, appErrors.ErrValidation("关联公告不存在"))
	case errors.Is(err, service.ErrAnnouncementLinked):
		response.Error(c, appErrors.ErrConflict("关联公告已被其他版本使用"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("应用版本操作失败"))
	}
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("应用版本 ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}
