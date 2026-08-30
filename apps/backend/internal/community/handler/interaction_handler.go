package handler

import (
	"errors"

	"backend/internal/community/dto"
	"backend/internal/community/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type InteractionHandler struct {
	interactionService *service.InteractionService
}

func NewInteractionHandler(interactionService *service.InteractionService) *InteractionHandler {
	return &InteractionHandler{interactionService: interactionService}
}

// LikePost godoc
// @Summary      点赞帖子
// @Description  当前用户点赞帖子，原子更新点赞计数
// @ID           likePost
// @Tags         posts
// @Produce      json
// @Param        id path int true "帖子 ID"
// @Success      200 {object} dto.PostLikeDataResponse "点赞结果"
// @Failure      400 {object} response.ErrorResponse "帖子 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "帖子不存在"
// @Failure      409 {object} response.ErrorResponse "已点赞该帖子"
// @Failure      500 {object} response.ErrorResponse "点赞失败"
// @Security     BearerAuth
// @Router       /api/v1/posts/{id}/like [post]
func (h *InteractionHandler) LikePost(c *gin.Context) {
	postID, ok := parseCommunityID(c, "帖子")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	post, err := h.interactionService.LikePost(c.Request.Context(), userID, postID)
	if err != nil {
		h.respondInteractionError(c, err, "like post")
		return
	}
	response.Ok(c, dto.PostLikeData{Liked: true, LikeCount: post.LikeCount})
}

// UnlikePost godoc
// @Summary      取消点赞帖子
// @Description  当前用户取消点赞帖子，原子更新点赞计数
// @ID           unlikePost
// @Tags         posts
// @Produce      json
// @Param        id path int true "帖子 ID"
// @Success      200 {object} dto.PostLikeDataResponse "取消结果"
// @Failure      400 {object} response.ErrorResponse "帖子 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "帖子不存在或未点赞"
// @Failure      500 {object} response.ErrorResponse "取消点赞失败"
// @Security     BearerAuth
// @Router       /api/v1/posts/{id}/like [delete]
func (h *InteractionHandler) UnlikePost(c *gin.Context) {
	postID, ok := parseCommunityID(c, "帖子")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	post, err := h.interactionService.UnlikePost(c.Request.Context(), userID, postID)
	if err != nil {
		h.respondInteractionError(c, err, "unlike post")
		return
	}
	response.Ok(c, dto.PostLikeData{Liked: false, LikeCount: post.LikeCount})
}

// FavoritePost godoc
// @Summary      收藏帖子
// @Description  当前用户收藏帖子，原子更新收藏计数
// @ID           favoritePost
// @Tags         posts
// @Produce      json
// @Param        id path int true "帖子 ID"
// @Success      200 {object} dto.PostFavoriteDataResponse "收藏结果"
// @Failure      400 {object} response.ErrorResponse "帖子 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "帖子不存在"
// @Failure      409 {object} response.ErrorResponse "已收藏该帖子"
// @Failure      500 {object} response.ErrorResponse "收藏失败"
// @Security     BearerAuth
// @Router       /api/v1/posts/{id}/favorite [post]
func (h *InteractionHandler) FavoritePost(c *gin.Context) {
	postID, ok := parseCommunityID(c, "帖子")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	post, err := h.interactionService.FavoritePost(c.Request.Context(), userID, postID)
	if err != nil {
		h.respondInteractionError(c, err, "favorite post")
		return
	}
	response.Ok(c, dto.PostFavoriteData{Favorited: true, FavoriteCount: post.FavoriteCount})
}

// UnfavoritePost godoc
// @Summary      取消收藏帖子
// @Description  当前用户取消收藏帖子，原子更新收藏计数
// @ID           unfavoritePost
// @Tags         posts
// @Produce      json
// @Param        id path int true "帖子 ID"
// @Success      200 {object} dto.PostFavoriteDataResponse "取消结果"
// @Failure      400 {object} response.ErrorResponse "帖子 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "帖子不存在或未收藏"
// @Failure      500 {object} response.ErrorResponse "取消收藏失败"
// @Security     BearerAuth
// @Router       /api/v1/posts/{id}/favorite [delete]
func (h *InteractionHandler) UnfavoritePost(c *gin.Context) {
	postID, ok := parseCommunityID(c, "帖子")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	post, err := h.interactionService.UnfavoritePost(c.Request.Context(), userID, postID)
	if err != nil {
		h.respondInteractionError(c, err, "unfavorite post")
		return
	}
	response.Ok(c, dto.PostFavoriteData{Favorited: false, FavoriteCount: post.FavoriteCount})
}

// LikeComment godoc
// @Summary      点赞评论
// @Description  当前用户点赞评论，原子更新点赞计数
// @ID           likeComment
// @Tags         comments
// @Produce      json
// @Param        id path int true "评论 ID"
// @Success      200 {object} dto.CommentLikeDataResponse "点赞结果"
// @Failure      400 {object} response.ErrorResponse "评论 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "评论不存在"
// @Failure      409 {object} response.ErrorResponse "已点赞该评论"
// @Failure      500 {object} response.ErrorResponse "点赞失败"
// @Security     BearerAuth
// @Router       /api/v1/comments/{id}/like [post]
func (h *InteractionHandler) LikeComment(c *gin.Context) {
	commentID, ok := parseCommunityID(c, "评论")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	comment, err := h.interactionService.LikeComment(c.Request.Context(), userID, commentID)
	if err != nil {
		h.respondInteractionError(c, err, "like comment")
		return
	}
	response.Ok(c, dto.CommentLikeData{Liked: true, LikeCount: comment.LikeCount})
}

// UnlikeComment godoc
// @Summary      取消点赞评论
// @Description  当前用户取消点赞评论，原子更新点赞计数
// @ID           unlikeComment
// @Tags         comments
// @Produce      json
// @Param        id path int true "评论 ID"
// @Success      200 {object} dto.CommentLikeDataResponse "取消结果"
// @Failure      400 {object} response.ErrorResponse "评论 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "评论不存在或未点赞"
// @Failure      500 {object} response.ErrorResponse "取消点赞失败"
// @Security     BearerAuth
// @Router       /api/v1/comments/{id}/like [delete]
func (h *InteractionHandler) UnlikeComment(c *gin.Context) {
	commentID, ok := parseCommunityID(c, "评论")
	if !ok {
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	comment, err := h.interactionService.UnlikeComment(c.Request.Context(), userID, commentID)
	if err != nil {
		h.respondInteractionError(c, err, "unlike comment")
		return
	}
	response.Ok(c, dto.CommentLikeData{Liked: false, LikeCount: comment.LikeCount})
}

func (h *InteractionHandler) respondInteractionError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrPostNotFound):
		response.Error(c, appErrors.ErrNotFound("帖子不存在"))
	case errors.Is(err, service.ErrCommentNotFound):
		response.Error(c, appErrors.ErrNotFound("评论不存在"))
	case errors.Is(err, service.ErrAlreadyLiked):
		response.Error(c, appErrors.ErrConflict("已点赞该内容"))
	case errors.Is(err, service.ErrLikeNotFound):
		response.Error(c, appErrors.ErrNotFound("未点赞该内容"))
	case errors.Is(err, service.ErrAlreadyFavorited):
		response.Error(c, appErrors.ErrConflict("已收藏该帖子"))
	case errors.Is(err, service.ErrFavoriteNotFound):
		response.Error(c, appErrors.ErrNotFound("未收藏该帖子"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("点赞或收藏操作失败"))
	}
}
