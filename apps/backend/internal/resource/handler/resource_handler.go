package handler

import (
	"errors"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/resource/dto"
	"backend/internal/resource/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ResourceHandler struct {
	resourceService *service.ResourceService
}

func NewResourceHandler(resourceService *service.ResourceService) *ResourceHandler {
	return &ResourceHandler{resourceService: resourceService}
}

// ListGalgameResources godoc
// @Summary      查看 Galgame 资源列表
// @Description  返回 Galgame 下全部已发布资源及其链接
// @ID           listGalgameResources
// @Tags         resources
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.ResourceListResponse "资源列表"
// @Failure      400 {object} response.ErrorResponse "Galgame ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "查询资源失败"
// @Router       /api/v1/galgames/{id}/resources [get]
func (h *ResourceHandler) ListGalgameResources(c *gin.Context) {
	galgameID, ok := parseResourceID(c, "Galgame")
	if !ok {
		return
	}
	resources, err := h.resourceService.ListPublishedByGalgame(c.Request.Context(), galgameID)
	if err != nil {
		h.respondResourceError(c, err, "list galgame resources")
		return
	}
	response.Ok(c, dto.NewResourceListData(resources))
}

// GetResource godoc
// @Summary      查看资源详情
// @Description  按 ID 返回已发布资源及其链接
// @ID           getResource
// @Tags         resources
// @Produce      json
// @Param        id path int true "资源 ID"
// @Success      200 {object} dto.ResourceDataResponse "资源详情"
// @Failure      400 {object} response.ErrorResponse "资源 ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "资源不存在"
// @Failure      500 {object} response.ErrorResponse "查询资源失败"
// @Router       /api/v1/resources/{id} [get]
func (h *ResourceHandler) GetResource(c *gin.Context) {
	id, ok := parseResourceID(c, "资源")
	if !ok {
		return
	}
	resource, err := h.resourceService.GetPublishedResource(c.Request.Context(), id)
	if err != nil {
		h.respondResourceError(c, err, "get resource")
		return
	}
	response.Ok(c, dto.NewResourceData(resource))
}

// CreateResource godoc
// @Summary      创建资源
// @Description  登录用户为已发布 Galgame 上传资源及下载链接
// @ID           createResource
// @Tags         resources
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateResourceRequest true "创建资源请求"
// @Success      200 {object} dto.ResourceDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "创建资源失败"
// @Security     BearerAuth
// @Router       /api/v1/resources [post]
func (h *ResourceHandler) CreateResource(c *gin.Context) {
	var req dto.CreateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	uploaderID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	resource, err := h.resourceService.CreateResource(c.Request.Context(), uploaderID, &req)
	if err != nil {
		h.respondResourceError(c, err, "create resource")
		return
	}
	response.Ok(c, dto.NewResourceData(resource))
}

// UpdateResource godoc
// @Summary      更新资源
// @Description  全量更新资源字段和链接；上传者本人或拥有 resource:update 权限
// @ID           updateResource
// @Tags         resources
// @Accept       json
// @Produce      json
// @Param        id path int true "资源 ID"
// @Param        request body dto.UpdateResourceRequest true "更新资源请求"
// @Success      200 {object} dto.ResourceDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有管理该资源的权限"
// @Failure      404 {object} response.ErrorResponse "资源不存在"
// @Failure      500 {object} response.ErrorResponse "更新资源失败"
// @Security     BearerAuth
// @Router       /api/v1/resources/{id} [put]
func (h *ResourceHandler) UpdateResource(c *gin.Context) {
	id, ok := parseResourceID(c, "资源")
	if !ok {
		return
	}
	var req dto.UpdateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	resource, err := h.resourceService.UpdateResource(c.Request.Context(), actorID, id, &req)
	if err != nil {
		h.respondResourceError(c, err, "update resource")
		return
	}
	response.Ok(c, dto.NewResourceData(resource))
}

