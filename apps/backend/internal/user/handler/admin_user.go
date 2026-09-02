package handler

import (
	"errors"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/user/dto"
	"backend/internal/user/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UserAdminHandler struct {
	users *service.UserAdminService
}

func NewUserAdminHandler(users *service.UserAdminService) *UserAdminHandler {
	return &UserAdminHandler{users: users}
}

// List godoc
// @Summary      管理端查询用户列表
// @Description  按用户名、邮箱或精确数字 ID 搜索用户；需要 user:list 权限
// @ID           listAdminUsers
// @Tags         users
// @Produce      json
// @Param        keyword query string false "用户名、邮箱或用户 ID"
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.AdminUserListResponse "用户列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询用户失败"
// @Security     BearerAuth
// @Router       /api/v1/users [get]
func (h *UserAdminHandler) List(c *gin.Context) {
	var query dto.AdminUserQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	users, total, page, limit, err := h.users.List(c.Request.Context(), &query)
	if err != nil {
		response.Error(c, appErrors.ErrInternal("查询用户失败"))
		return
	}
	response.Ok(c, dto.AdminUserListData{
		Items: dto.NewAdminUserList(users), Total: total, Page: page, Limit: limit,
	})
}

// Get godoc
// @Summary      管理端查看用户详情
// @Description  按 ID 返回用户详情；需要 user:read 权限
// @ID           getAdminUser
// @Tags         users
// @Produce      json
// @Param        id path int true "用户 ID"
// @Success      200 {object} dto.AdminUserDataResponse "用户详情"
// @Failure      400 {object} response.ErrorResponse "用户 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "用户不存在"
// @Failure      500 {object} response.ErrorResponse "查询用户失败"
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [get]
func (h *UserAdminHandler) Get(c *gin.Context) {
	id, ok := parseAdminUserID(c)
	if !ok {
		return
	}
	user, err := h.users.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "get admin user")
		return
	}
	response.Ok(c, dto.NewAdminUserData(user))
}

// Create godoc
// @Summary      管理端创建用户
// @Description  创建用户并分配默认 user 角色；需要 user:create 权限
// @ID           createAdminUser
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateAdminUserRequest true "创建用户请求"
// @Success      200 {object} dto.AdminUserDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      409 {object} response.ErrorResponse "用户名或邮箱已存在"
// @Failure      500 {object} response.ErrorResponse "创建用户失败"
// @Security     BearerAuth
// @Router       /api/v1/users [post]
func (h *UserAdminHandler) Create(c *gin.Context) {
	var req dto.CreateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	user, err := h.users.Create(c.Request.Context(), &req)
	if err != nil {
		h.respondError(c, err, "create admin user")
		return
	}
	operatorID, _ := middleware.CurrentUserID(c)
	logger.Info("admin user created", zap.Uint("user_id", user.ID), zap.Uint("operator_id", operatorID))
	response.Ok(c, dto.NewAdminUserData(user))
}

// Update godoc
// @Summary      管理端更新用户
// @Description  更新提供的用户名、邮箱、封禁状态或密码；省略字段保持不变；需要 user:update 权限
// @ID           updateAdminUser
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path int true "用户 ID"
// @Param        request body dto.UpdateAdminUserRequest true "更新用户请求"
// @Success      200 {object} dto.AdminUserDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "用户不存在"
// @Failure      409 {object} response.ErrorResponse "用户名或邮箱已存在"
// @Failure      500 {object} response.ErrorResponse "更新用户失败"
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [put]
func (h *UserAdminHandler) Update(c *gin.Context) {
	id, ok := parseAdminUserID(c)
	if !ok {
		return
	}
	var req dto.UpdateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	user, err := h.users.Update(c.Request.Context(), actorID, id, &req)
	if err != nil {
		h.respondError(c, err, "update admin user")
		return
	}
	logger.Info("admin user updated", zap.Uint("user_id", user.ID), zap.Uint("operator_id", actorID))
	response.Ok(c, dto.NewAdminUserData(user))
}

// Delete godoc
// @Summary      管理端删除用户
// @Description  删除用户及用户角色关系，不能删除当前登录用户；需要 user:delete 权限
// @ID           deleteAdminUser
// @Tags         users
// @Produce      json
// @Param        id path int true "用户 ID"
// @Success      200 {object} response.MessageResponse "用户已删除"
// @Failure      400 {object} response.ErrorResponse "用户 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "用户不存在"
// @Failure      409 {object} response.ErrorResponse "不能删除当前登录用户"
// @Failure      500 {object} response.ErrorResponse "删除用户失败"
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [delete]
func (h *UserAdminHandler) Delete(c *gin.Context) {
	id, ok := parseAdminUserID(c)
	if !ok {
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if err := h.users.Delete(c.Request.Context(), actorID, id); err != nil {
		h.respondError(c, err, "delete admin user")
		return
	}
	logger.Info("admin user deleted", zap.Uint("user_id", id), zap.Uint("operator_id", actorID))
	response.OkWithMsg(c, "用户已删除")
}

func (h *UserAdminHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrAdminUserNotFound):
		response.Error(c, appErrors.ErrNotFound("用户不存在"))
	case errors.Is(err, service.ErrInvalidUsername):
		response.Error(c, appErrors.ErrValidation("用户名不能为空"))
	case errors.Is(err, service.ErrInvalidEmail):
		response.Error(c, appErrors.ErrValidation("邮箱不能为空"))
	case errors.Is(err, service.ErrInvalidPassword):
		response.Error(c, appErrors.ErrValidation("密码长度不能小于 8"))
	case errors.Is(err, service.ErrUsernameExists):
		response.Error(c, appErrors.ErrConflict("用户名已存在"))
	case errors.Is(err, service.ErrEmailExists):
		response.Error(c, appErrors.ErrConflict("邮箱已存在"))
	case errors.Is(err, service.ErrSelfDelete):
		response.Error(c, appErrors.ErrConflict("不能删除当前登录用户"))
	case errors.Is(err, service.ErrSelfBan):
		response.Error(c, appErrors.ErrForbidden("不能封禁当前登录用户"))
	case errors.Is(err, service.ErrSuperAdminUserProtected):
		response.Error(c, appErrors.ErrForbidden("只有超级管理员可以操作超级管理员账号"))
	case errors.Is(err, service.ErrLastSuperAdmin):
		response.Error(c, appErrors.ErrConflict("不能删除最后一个超级管理员"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("用户操作失败"))
	}
}

func parseAdminUserID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("用户 ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}
