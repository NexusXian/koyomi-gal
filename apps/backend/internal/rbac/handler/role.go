package handler

import (
	"errors"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/rbac/dto"
	"backend/internal/rbac/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RoleHandler struct {
	rbacService *service.RBACService
}

func NewRoleHandler(rbacService *service.RBACService) *RoleHandler {
	return &RoleHandler{rbacService: rbacService}
}

// List godoc
// @Summary      查看角色列表
// @Description  返回全部角色
// @ID           listRoles
// @Tags         roles
// @Produce      json
// @Success      200 {object} dto.RoleListResponse "角色列表"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询角色失败"
// @Security     BearerAuth
// @Router       /api/v1/roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	roles, err := h.rbacService.ListRoles(c.Request.Context())
	if err != nil {
		response.Error(c, appErrors.ErrInternal("查询角色失败"))
		return
	}
	response.Ok(c, dto.NewRoleResponses(roles))
}

// Create godoc
// @Summary      创建角色
// @Description  创建新角色，code 全局唯一且创建后不可修改
// @ID           createRole
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateRoleRequest true "创建角色请求"
// @Success      200 {object} dto.RoleResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      409 {object} response.ErrorResponse "角色 code 已存在"
// @Failure      500 {object} response.ErrorResponse "创建角色失败"
// @Security     BearerAuth
// @Router       /api/v1/roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}

	role, err := h.rbacService.CreateRole(c.Request.Context(), &req)
	if err != nil {
		h.respondRoleError(c, err, "create role")
		return
	}
	response.Ok(c, dto.NewRoleResponse(role))
}

// Get godoc
// @Summary      查看角色详情
// @Description  按角色 ID 返回角色详情
// @ID           getRole
// @Tags         roles
// @Produce      json
// @Param        id path int true "角色 ID"
// @Success      200 {object} dto.RoleResponse "角色详情"
// @Failure      400 {object} response.ErrorResponse "角色 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "角色不存在"
// @Failure      500 {object} response.ErrorResponse "查询角色失败"
// @Security     BearerAuth
// @Router       /api/v1/roles/{id} [get]
func (h *RoleHandler) Get(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || roleID <= 0 {
		response.Error(c, appErrors.ErrValidation("角色 ID 格式不正确"))
		return
	}

	role, err := h.rbacService.GetRole(c.Request.Context(), roleID)
	if err != nil {
		h.respondRoleError(c, err, "get role")
		return
	}
	response.Ok(c, dto.NewRoleResponse(role))
}

// Update godoc
// @Summary      更新角色
// @Description  更新角色名称和描述，code 不可修改
// @ID           updateRole
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id path int true "角色 ID"
// @Param        request body dto.UpdateRoleRequest true "更新角色请求"
// @Success      200 {object} dto.RoleResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "角色不存在"
// @Failure      500 {object} response.ErrorResponse "更新角色失败"
// @Security     BearerAuth
// @Router       /api/v1/roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || roleID <= 0 {
		response.Error(c, appErrors.ErrValidation("角色 ID 格式不正确"))
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}

	role, err := h.rbacService.UpdateRole(c.Request.Context(), roleID, &req)
	if err != nil {
		h.respondRoleError(c, err, "update role")
		return
	}
	response.Ok(c, dto.NewRoleResponse(role))
}

// Delete godoc
// @Summary      删除角色
// @Description  删除角色并清理用户角色、角色权限关联，超级管理员角色不可删除
// @ID           deleteRole
// @Tags         roles
// @Produce      json
// @Param        id path int true "角色 ID"
// @Success      200 {object} response.MessageResponse "角色已删除"
// @Failure      400 {object} response.ErrorResponse "角色 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "角色不存在"
// @Failure      409 {object} response.ErrorResponse "角色受保护，不允许删除"
// @Failure      500 {object} response.ErrorResponse "删除角色失败"
// @Security     BearerAuth
// @Router       /api/v1/roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || roleID <= 0 {
		response.Error(c, appErrors.ErrValidation("角色 ID 格式不正确"))
		return
	}

	if err := h.rbacService.DeleteRole(c.Request.Context(), roleID); err != nil {
		h.respondRoleError(c, err, "delete role")
		return
	}
	response.OkWithMsg(c, "角色已删除")
}

