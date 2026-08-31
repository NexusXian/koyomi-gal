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

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// ListPostComments godoc
// @Summary      查看帖子评论
// @Description  分页返回帖子的一级评论及回复数量
// @ID           listPostComments
// @Tags         comments
// @Produce      json
// @Param        id path int true "帖子 ID"
// @Param        page query int false "页码（按一级评论分页）" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.CommentListResponse "评论列表"
// @Failure      400 {object} response.ErrorResponse "参数格式不正确"
// @Failure      404 {object} response.ErrorResponse "帖子不存在"
// @Failure      500 {object} response.ErrorResponse "查询评论失败"
// @Router       /api/v2/posts/{id}/comments [get]
func (h *CommentHandler) ListPostComments(c *gin.Context) {
	postID, ok := parseCommunityID(c, "帖子")
	if !ok {
		return
	}
	var query dto.CommentQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	comments, replyCounts, total, page, limit, err := h.commentService.ListByPost(
		c.Request.Context(), postID, query.Page, query.Limit,
	)
	if err != nil {
		h.respondCommentError(c, err, "list post comments")
		return
	}
	items := make([]dto.CommentData, 0, len(comments))
	for i := range comments {
		items = append(items, dto.NewCommentData(&comments[i], replyCounts[comments[i].ID]))
	}
	response.Ok(c, dto.CommentListData{Items: items, Total: total, Page: page, Limit: limit})
}

// ListCommentReplies godoc
// @Summary      查看评论回复
// @Description  分页返回一级评论下的回复
// @ID           listCommentReplies
// @Tags         comments
// @Produce      json
// @Param        id path int true "一级评论 ID"
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.CommentListResponse "回复列表"
// @Failure      400 {object} response.ErrorResponse "参数格式不正确"
// @Failure      404 {object} response.ErrorResponse "评论不存在"
// @Failure      500 {object} response.ErrorResponse "查询回复失败"
// @Router       /api/v2/comments/{id}/replies [get]
func (h *CommentHandler) ListCommentReplies(c *gin.Context) {
	commentID, ok := parseCommunityID(c, "评论")
	if !ok {
		return
	}
	var query dto.CommentQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	replies, total, page, limit, err := h.commentService.ListReplies(
		c.Request.Context(), commentID, query.Page, query.Limit,
	)
	if err != nil {
		h.respondCommentError(c, err, "list comment replies")
		return
	}
	items := make([]dto.CommentData, 0, len(replies))
	for i := range replies {
		items = append(items, dto.NewCommentData(&replies[i], 0))
	}
	response.Ok(c, dto.CommentListData{Items: items, Total: total, Page: page, Limit: limit})
}

// CreateComment godoc
// @Summary      发表评论
// @Description  登录用户评论帖子；parent_id 指向一级评论，回复另一条回复时传 reply_to_comment_id
// @ID           createComment
// @Tags         comments
// @Accept       json
// @Produce      json
// @Param        id path int true "帖子 ID"
// @Param        request body dto.CreateCommentRequest true "创建评论请求"
// @Success      200 {object} dto.CommentDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "帖子不存在"
// @Failure      500 {object} response.ErrorResponse "创建评论失败"
// @Security     BearerAuth
// @Router       /api/v1/posts/{id}/comments [post]
func (h *CommentHandler) CreateComment(c *gin.Context) {
	postID, ok := parseCommunityID(c, "帖子")
	if !ok {
		return
	}
	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	authorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	comment, err := h.commentService.Create(c.Request.Context(), authorID, postID, &req)
	if err != nil {
		h.respondCommentError(c, err, "create comment")
		return
	}
	response.Ok(c, dto.NewCommentData(comment, 0))
}

// UpdateComment godoc
// @Summary      更新评论
// @Description  更新评论内容；作者本人或拥有 comment:moderate 权限
// @ID           updateComment
// @Tags         comments
// @Accept       json
// @Produce      json
// @Param        id path int true "评论 ID"
// @Param        request body dto.UpdateCommentRequest true "更新评论请求"
// @Success      200 {object} dto.CommentDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有管理该评论的权限"
// @Failure      404 {object} response.ErrorResponse "评论不存在"
// @Failure      500 {object} response.ErrorResponse "更新评论失败"
// @Security     BearerAuth
// @Router       /api/v1/comments/{id} [put]
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	id, ok := parseCommunityID(c, "评论")
	if !ok {
		return
	}
	var req dto.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	comment, err := h.commentService.Update(c.Request.Context(), actorID, id, &req)
	if err != nil {
		h.respondCommentError(c, err, "update comment")
		return
	}
	response.Ok(c, dto.NewCommentData(comment, 0))
}

// DeleteComment godoc
// @Summary      删除评论
// @Description  删除评论（回复级联）并按子树大小原子减少帖子评论计数；作者本人或拥有 comment:moderate 权限
// @ID           deleteComment
// @Tags         comments
// @Produce      json
// @Param        id path int true "评论 ID"
// @Success      200 {object} response.MessageResponse "评论已删除"
// @Failure      400 {object} response.ErrorResponse "评论 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有管理该评论的权限"
// @Failure      404 {object} response.ErrorResponse "评论不存在"
// @Failure      500 {object} response.ErrorResponse "删除评论失败"
// @Security     BearerAuth
// @Router       /api/v1/comments/{id} [delete]
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	id, ok := parseCommunityID(c, "评论")
	if !ok {
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if err := h.commentService.Delete(c.Request.Context(), actorID, id); err != nil {
		h.respondCommentError(c, err, "delete comment")
		return
	}
	response.OkWithMsg(c, "评论已删除")
}

func (h *CommentHandler) respondCommentError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrPostNotFound):
		response.Error(c, appErrors.ErrNotFound("帖子不存在"))
	case errors.Is(err, service.ErrCommentNotFound):
		response.Error(c, appErrors.ErrNotFound("评论不存在"))
	case errors.Is(err, service.ErrForbiddenComment):
		response.Error(c, appErrors.ErrForbidden("没有管理该评论的权限"))
	case errors.Is(err, service.ErrInvalidCommentInput):
		response.Error(c, appErrors.ErrValidation("评论内容不能为空"))
	case errors.Is(err, service.ErrInvalidCommentParent):
		response.Error(c, appErrors.ErrValidation("parent_id 必须指向同帖子的一级评论"))
	case errors.Is(err, service.ErrCommentNotTopLevel):
		response.Error(c, appErrors.ErrValidation("只能查询一级评论的回复"))
	case errors.Is(err, service.ErrInvalidCommentReplyTo):
		response.Error(c, appErrors.ErrValidation("reply_to_comment_id 必须属于同一评论楼层"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("评论操作失败"))
	}
}
