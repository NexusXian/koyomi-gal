package handler

import (
	"errors"
	"strconv"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
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

// PutMyRating godoc
// @Summary      创建或更新当前用户的 Galgame 评价
// @Description  使用扁平字段完整替换可编辑评价；overall 使用现有 score 存储；未提供的可空字段保存为 null；维度评分范围均为 1-10；recommendation 为 -1/0/1/2；spoiler_level 为 0/1/2
// @ID           putMyGalgameRating
// @Tags         galgames
// @Accept       json
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        request body dto.PutRatingRequest true "多维评价"
// @Success      200 {object} dto.RatingRecordResponse "评价详情"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "保存评价失败"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id}/ratings/me [put]
func (h *UserRelationHandler) PutMyRating(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	var req dto.PutRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	rating, err := h.ratingService.PutRating(c.Request.Context(), id, userID, &req)
	if err != nil {
		h.respondRelationError(c, err, "put multidimensional galgame rating")
		return
	}
	response.Ok(c, dto.NewRatingRecordData(rating))
}

// GetMyRating godoc
// @Summary      查询当前用户的 Galgame 评价
// @ID           getMyGalgameRating
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.RatingRecordResponse "评价详情；未评分时 data 为 null"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id}/ratings/me [get]
func (h *UserRelationHandler) GetMyRating(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	rating, err := h.ratingService.GetMyRating(c.Request.Context(), id, userID)
	if err != nil {
		h.respondRelationError(c, err, "get current user galgame rating")
		return
	}
	if rating == nil {
		response.Ok(c, nil)
		return
	}
	response.Ok(c, dto.NewRatingRecordData(rating))
}

// DeleteMyRating godoc
// @Summary      删除当前用户的 Galgame 评价
// @ID           deleteMyGalgameRating
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} response.MessageResponse "评价已删除"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 或评价不存在"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id}/ratings/me [delete]
func (h *UserRelationHandler) DeleteMyRating(c *gin.Context) {
	h.DeleteRating(c)
}

// ListRatings godoc
// @Summary      查询 Galgame 评价列表
// @Description  支持 newest、highest、lowest、popular 排序；已登录时返回当前用户的 liked 状态
// @ID           listGalgameRatings
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量" default(20) maximum(100)
// @Param        limit query int false "兼容的每页数量参数" maximum(100)
// @Param        sort query string false "排序" Enums(newest,highest,lowest,popular) default(newest)
// @Success      200 {object} dto.RatingListResponse "评价列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Router       /api/v1/galgames/{id}/ratings [get]
func (h *UserRelationHandler) ListRatings(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	var query dto.RatingListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	page := query.Page
	if page == 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize == 0 {
		pageSize = query.Limit
	}
	if pageSize == 0 {
		pageSize = 20
	}
	var viewerID *uint
	if currentUserID, authenticated := middleware.CurrentUserID(c); authenticated {
		viewerID = &currentUserID
	}
	items, total, err := h.ratingService.ListRatings(c.Request.Context(), id, viewerID, page, pageSize, query.Sort)
	if err != nil {
		h.respondRelationError(c, err, "list galgame ratings")
		return
	}
	data := make([]dto.RatingRecordData, 0, len(items))
	for i := range items {
		data = append(data, dto.NewRatingRecordData(&items[i]))
	}
	response.Ok(c, dto.RatingListData{Items: data, Total: total, Page: page, PageSize: pageSize})
}

// GetRatingSummary godoc
// @Summary      查询 Galgame 评价汇总
// @Description  overall 和每个维度均从数据库评价聚合；无评价的平均值为 null，维度附带各自有效评分数
// @ID           getGalgameRatingSummary
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.RatingSummaryResponse "评价汇总"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Router       /api/v1/galgames/{id}/ratings/summary [get]
func (h *UserRelationHandler) GetRatingSummary(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	summary, err := h.ratingService.Summary(c.Request.Context(), id)
	if err != nil {
		h.respondRelationError(c, err, "get galgame rating summary")
		return
	}
	response.Ok(c, dto.NewRatingSummaryData(summary))
}

// LikeRating godoc
// @Summary      点赞评价
// @Description  幂等操作，重复点赞仍返回成功
// @ID           likeGalgameRating
// @Tags         galgames
// @Produce      json
// @Param        rating_id path int true "评价 ID"
// @Success      200 {object} dto.RatingLikeResponse "点赞状态"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "评价不存在"
// @Security     BearerAuth
// @Router       /api/v1/game-ratings/{rating_id}/like [post]
func (h *UserRelationHandler) LikeRating(c *gin.Context) {
	h.setRatingLike(c, true)
}

// UnlikeRating godoc
// @Summary      取消评价点赞
// @Description  幂等操作，未点赞时仍返回成功
// @ID           unlikeGalgameRating
// @Tags         galgames
// @Produce      json
// @Param        rating_id path int true "评价 ID"
// @Success      200 {object} dto.RatingLikeResponse "点赞状态"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "评价不存在"
// @Security     BearerAuth
// @Router       /api/v1/game-ratings/{rating_id}/like [delete]
func (h *UserRelationHandler) UnlikeRating(c *gin.Context) {
	h.setRatingLike(c, false)
}

func (h *UserRelationHandler) setRatingLike(c *gin.Context, liked bool) {
	ratingID, err := strconv.ParseUint(c.Param("rating_id"), 10, 0)
	if err != nil || ratingID == 0 {
		response.Error(c, appErrors.ErrValidation("评价 ID 格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	var rating *model.RatingView
	if liked {
		rating, err = h.ratingService.LikeRating(c.Request.Context(), uint(ratingID), userID)
	} else {
		rating, err = h.ratingService.UnlikeRating(c.Request.Context(), uint(ratingID), userID)
	}
	if err != nil {
		h.respondRelationError(c, err, "set galgame rating like")
		return
	}
	response.Ok(c, dto.RatingLikeData{RatingID: rating.ID, LikeCount: rating.LikeCount, Liked: liked})
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
