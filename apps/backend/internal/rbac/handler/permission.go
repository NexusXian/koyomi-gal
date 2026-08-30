package handler

import (
	"errors"
	"strconv"

	"backend/internal/rbac/dto"
	"backend/internal/rbac/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PermissionHandler struct {
	rbacService *service.RBACService
}

func NewPermissionHandler(rbacService *service.RBACService) *PermissionHandler {
	return &PermissionHandler{rbacService: rbacService}
}

// List godoc
// @Summary      查看权限列表
// @Description  返回全部权限
// @ID           listPermissions
// @Tags         permissions
// @Produce      json
// @Success      200 {object} dto.PermissionListResponse "权限列表"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询权限失败"
// @Security     BearerAuth
// @Router       /api/v1/permissions [get]
func (h *PermissionHandler) List(c *gin.Context) {
	permissions, err := h.rbacService.ListPermissions(c.Request.Context())
	if err != nil {
		response.Error(c, appErrors.ErrInternal("查询权限失败"))
		return
	}
	response.Ok(c, dto.NewPermissionResponses(permissions))
}

// Create godoc
// @Summary      创建权限
// @Description  创建新权限，code 必须为 resource:action 格式且全局唯一，创建后不可修改
// @ID           createPermission
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        request body dto.CreatePermissionRequest true "创建权限请求"
// @Success      200 {object} dto.PermissionResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      409 {object} response.ErrorResponse "权限 code 已存在"
// @Failure      500 {object} response.ErrorResponse "创建权限失败"
// @Security     BearerAuth
// @Router       /api/v1/permissions [post]
func (h *PermissionHandler) Create(c *gin.Context) {
	var req dto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}

	permission, err := h.rbacService.CreatePermission(c.Request.Context(), &req)
	if err != nil {
		h.respondPermissionError(c, err, "create permission")
		return
	}
	response.Ok(c, dto.NewPermissionResponse(permission))
}

// Update godoc
// @Summary      更新权限
// @Description  更新权限名称和描述，code/resource/action 不可修改
// @ID           updatePermission
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        id path int true "权限 ID"
// @Param        request body dto.UpdatePermissionRequest true "更新权限请求"
// @Success      200 {object} dto.PermissionResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "权限不存在"
// @Failure      500 {object} response.ErrorResponse "更新权限失败"
// @Security     BearerAuth
// @Router       /api/v1/permissions/{id} [put]
func (h *PermissionHandler) Update(c *gin.Context) {
	permissionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || permissionID <= 0 {
		response.Error(c, appErrors.ErrValidation("权限 ID 格式不正确"))
		return
	}

	var req dto.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}

	permission, err := h.rbacService.UpdatePermission(c.Request.Context(), permissionID, &req)
	if err != nil {
		h.respondPermissionError(c, err, "update permission")
		return
	}
	response.Ok(c, dto.NewPermissionResponse(permission))
}

// Delete godoc
// @Summary      删除权限
// @Description  删除权限并清理角色权限关联
// @ID           deletePermission
// @Tags         permissions
// @Produce      json
// @Param        id path int true "权限 ID"
// @Success      200 {object} response.MessageResponse "权限已删除"
// @Failure      400 {object} response.ErrorResponse "权限 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "权限不存在"
// @Failure      500 {object} response.ErrorResponse "删除权限失败"
// @Security     BearerAuth
// @Router       /api/v1/permissions/{id} [delete]
func (h *PermissionHandler) Delete(c *gin.Context) {
	permissionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || permissionID <= 0 {
		response.Error(c, appErrors.ErrValidation("权限 ID 格式不正确"))
		return
	}

	if err := h.rbacService.DeletePermission(c.Request.Context(), permissionID); err != nil {
		h.respondPermissionError(c, err, "delete permission")
		return
	}
	response.OkWithMsg(c, "权限已删除")
}

func (h *PermissionHandler) respondPermissionError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrPermissionNotFound):
		response.Error(c, appErrors.ErrNotFound("权限不存在"))
	case errors.Is(err, service.ErrPermissionCodeExists):
		response.Error(c, appErrors.ErrConflict("权限 code 已存在"))
	case errors.Is(err, service.ErrInvalidPermissionCode):
		response.Error(c, appErrors.ErrValidation("权限 code 必须为 resource:action 格式"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("权限操作失败"))
	}
}
