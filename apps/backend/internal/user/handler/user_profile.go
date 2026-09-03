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

// GetPublicProfile godoc
// @Summary      查看用户公开资料
// @Description  私密或仅注册用户可见的资料在无权查看时返回最小身份和访问标记
// @ID           getPublicUserProfile
// @Tags         users
// @Produce      json
// @Param        username path string true "用户名"
// @Success      200 {object} dto.PublicUserProfileResponse
// @Failure      401 {object} response.ErrorResponse "提供的登录凭证失效"
// @Failure      404 {object} response.ErrorResponse "用户不存在"
// @Router       /api/v1/users/{username} [get]
func (h *UserProfileHandler) GetPublicProfile(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		username = "me"
	}
	data, err := h.profileService.GetPublicProfile(c.Request.Context(), username, optionalUserID(c))
	if err != nil {
		h.respondError(c, err, "get public profile")
		return
	}
	response.Ok(c, data)
}

// GetMyProfile godoc
// @Summary  查看当前用户资料
// @ID       getMyProfile
// @Tags     me
// @Produce  json
// @Success  200 {object} dto.PublicUserProfileResponse
// @Failure  401 {object} response.ErrorResponse
// @Security BearerAuth
// @Router   /api/v1/users/me/profile [get]
func (h *UserProfileHandler) GetMyProfile(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	data, err := h.profileService.GetMyProfile(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err, "get my profile")
		return
	}
	response.Ok(c, data)
}

// UpdateProfile godoc
// @Summary  更新当前用户资料
// @Description  头像和横幅只能引用本人已激活的 avatars/profile-banners 图片；null 清除图片
// @ID       updateMyProfile
// @Tags     me
// @Accept   json
// @Produce  json
// @Param    request body dto.UpdateProfileRequest true "资料更新"
// @Success  200 {object} dto.PublicUserProfileResponse
// @Failure  400 {object} response.ErrorResponse
// @Failure  401 {object} response.ErrorResponse
// @Security BearerAuth
// @Router   /api/v1/users/me/profile [patch]
func (h *UserProfileHandler) UpdateProfile(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数不正确"))
		return
	}
	data, err := h.profileService.UpdateProfile(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondError(c, err, "update profile")
		return
	}
	response.Ok(c, data)
}

// GetPrivacy godoc
// @Summary  查看当前用户隐私设置
// @ID       getMyPrivacy
// @Tags     me
// @Produce  json
// @Success  200 {object} dto.PrivacySettingsResponse
// @Failure  401 {object} response.ErrorResponse
// @Security BearerAuth
// @Router   /api/v1/users/me/privacy [get]
func (h *UserProfileHandler) GetPrivacy(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	data, err := h.profileService.GetPrivacy(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err, "get privacy")
		return
	}
	response.Ok(c, data)
}

// UpdatePrivacy godoc
// @Summary  更新当前用户隐私设置
// @ID       updateMyPrivacy
// @Tags     me
// @Accept   json
// @Produce  json
// @Param    request body dto.UpdatePrivacyRequest true "隐私设置"
// @Success  200 {object} dto.PrivacySettingsResponse
// @Failure  400 {object} response.ErrorResponse
// @Failure  401 {object} response.ErrorResponse
// @Security BearerAuth
// @Router   /api/v1/users/me/privacy [patch]
func (h *UserProfileHandler) UpdatePrivacy(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req dto.UpdatePrivacyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数不正确"))
		return
	}
	data, err := h.profileService.UpdatePrivacy(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondError(c, err, "update privacy")
		return
	}
	response.Ok(c, data)
}

// ListUserPosts godoc
// @Summary 查看用户帖子
// @ID listUserPosts
// @Tags users
// @Produce json
// @Param username path string true "用户名"
// @Param page query int false "页码"
// @Param limit query int false "每页数量"
// @Success 200 {object} dto.ProfilePostListResponse
// @Failure 403 {object} response.ErrorResponse
// @Router /api/v1/users/{username}/posts [get]
func (h *UserProfileHandler) ListUserPosts(c *gin.Context) {
	var query dto.ProfileListQuery
	if !h.bindListQuery(c, &query) {
		return
	}
	items, total, page, limit, err := h.profileService.ListPosts(c.Request.Context(), c.Param("username"), optionalUserID(c), query.Page, query.Limit)
	if err != nil {
		h.respondError(c, err, "list user posts")
		return
	}
	response.Ok(c, dto.ProfilePostListData{Items: items, Total: total, Page: page, Limit: limit})
}