// DeleteResource godoc
// @Summary      删除资源
// @Description  删除资源及其链接并原子减少计数；上传者本人或拥有 resource:delete 权限
// @ID           deleteResource
// @Tags         resources
// @Produce      json
// @Param        id path int true "资源 ID"
// @Success      200 {object} response.MessageResponse "资源已删除"
// @Failure      400 {object} response.ErrorResponse "资源 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有管理该资源的权限"
// @Failure      404 {object} response.ErrorResponse "资源不存在"
// @Failure      500 {object} response.ErrorResponse "删除资源失败"
// @Security     BearerAuth
// @Router       /api/v1/resources/{id} [delete]
func (h *ResourceHandler) DeleteResource(c *gin.Context) {
	id, ok := parseResourceID(c, "资源")
	if !ok {
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if err := h.resourceService.DeleteResource(c.Request.Context(), actorID, id); err != nil {
		h.respondResourceError(c, err, "delete resource")
		return
	}
	response.OkWithMsg(c, "资源已删除")
}

// ListAdminResources godoc
// @Summary      管理端查询资源
// @Description  分页查询全部状态的资源，可按状态筛选；需要 resource:review 权限
// @ID           listAdminResources
// @Tags         admin
// @Produce      json
// @Param        status query int false "状态过滤：0 待审核，1 已发布，2 已拒绝，3 已隐藏"
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.AdminResourceListResponse "资源列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询资源失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/resources [get]
func (h *ResourceHandler) ListAdminResources(c *gin.Context) {
	var query struct {
		Status *int16 `form:"status" binding:"omitempty,oneof=0 1 2 3"`
		Page   int    `form:"page" binding:"omitempty,min=1,max=1000000"`
		Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	resources, total, page, limit, err := h.resourceService.ListAdminResources(
		c.Request.Context(), query.Status, query.Page, query.Limit,
	)
	if err != nil {
		h.respondResourceError(c, err, "list admin resources")
		return
	}
	response.Ok(c, dto.AdminResourceListData{
		Items: dto.NewResourceListData(resources),
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// ReviewResource godoc
// @Summary      管理端审核资源
// @Description  将资源标记为已发布、已拒绝或已隐藏；需要 resource:review 权限
// @ID           reviewResource
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "资源 ID"
// @Param        request body dto.ReviewResourceRequest true "审核资源请求"
// @Success      200 {object} dto.ResourceDataResponse "审核结果"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "资源不存在"
// @Failure      500 {object} response.ErrorResponse "审核资源失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/resources/{id}/review [put]
func (h *ResourceHandler) ReviewResource(c *gin.Context) {
	id, ok := parseResourceID(c, "资源")
	if !ok {
		return
	}
	var req dto.ReviewResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	resource, err := h.resourceService.ReviewResource(c.Request.Context(), id, &req)
	if err != nil {
		h.respondResourceError(c, err, "review resource")
		return
	}
	response.Ok(c, dto.NewResourceData(resource))
}

func (h *ResourceHandler) respondResourceError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrGalgameNotFound):
		response.Error(c, appErrors.ErrNotFound("Galgame 不存在"))
	case errors.Is(err, service.ErrResourceNotFound):
		response.Error(c, appErrors.ErrNotFound("资源不存在"))
	case errors.Is(err, service.ErrForbiddenResource):
		response.Error(c, appErrors.ErrForbidden("没有管理该资源的权限"))
	case errors.Is(err, service.ErrInvalidResourceInput):
		response.Error(c, appErrors.ErrValidation("资源标题不能为空"))
	case errors.Is(err, service.ErrInvalidResourceType):
		response.Error(c, appErrors.ErrValidation("资源类型不正确"))
	case errors.Is(err, service.ErrInvalidResourceStatus):
		response.Error(c, appErrors.ErrValidation("资源状态不正确"))
	case errors.Is(err, service.ErrEmptyResourceLinks):
		response.Error(c, appErrors.ErrValidation("资源至少需要一个有效链接"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("资源操作失败"))
	}
}

func parseResourceID(c *gin.Context, resource string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation(resource+" ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}
