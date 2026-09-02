package handler

import (
	"errors"

	"backend/internal/middleware"
	"backend/internal/user/dto"
	"backend/internal/user/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UserProfileHandler struct {
	profileService *service.UserProfileService
}

func NewUserProfileHandler(profileService *service.UserProfileService) *UserProfileHandler {
	return &UserProfileHandler{profileService: profileService}
}

// UpdateMe godoc
// @Summary      更新当前用户资料
// @Description  更新头像引用；头像必须是本人上传的 avatars 分类图片，更换后旧头像资源会被删除
// @ID           updateMe
// @Tags         me
// @Accept       json
// @Produce      json
// @Param        request body dto.UpdateMeRequest true "资料更新请求"
// @Success      200 {object} dto.MeResponse "更新后的用户资料"
// @Failure      400 {object} response.ErrorResponse "请求参数或头像资源不合法"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      500 {object} response.ErrorResponse "更新失败"
// @Security     BearerAuth
// @Router       /api/v1/me [patch]
func (h *UserProfileHandler) UpdateMe(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req dto.UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数不正确"))
		return
	}
	data, err := h.profileService.UpdateMe(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondError(c, err, "update me")
		return
	}
	response.Ok(c, data)
}

// GetPreferences godoc
// @Summary      查看个性化背景设置
// @Description  返回当前用户的背景偏好；从未设置时返回默认值
// @ID           getMePreferences
// @Tags         me
// @Produce      json
// @Success      200 {object} dto.UserPreferencesResponse "背景偏好"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      500 {object} response.ErrorResponse "查询失败"
// @Security     BearerAuth
// @Router       /api/v1/me/preferences [get]
func (h *UserProfileHandler) GetPreferences(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	data, err := h.profileService.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err, "get preferences")
		return
	}
	response.Ok(c, data)
}

// UpdatePreferences godoc
// @Summary      更新个性化背景设置
// @Description  保存背景偏好；custom 来源必须引用本人上传的 backgrounds 分类图片，更换后旧背景资源会被删除
// @ID           updateMePreferences
// @Tags         me
// @Accept       json
// @Produce      json
// @Param        request body dto.UpdateUserPreferencesRequest true "背景偏好更新请求"
// @Success      200 {object} dto.UserPreferencesResponse "保存后的背景偏好"
// @Failure      400 {object} response.ErrorResponse "请求参数或背景资源不合法"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      500 {object} response.ErrorResponse "保存失败"
// @Security     BearerAuth
// @Router       /api/v1/me/preferences [patch]
func (h *UserProfileHandler) UpdatePreferences(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req dto.UpdateUserPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数不正确"))
		return
	}
	data, err := h.profileService.UpdatePreferences(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondError(c, err, "update preferences")
		return
	}
	response.Ok(c, data)
}

func (h *UserProfileHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrInvalidAvatarAsset):
		response.Error(c, appErrors.ErrValidation("头像资源不合法"))
	case errors.Is(err, service.ErrInvalidBackgroundAsset):
		response.Error(c, appErrors.ErrValidation("背景图片资源不合法"))
	case errors.Is(err, service.ErrInvalidPreferences):
		response.Error(c, appErrors.ErrValidation("背景设置不完整"))
	case errors.Is(err, service.ErrInvalidCredentials):
		response.Error(c, appErrors.ErrNotFound("用户不存在"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("操作失败"))
	}
}

func currentUserID(c *gin.Context) (uint, bool) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return 0, false
	}
	return userID, true
}
