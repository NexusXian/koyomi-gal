package handler

import (
	"errors"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UserRelationHandler struct {
	ratingService    *service.RatingService
	favoriteService  *service.FavoriteService
	userStateService *service.UserStateService
	relationService  *service.UserRelationService
}

func NewUserRelationHandler(
	ratingService *service.RatingService,
	favoriteService *service.FavoriteService,
	userStateService *service.UserStateService,
	relationService *service.UserRelationService,
) *UserRelationHandler {
	return &UserRelationHandler{
		ratingService:    ratingService,
		favoriteService:  favoriteService,
		userStateService: userStateService,
		relationService:  relationService,
	}
}

// UpsertRating godoc
// @Summary      评分 Galgame
// @Description  创建或更新当前用户对 Galgame 的评分，并重新计算评分统计
// @ID           upsertGalgameRating
// @Tags         galgames
// @Accept       json
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        request body dto.UpsertRatingRequest true "评分请求"
// @Success      200 {object} dto.RatingDataResponse "评分结果"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "评分失败"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id}/rating [put]
func (h *UserRelationHandler) UpsertRating(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	var req dto.UpsertRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	rating, err := h.ratingService.UpsertRating(c.Request.Context(), id, userID, req.Score)
	if err != nil {
		h.respondRelationError(c, err, "upsert galgame rating")
		return
	}
	response.Ok(c, dto.NewRatingData(rating))
}

// DeleteRating godoc
// @Summary      删除 Galgame 评分
// @Description  删除当前用户对 Galgame 的评分，并重新计算评分统计
// @ID           deleteGalgameRating
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} response.MessageResponse "评分已删除"
// @Failure      400 {object} response.ErrorResponse "Galgame ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在或未评分"
// @Failure      500 {object} response.ErrorResponse "删除评分失败"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id}/rating [delete]
func (h *UserRelationHandler) DeleteRating(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if err := h.ratingService.DeleteRating(c.Request.Context(), id, userID); err != nil {
		h.respondRelationError(c, err, "delete galgame rating")
		return
	}
	response.OkWithMsg(c, "评分已删除")
}

// AddFavorite godoc
// @Summary      收藏 Galgame
// @Description  收藏当前用户与 Galgame 的关系，并原子更新收藏计数
// @ID           addGalgameFavorite
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.FavoriteDataResponse "收藏结果"
// @Failure      400 {object} response.ErrorResponse "Galgame ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      409 {object} response.ErrorResponse "已收藏该 Galgame"
// @Failure      500 {object} response.ErrorResponse "收藏失败"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id}/favorite [post]
func (h *UserRelationHandler) AddFavorite(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	favorite, err := h.favoriteService.AddFavorite(c.Request.Context(), id, userID)
	if err != nil {
		h.respondRelationError(c, err, "add galgame favorite")
		return
	}
	response.Ok(c, dto.NewFavoriteData(favorite))
}

// RemoveFavorite godoc
// @Summary      取消收藏 Galgame
// @Description  删除当前用户与 Galgame 的收藏关系，并原子更新收藏计数
// @ID           removeGalgameFavorite
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} response.MessageResponse "收藏已取消"
// @Failure      400 {object} response.ErrorResponse "Galgame ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在或未收藏"
// @Failure      500 {object} response.ErrorResponse "取消收藏失败"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id}/favorite [delete]
func (h *UserRelationHandler) RemoveFavorite(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if err := h.favoriteService.RemoveFavorite(c.Request.Context(), id, userID); err != nil {
		h.respondRelationError(c, err, "remove galgame favorite")
		return
	}
	response.OkWithMsg(c, "收藏已取消")
}

// UpsertState godoc
// @Summary      设置 Galgame 游玩状态
// @Description  创建或更新当前用户对 Galgame 的游玩状态和游玩时长
// @ID           upsertGalgameUserState
// @Tags         galgames
// @Accept       json
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        request body dto.UpsertUserStateRequest true "游玩状态请求"
// @Success      200 {object} dto.UserStateDataResponse "游玩状态结果"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "设置游玩状态失败"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id}/state [put]
func (h *UserRelationHandler) UpsertState(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	var req dto.UpsertUserStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	userState, err := h.userStateService.UpsertState(
		c.Request.Context(), id, userID, req.State, req.PlayTimeMinutes,
	)
	if err != nil {
		h.respondRelationError(c, err, "upsert galgame user state")
		return
	}
	response.Ok(c, dto.NewUserStateData(userState))
}

// DeleteState godoc
// @Summary      删除 Galgame 游玩状态
// @Description  删除当前用户对 Galgame 的游玩状态
// @ID           deleteGalgameUserState
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} response.MessageResponse "游玩状态已删除"
// @Failure      400 {object} response.ErrorResponse "Galgame ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在或未设置游玩状态"
// @Failure      500 {object} response.ErrorResponse "删除游玩状态失败"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id}/state [delete]
func (h *UserRelationHandler) DeleteState(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if err := h.userStateService.DeleteState(c.Request.Context(), id, userID); err != nil {
		h.respondRelationError(c, err, "delete galgame user state")
		return
	}
	response.OkWithMsg(c, "游玩状态已删除")
}

// GetMyRelation godoc
// @Summary      查看当前用户与 Galgame 的关系
// @Description  返回当前用户对 Galgame 的评分、收藏和游玩状态
// @ID           getMyGalgameRelation
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.GalgameUserRelationResponse "用户关系详情"
// @Failure      400 {object} response.ErrorResponse "Galgame ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "查询用户关系失败"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id}/me [get]
func (h *UserRelationHandler) GetMyRelation(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	summary, err := h.relationService.GetGalgameRelation(c.Request.Context(), id, userID)
	if err != nil {
		h.respondRelationError(c, err, "get galgame user relation")
		return
	}
	response.Ok(c, dto.GalgameUserRelationData{
		GalgameID: id,
		Rating:    dto.NewRatingData(summary.Rating),
		Favorite:  dto.NewFavoriteData(summary.Favorite),
		State:     dto.NewUserStateData(summary.State),
	})
}

func (h *UserRelationHandler) respondRelationError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrGalgameNotFound):
		response.Error(c, appErrors.ErrNotFound("Galgame 不存在"))
	case errors.Is(err, service.ErrInvalidScore):
		response.Error(c, appErrors.ErrValidation("评分必须是 1-10 的整数"))
	case errors.Is(err, service.ErrRatingNotFound):
		response.Error(c, appErrors.ErrNotFound("未评分该 Galgame"))
	case errors.Is(err, service.ErrAlreadyFavorited):
		response.Error(c, appErrors.ErrConflict("已收藏该 Galgame"))
	case errors.Is(err, service.ErrFavoriteNotFound):
		response.Error(c, appErrors.ErrNotFound("未收藏该 Galgame"))
	case errors.Is(err, service.ErrInvalidUserState):
		response.Error(c, appErrors.ErrValidation("游玩状态不正确"))
	case errors.Is(err, service.ErrInvalidPlayTime):
		response.Error(c, appErrors.ErrValidation("游玩时长不能为负数"))
	case errors.Is(err, service.ErrUserStateNotFound):
		response.Error(c, appErrors.ErrNotFound("未设置游玩状态"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("Galgame 用户关系操作失败"))
	}
}
