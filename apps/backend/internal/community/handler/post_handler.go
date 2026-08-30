package handler

import (
	"errors"
	"strconv"

	"backend/internal/community/dto"
	"backend/internal/community/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PostHandler struct {
	postService *service.PostService
}

func NewPostHandler(postService *service.PostService) *PostHandler {
	return &PostHandler{postService: postService}
}

// ListPosts godoc
// @Summary      查询帖子列表
// @Description  分页查询社区帖子；galgame_id 过滤 Galgame 讨论帖
// @ID           listPosts
// @Tags         posts
// @Produce      json
// @Param        galgame_id query int false "Galgame ID"
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.PostListResponse "帖子列表"
// @Failure      500 {object} response.ErrorResponse "查询帖子失败"
// @Router       /api/v1/posts [get]
func (h *PostHandler) ListPosts(c *gin.Context) {
	var query dto.PostQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	posts, total, page, limit, err := h.postService.List(c.Request.Context(), &query)
	if err != nil {
		logger.Error("list posts", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询帖子失败"))
		return
	}
	response.Ok(c, dto.PostListData{
		Items: dto.NewPostListItems(posts),
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// GetPost godoc
// @Summary      查看帖子详情
// @Description  按 ID 返回帖子详情
// @ID           getPost
// @Tags         posts
// @Produce      json
// @Param        id path int true "帖子 ID"
// @Success      200 {object} dto.PostDataResponse "帖子详情"
// @Failure      400 {object} response.ErrorResponse "帖子 ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "帖子不存在"
// @Failure      500 {object} response.ErrorResponse "查询帖子失败"
// @Router       /api/v1/posts/{id} [get]
func (h *PostHandler) GetPost(c *gin.Context) {
	id, ok := parseCommunityID(c, "帖子")
	if !ok {
		return
	}
	post, err := h.postService.Get(c.Request.Context(), id)
	if err != nil {
		h.respondPostError(c, err, "get post")
		return
	}
	response.Ok(c, dto.NewPostData(post))
}

// CreatePost godoc
// @Summary      创建帖子
// @Description  登录用户发布社区帖子或 Galgame 讨论帖
// @ID           createPost
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        request body dto.CreatePostRequest true "创建帖子请求"
// @Success      200 {object} dto.PostDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "创建帖子失败"
// @Security     BearerAuth
// @Router       /api/v1/posts [post]
func (h *PostHandler) CreatePost(c *gin.Context) {
	var req dto.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	authorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	post, err := h.postService.Create(c.Request.Context(), authorID, &req)
	if err != nil {
		h.respondPostError(c, err, "create post")
		return
	}
	response.Ok(c, dto.NewPostData(post))
}

// UpdatePost godoc
// @Summary      更新帖子
// @Description  更新帖子标题和内容；作者本人或拥有 post:moderate 权限
// @ID           updatePost
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        id path int true "帖子 ID"
// @Param        request body dto.UpdatePostRequest true "更新帖子请求"
// @Success      200 {object} dto.PostDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有管理该帖子的权限"
// @Failure      404 {object} response.ErrorResponse "帖子不存在"
// @Failure      500 {object} response.ErrorResponse "更新帖子失败"
// @Security     BearerAuth
// @Router       /api/v1/posts/{id} [put]
func (h *PostHandler) UpdatePost(c *gin.Context) {
	id, ok := parseCommunityID(c, "帖子")
	if !ok {
		return
	}
	var req dto.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	post, err := h.postService.Update(c.Request.Context(), actorID, id, &req)
	if err != nil {
		h.respondPostError(c, err, "update post")
		return
	}
	response.Ok(c, dto.NewPostData(post))
}

// DeletePost godoc
// @Summary      删除帖子
// @Description  删除帖子及其评论并原子减少计数；作者本人或拥有 post:moderate 权限
// @ID           deletePost
// @Tags         posts
// @Produce      json
// @Param        id path int true "帖子 ID"
// @Success      200 {object} response.MessageResponse "帖子已删除"
// @Failure      400 {object} response.ErrorResponse "帖子 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有管理该帖子的权限"
// @Failure      404 {object} response.ErrorResponse "帖子不存在"
// @Failure      500 {object} response.ErrorResponse "删除帖子失败"
// @Security     BearerAuth
// @Router       /api/v1/posts/{id} [delete]
func (h *PostHandler) DeletePost(c *gin.Context) {
	id, ok := parseCommunityID(c, "帖子")
	if !ok {
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if err := h.postService.Delete(c.Request.Context(), actorID, id); err != nil {
		h.respondPostError(c, err, "delete post")
		return
	}
	response.OkWithMsg(c, "帖子已删除")
}

func (h *PostHandler) respondPostError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrGalgameNotFound):
		response.Error(c, appErrors.ErrNotFound("Galgame 不存在"))
	case errors.Is(err, service.ErrPostNotFound):
		response.Error(c, appErrors.ErrNotFound("帖子不存在"))
	case errors.Is(err, service.ErrForbiddenPost):
		response.Error(c, appErrors.ErrForbidden("没有管理该帖子的权限"))
	case errors.Is(err, service.ErrInvalidPostInput):
		response.Error(c, appErrors.ErrValidation("帖子标题和内容不能为空"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("帖子操作失败"))
	}
}

func parseCommunityID(c *gin.Context, resource string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation(resource+" ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}
