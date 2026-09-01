package handler

import (
	"errors"
	"strconv"

	"backend/internal/banner/dto"
	"backend/internal/banner/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type BannerHandler struct {
	bannerService *service.BannerService
}

func NewBannerHandler(bannerService *service.BannerService) *BannerHandler {
	return &BannerHandler{bannerService: bannerService}
}

// ListBanners godoc
// @Summary      查看有效 Banner
// @Description  返回当前启用且在展示时间范围内的 Banner
// @ID           listBanners
// @Tags         banners
// @Produce      json
// @Success      200 {object} dto.BannerListResponse "Banner 列表"
// @Failure      500 {object} response.ErrorResponse "查询 Banner 失败"
// @Router       /api/v1/banners [get]
func (h *BannerHandler) ListBanners(c *gin.Context) {
	banners, err := h.bannerService.ListPublic(c.Request.Context())
	if err != nil {
		h.respondError(c, err, "list banners")
		return
	}
	response.Ok(c, dto.NewBannerList(banners))
}

// ListAdminBanners godoc
// @Summary      管理端查询 Banner
// @Description  分页返回全部 Banner；需要 banner:read 权限
// @ID           listAdminBanners
// @Tags         admin
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.AdminBannerListResponse "Banner 列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询 Banner 失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/banners [get]
func (h *BannerHandler) ListAdminBanners(c *gin.Context) {
	var query dto.AdminBannerQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	banners, total, page, limit, err := h.bannerService.ListAdmin(c.Request.Context(), query.Page, query.Limit)
	if err != nil {
		h.respondError(c, err, "list admin banners")
		return
	}
	response.Ok(c, dto.BannerListData{Items: dto.NewBannerList(banners), Total: total, Page: page, Limit: limit})
}

// GetAdminBanner godoc
// @Summary      管理端查看 Banner
// @Description  按 ID 返回 Banner；需要 banner:read 权限
// @ID           getAdminBanner
// @Tags         admin
// @Produce      json
// @Param        id path int true "Banner ID"
// @Success      200 {object} dto.BannerDataResponse "Banner 详情"
// @Failure      400 {object} response.ErrorResponse "Banner ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "Banner 不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/banners/{id} [get]
func (h *BannerHandler) GetAdminBanner(c *gin.Context) {
	id, ok := parseBannerID(c)
	if !ok {
		return
	}
	banner, err := h.bannerService.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "get admin banner")
		return
	}
	response.Ok(c, dto.NewBannerData(banner))
}

// CreateAdminBanner godoc
// @Summary      创建 Banner
// @Description  创建首页 Banner；需要 banner:create 权限
// @ID           createAdminBanner
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateBannerRequest true "创建 Banner 请求"
// @Success      200 {object} dto.BannerDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Security     BearerAuth
// @Router       /api/v1/admin/banners [post]
func (h *BannerHandler) CreateAdminBanner(c *gin.Context) {
	var req dto.CreateBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	banner, err := h.bannerService.Create(c.Request.Context(), &req)
	if err != nil {
		h.respondError(c, err, "create banner")
		return
	}
	response.Ok(c, dto.NewBannerData(banner))
}

// UpdateAdminBanner godoc
// @Summary      更新 Banner
// @Description  全量更新 Banner；需要 banner:update 权限
// @ID           updateAdminBanner
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Banner ID"
// @Param        request body dto.UpdateBannerRequest true "更新 Banner 请求"
// @Success      200 {object} dto.BannerDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      404 {object} response.ErrorResponse "Banner 不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/banners/{id} [put]
func (h *BannerHandler) UpdateAdminBanner(c *gin.Context) {
	id, ok := parseBannerID(c)
	if !ok {
		return
	}
	var req dto.UpdateBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	banner, err := h.bannerService.Update(c.Request.Context(), id, &req)
	if err != nil {
		h.respondError(c, err, "update banner")
		return
	}
	response.Ok(c, dto.NewBannerData(banner))
}

// DeleteAdminBanner godoc
// @Summary      删除 Banner
// @Description  删除 Banner；需要 banner:delete 权限
// @ID           deleteAdminBanner
// @Tags         admin
// @Produce      json
// @Param        id path int true "Banner ID"
// @Success      200 {object} response.MessageResponse "Banner 已删除"
// @Failure      400 {object} response.ErrorResponse "Banner ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "Banner 不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/banners/{id} [delete]
func (h *BannerHandler) DeleteAdminBanner(c *gin.Context) {
	id, ok := parseBannerID(c)
	if !ok {
		return
	}
	if err := h.bannerService.Delete(c.Request.Context(), id); err != nil {
		h.respondError(c, err, "delete banner")
		return
	}
	response.OkWithMsg(c, "Banner 已删除")
}

func (h *BannerHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrBannerNotFound):
		response.Error(c, appErrors.ErrNotFound("Banner 不存在"))
	case errors.Is(err, service.ErrInvalidBannerInput):
		response.Error(c, appErrors.ErrValidation("Banner 标题和图片地址不能为空"))
	case errors.Is(err, service.ErrInvalidBannerLink):
		response.Error(c, appErrors.ErrValidation("Banner 链接格式不正确"))
	case errors.Is(err, service.ErrInvalidSchedule):
		response.Error(c, appErrors.ErrValidation("Banner 展示时间范围不正确"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("Banner 操作失败"))
	}
}

func parseBannerID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("Banner ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}