// GetPermissions godoc
// @Summary      查看角色权限
// @Description  返回角色当前拥有的全部权限
// @ID           getRolePermissions
// @Tags         roles
// @Produce      json
// @Param        id path int true "角色 ID"
// @Success      200 {object} dto.PermissionListResponse "权限列表"
// @Failure      400 {object} response.ErrorResponse "角色 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "角色不存在"
// @Failure      500 {object} response.ErrorResponse "查询角色权限失败"
// @Security     BearerAuth
// @Router       /api/v1/roles/{id}/permissions [get]
func (h *RoleHandler) GetPermissions(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || roleID <= 0 {
		response.Error(c, appErrors.ErrValidation("角色 ID 格式不正确"))
		return
	}

	permissions, err := h.rbacService.GetRolePermissions(c.Request.Context(), roleID)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			response.Error(c, appErrors.ErrNotFound("角色不存在"))
			return
		}
		logger.Error("get role permissions", zap.Int64("role_id", roleID), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询角色权限失败"))
		return
	}
	response.Ok(c, dto.NewPermissionResponses(permissions))
}

// UpdatePermissions godoc
// @Summary      更新角色权限
// @Description  全量替换角色拥有的权限集合
// @ID           updateRolePermissions
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id path int true "角色 ID"
// @Param        request body dto.UpdateRolePermissionsRequest true "更新角色权限请求"
// @Success      200 {object} response.MessageResponse "角色权限已更新"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "角色不存在"
// @Failure      500 {object} response.ErrorResponse "更新角色权限失败"
// @Security     BearerAuth
// @Router       /api/v1/roles/{id}/permissions [put]
func (h *RoleHandler) UpdatePermissions(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || roleID <= 0 {
		response.Error(c, appErrors.ErrValidation("角色 ID 格式不正确"))
		return
	}

	var req dto.UpdateRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}

	err = h.rbacService.SetRolePermissions(c.Request.Context(), actorID, roleID, req.PermissionIDs)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRoleNotFound):
			response.Error(c, appErrors.ErrNotFound("角色不存在"))
		case errors.Is(err, service.ErrUnknownPermissionIDs):
			response.Error(c, appErrors.ErrValidation("权限列表包含不存在的权限"))
		case errors.Is(err, service.ErrSuperAdminPermissions):
			response.Error(c, appErrors.ErrForbidden("只有超级管理员可以修改超级管理员角色权限"))
		default:
			logger.Error("update role permissions", zap.Int64("role_id", roleID), zap.Error(err))
			response.Error(c, appErrors.ErrInternal("更新角色权限失败"))
		}
		return
	}

	logger.Info(
		"update role permissions",
		zap.Int64("role_id", roleID),
		zap.Int64s("permission_ids", req.PermissionIDs),
		zap.Uint("operator_id", actorID),
	)
	response.OkWithMsg(c, "角色权限已更新")
}

func (h *RoleHandler) respondRoleError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrRoleNotFound):
		response.Error(c, appErrors.ErrNotFound("角色不存在"))
	case errors.Is(err, service.ErrRoleCodeExists):
		response.Error(c, appErrors.ErrConflict("角色 code 已存在"))
	case errors.Is(err, service.ErrInvalidRoleCode):
		response.Error(c, appErrors.ErrValidation("角色 code 格式不正确"))
	case errors.Is(err, service.ErrRoleProtected):
		response.Error(c, appErrors.ErrConflict("角色受保护，不允许删除"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("角色操作失败"))
	}
}
