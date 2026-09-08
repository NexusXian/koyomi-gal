package handler

import (
	"errors"
	"strconv"

	"backend/internal/announcement/dto"
	"backend/internal/announcement/model"
	"backend/internal/announcement/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AnnouncementHandler struct {
	service *service.AnnouncementService
}

func NewAnnouncementHandler(value *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{service: value}
}

// ListActive godoc
// @Summary      List active announcements
// @Description  Returns currently active announcements for a client platform
// @ID           listActiveAnnouncements
// @Tags         announcements
// @Produce      json
// @Param        platform query string true "Client platform" Enums(web,android,ios,windows,macos,linux,desktop)
// @Success      200 {object} dto.AnnouncementListResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /api/v1/announcements/active [get]
func (h *AnnouncementHandler) ListActive(c *gin.Context) {
	var query dto.ActiveAnnouncementQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("公告平台参数不正确"))
		return
	}
	values, err := h.service.ListActive(c.Request.Context(), query.Platform)
	if err != nil {
		h.respondError(c, err, "list active announcements")
		return
	}
	c.Header("Cache-Control", "public, max-age=30")
	response.Ok(c, dto.NewActiveAnnouncementList(values))
}

// ListAdmin godoc
// @Summary      List announcements for administration
// @Description  Requires announcement:read
// @ID           listAdminAnnouncements
// @Tags         admin
// @Produce      json
// @Param        page query int false "Page" default(1)
// @Param        limit query int false "Page size" default(20)
// @Success      200 {object} dto.AdminAnnouncementListResponse
// @Failure      400 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/announcements [get]
func (h *AnnouncementHandler) ListAdmin(c *gin.Context) {
	var query dto.AdminAnnouncementQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	values, total, page, limit, err := h.service.ListAdmin(c.Request.Context(), query.Page, query.Limit)
	if err != nil {
		h.respondError(c, err, "list admin announcements")
		return
	}
	response.Ok(c, dto.AnnouncementListData{Items: dto.NewAnnouncementList(values), Total: total, Page: page, Limit: limit})
}

// GetAdmin godoc
// @Summary      Get an announcement
// @Description  Requires announcement:read
// @ID           getAdminAnnouncement
// @Tags         admin
// @Produce      json
// @Param        id path int true "Announcement ID"
// @Success      200 {object} dto.AnnouncementDataResponse
// @Failure      404 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/announcements/{id} [get]
func (h *AnnouncementHandler) GetAdmin(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	value, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "get announcement")
		return
	}
	response.Ok(c, dto.NewAnnouncementData(value))
}

// CreateAdmin godoc
// @Summary      Create an announcement
// @Description  Requires announcement:create; publishing also requires announcement:publish
// @ID           createAdminAnnouncement
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body dto.AnnouncementRequest true "Announcement"
// @Success      200 {object} dto.AnnouncementDataResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      403 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/announcements [post]
func (h *AnnouncementHandler) CreateAdmin(c *gin.Context) {
	var req dto.AnnouncementRequest
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
		h.respondError(c, err, "create announcement")
		return
	}
	response.Ok(c, dto.NewAnnouncementData(value))
}

// UpdateAdmin godoc
// @Summary      Update an announcement
// @Description  Requires announcement:update; publication changes also require announcement:publish
// @ID           updateAdminAnnouncement
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Announcement ID"
// @Param        request body dto.AnnouncementRequest true "Announcement"
// @Success      200 {object} dto.AnnouncementDataResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      403 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/announcements/{id} [put]
func (h *AnnouncementHandler) UpdateAdmin(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.AnnouncementRequest
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
		h.respondError(c, err, "update announcement")
		return
	}
	response.Ok(c, dto.NewAnnouncementData(value))
}

// DeleteAdmin godoc
// @Summary      Delete an announcement
// @Description  Requires announcement:delete
// @ID           deleteAdminAnnouncement
// @Tags         admin
// @Produce      json
// @Param        id path int true "Announcement ID"
// @Success      200 {object} response.MessageResponse
// @Failure      404 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/announcements/{id} [delete]
func (h *AnnouncementHandler) DeleteAdmin(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.respondError(c, err, "delete announcement")
		return
	}
	response.OkWithMsg(c, "公告已删除")
}

// Publish godoc
// @Summary      Publish an announcement
// @Description  Requires announcement:publish
// @ID           publishAnnouncement
// @Tags         admin
// @Produce      json
// @Param        id path int true "Announcement ID"
// @Success      200 {object} dto.AnnouncementDataResponse
// @Failure      404 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/announcements/{id}/publish [patch]
func (h *AnnouncementHandler) Publish(c *gin.Context) {
	h.setPublication(c, true)
}

// Withdraw godoc
// @Summary      Withdraw an announcement
// @Description  Requires announcement:publish
// @ID           withdrawAnnouncement
// @Tags         admin
// @Produce      json
// @Param        id path int true "Announcement ID"
// @Success      200 {object} dto.AnnouncementDataResponse
// @Failure      404 {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/announcements/{id}/withdraw [patch]
func (h *AnnouncementHandler) Withdraw(c *gin.Context) {
	h.setPublication(c, false)
}

func (h *AnnouncementHandler) setPublication(c *gin.Context, published bool) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var announcement *model.Announcement
	var err error
	if published {
		announcement, err = h.service.Publish(c.Request.Context(), id)
	} else {
		announcement, err = h.service.Withdraw(c.Request.Context(), id)
	}
	if err != nil {
		h.respondError(c, err, "set announcement publication")
		return
	}
	response.Ok(c, dto.NewAnnouncementData(announcement))
}

func (h *AnnouncementHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrAnnouncementNotFound):
		response.Error(c, appErrors.ErrNotFound("公告不存在"))
	case errors.Is(err, service.ErrInvalidAnnouncement), errors.Is(err, service.ErrInvalidPlatform):
		response.Error(c, appErrors.ErrValidation("公告参数不正确"))
	case errors.Is(err, service.ErrInvalidSchedule):
		response.Error(c, appErrors.ErrValidation("公告时间范围不正确"))
	case errors.Is(err, service.ErrAnnouncementForbidden):
		response.Error(c, appErrors.ErrForbidden("没有执行该操作的权限"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("公告操作失败"))
	}
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("公告 ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}
