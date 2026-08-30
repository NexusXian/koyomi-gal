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

type AssignmentHandler struct {
	rbacService *service.RBACService
}

func NewAssignmentHandler(rbacService *service.RBACService) *AssignmentHandler {
	return &AssignmentHandler{rbacService: rbacService}
}

// ListUserRoles godoc
// @Summary      查看用户角色
// @Description  返回指定用户拥有的全部角色
// @ID           listUserRoles
// @Tags         users
// @Produce      json
// @Param        id path int true "用户 ID"
// @Success      200 {object} dto.RoleListResponse "用户角色列表"
// @Failure      400 {object} response.ErrorResponse "用户 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询用户角色失败"
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/roles [get]
func (h *AssignmentHandler) ListUserRoles(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || userID == 0 {
		response.Error(c, appErrors.ErrValidation("用户 ID 格式不正确"))
		return
	}

	roles, err := h.rbacService.GetUserRoles(c.Request.Context(), uint(userID))
	if err != nil {
		logger.Error("list user roles", zap.Uint("user_id", uint(userID)), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询用户角色失败"))
		return
	}
	response.Ok(c, dto.NewRoleResponses(roles))
}

// UpdateUserRoles godoc
// @Summary      更新用户角色
// @Description  全量替换指定用户的角色集合，传空数组即清空角色
// @ID           updateUserRoles
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path int true "用户 ID"
// @Param        request body dto.UpdateUserRolesRequest true "更新用户角色请求"
// @Success      200 {object} response.MessageResponse "用户角色已更新"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "用户不存在"
// @Failure      500 {object} response.ErrorResponse "更新用户角色失败"
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/roles [put]
func (h *AssignmentHandler) UpdateUserRoles(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || userID == 0 {
		response.Error(c, appErrors.ErrValidation("用户 ID 格式不正确"))
		return
	}

	var req dto.UpdateUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}

	err = h.rbacService.SetUserRoles(c.Request.Context(), uint(userID), req.RoleIDs)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Error(c, appErrors.ErrNotFound("用户不存在"))
		case errors.Is(err, service.ErrUnknownRoleIDs):
			response.Error(c, appErrors.ErrValidation("角色列表包含不存在的角色"))
		default:
			logger.Error("update user roles", zap.Uint("user_id", uint(userID)), zap.Error(err))
			response.Error(c, appErrors.ErrInternal("更新用户角色失败"))
		}
		return
	}

	operatorID, _ := middleware.CurrentUserID(c)
	logger.Info(
		"assign user roles",
		zap.Uint("user_id", uint(userID)),
		zap.Int64s("role_ids", req.RoleIDs),
		zap.Uint("operator_id", operatorID),
	)
	response.OkWithMsg(c, "用户角色已更新")
}

// MePermissions godoc
// @Summary      查看当前用户权限
// @Description  返回当前登录用户的角色 code 与权限 code 集合，用于前端控制 UI 展示
// @ID           getMePermissions
// @Tags         me
// @Produce      json
// @Success      200 {object} dto.MePermissionsResponse "当前用户角色与权限"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      500 {object} response.ErrorResponse "查询用户权限失败"
// @Security     BearerAuth
// @Router       /api/v1/me/permissions [get]
func (h *AssignmentHandler) MePermissions(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}

	ctx := c.Request.Context()
	roles, err := h.rbacService.GetUserRoleCodes(ctx, userID)
	if err != nil {
		logger.Error("get user role codes", zap.Uint("user_id", userID), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询用户权限失败"))
		return
	}
	permissions, err := h.rbacService.GetUserPermissions(ctx, userID)
	if err != nil {
		logger.Error("get user permissions", zap.Uint("user_id", userID), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询用户权限失败"))
		return
	}
	if roles == nil {
		roles = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}

	response.Ok(c, dto.MePermissionsData{Roles: roles, Permissions: permissions})
}