// ListUserComments godoc
// @Summary 查看用户评论
// @ID listUserComments
// @Tags users
// @Produce json
// @Param username path string true "用户名"
// @Param page query int false "页码"
// @Param limit query int false "每页数量"
// @Success 200 {object} dto.ProfileCommentListResponse
// @Failure 403 {object} response.ErrorResponse
// @Router /api/v1/users/{username}/comments [get]
func (h *UserProfileHandler) ListUserComments(c *gin.Context) {
	var query dto.ProfileListQuery
	if !h.bindListQuery(c, &query) {
		return
	}
	items, total, page, limit, err := h.profileService.ListComments(c.Request.Context(), c.Param("username"), optionalUserID(c), query.Page, query.Limit)
	if err != nil {
		h.respondError(c, err, "list user comments")
		return
	}
	response.Ok(c, dto.ProfileCommentListData{Items: items, Total: total, Page: page, Limit: limit})
}

// ListUserRatings godoc
// @Summary 查看用户评分
// @ID listUserRatings
// @Tags users
// @Produce json
// @Param username path string true "用户名"
// @Param page query int false "页码"
// @Param limit query int false "每页数量"
// @Success 200 {object} dto.ProfileGalgameListResponse
// @Failure 403 {object} response.ErrorResponse
// @Router /api/v1/users/{username}/ratings [get]
func (h *UserProfileHandler) ListUserRatings(c *gin.Context) { h.listGalgames(c, false) }

// ListUserFavorites godoc
// @Summary 查看用户收藏
// @ID listUserFavorites
// @Tags users
// @Produce json
// @Param username path string true "用户名"
// @Param page query int false "页码"
// @Param limit query int false "每页数量"
// @Success 200 {object} dto.ProfileGalgameListResponse
// @Failure 403 {object} response.ErrorResponse
// @Router /api/v1/users/{username}/favorites [get]
func (h *UserProfileHandler) ListUserFavorites(c *gin.Context) { h.listGalgames(c, true) }

func (h *UserProfileHandler) listGalgames(c *gin.Context, favorites bool) {
	var query dto.ProfileListQuery
	if !h.bindListQuery(c, &query) {
		return
	}
	var items []dto.ProfileGalgameData
	var total int64
	var page, limit int
	var err error
	if favorites {
		items, total, page, limit, err = h.profileService.ListFavorites(c.Request.Context(), c.Param("username"), optionalUserID(c), query.Page, query.Limit)
	} else {
		items, total, page, limit, err = h.profileService.ListRatings(c.Request.Context(), c.Param("username"), optionalUserID(c), query.Page, query.Limit)
	}
	if err != nil {
		h.respondError(c, err, "list user galgames")
		return
	}
	response.Ok(c, dto.ProfileGalgameListData{Items: items, Total: total, Page: page, Limit: limit})
}

// ListUserActivities godoc
// @Summary 查看用户动态
// @ID listUserActivities
// @Tags users
// @Produce json
// @Param username path string true "用户名"
// @Param page query int false "页码"
// @Param limit query int false "每页数量"
// @Success 200 {object} dto.UserActivityListResponse
// @Failure 403 {object} response.ErrorResponse
// @Router /api/v1/users/{username}/activities [get]
func (h *UserProfileHandler) ListUserActivities(c *gin.Context) {
	var query dto.ProfileListQuery
	if !h.bindListQuery(c, &query) {
		return
	}
	items, total, page, limit, err := h.profileService.ListActivities(c.Request.Context(), c.Param("username"), optionalUserID(c), query.Page, query.Limit)
	if err != nil {
		h.respondError(c, err, "list user activities")
		return
	}
	response.Ok(c, dto.UserActivityListData{Items: items, Total: total, Page: page, Limit: limit})
}

func (h *UserProfileHandler) bindListQuery(c *gin.Context, query *dto.ProfileListQuery) bool {
	if err := c.ShouldBindQuery(query); err != nil {
		response.Error(c, appErrors.ErrValidation("分页参数不正确"))
		return false
	}
	return true
}

func optionalUserID(c *gin.Context) *uint {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return nil
	}
	return &userID
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
	case errors.Is(err, service.ErrInvalidBannerAsset):
		response.Error(c, appErrors.ErrValidation("横幅资源不合法"))
	case errors.Is(err, service.ErrInvalidProfile):
		response.Error(c, appErrors.ErrValidation("用户资料不合法"))
	case errors.Is(err, service.ErrProfileNotFound):
		response.Error(c, appErrors.ErrNotFound("用户不存在"))
	case errors.Is(err, service.ErrProfileForbidden):
		response.Error(c, appErrors.ErrForbidden("该用户未公开此内容"))
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
