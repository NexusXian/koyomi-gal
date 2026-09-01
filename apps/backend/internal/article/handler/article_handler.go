package handler

import (
	"errors"
	"strconv"

	"backend/internal/article/dto"
	"backend/internal/article/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ArticleHandler struct {
	articleService *service.ArticleService
}

func NewArticleHandler(articleService *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{articleService: articleService}
}

// ListArticles godoc
// @Summary      查看文章列表
// @Description  分页返回已发布时间不晚于当前时间的文章，不包含正文
// @ID           listArticles
// @Tags         articles
// @Produce      json
// @Param        type query string false "类型：announcement、news、event、update"
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.ArticleListResponse "文章列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      500 {object} response.ErrorResponse "查询文章失败"
// @Router       /api/v1/articles [get]
func (h *ArticleHandler) ListArticles(c *gin.Context) {
	var query dto.ArticleQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	articles, total, page, limit, err := h.articleService.ListPublished(
		c.Request.Context(), query.Type, query.Page, query.Limit,
	)
	if err != nil {
		h.respondError(c, err, "list articles")
		return
	}
	response.Ok(c, dto.ArticleListData{Items: dto.NewArticleList(articles), Total: total, Page: page, Limit: limit})
}

// GetArticle godoc
// @Summary      查看文章详情
// @Description  返回已发布文章详情并安全增加浏览次数
// @ID           getArticle
// @Tags         articles
// @Produce      json
// @Param        id path int true "文章 ID"
// @Success      200 {object} dto.ArticleDataResponse "文章详情"
// @Failure      400 {object} response.ErrorResponse "文章 ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "文章不存在"
// @Router       /api/v1/articles/{id} [get]
func (h *ArticleHandler) GetArticle(c *gin.Context) {
	id, ok := parseArticleID(c)
	if !ok {
		return
	}
	article, err := h.articleService.GetPublished(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "get article")
		return
	}
	response.Ok(c, dto.NewArticleData(article))
}

// ListAdminArticles godoc
// @Summary      管理端查询文章
// @Description  分页返回全部文章；需要 article:read 权限
// @ID           listAdminArticles
// @Tags         admin
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.AdminArticleListResponse "文章列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Security     BearerAuth
// @Router       /api/v1/admin/articles [get]
func (h *ArticleHandler) ListAdminArticles(c *gin.Context) {
	var query dto.AdminArticleQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	articles, total, page, limit, err := h.articleService.ListAdmin(c.Request.Context(), query.Page, query.Limit)
	if err != nil {
		h.respondError(c, err, "list admin articles")
		return
	}
	response.Ok(c, dto.AdminArticleListData{Items: dto.NewAdminArticleList(articles), Total: total, Page: page, Limit: limit})
}

// GetAdminArticle godoc
// @Summary      管理端查看文章
// @Description  按 ID 返回任意发布状态的文章；需要 article:read 权限
// @ID           getAdminArticle
// @Tags         admin
// @Produce      json
// @Param        id path int true "文章 ID"
// @Success      200 {object} dto.ArticleDataResponse "文章详情"
// @Failure      404 {object} response.ErrorResponse "文章不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/articles/{id} [get]
func (h *ArticleHandler) GetAdminArticle(c *gin.Context) {
	id, ok := parseArticleID(c)
	if !ok {
		return
	}
	article, err := h.articleService.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "get admin article")
		return
	}
	response.Ok(c, dto.NewArticleData(article))
}

// CreateAdminArticle godoc
// @Summary      创建文章
// @Description  创建草稿或文章；设置发布时间还需要 article:publish 权限
// @ID           createAdminArticle
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateArticleRequest true "创建文章请求"
// @Success      200 {object} dto.ArticleDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Security     BearerAuth
// @Router       /api/v1/admin/articles [post]
func (h *ArticleHandler) CreateAdminArticle(c *gin.Context) {
	var req dto.CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	article, err := h.articleService.Create(c.Request.Context(), actorID, &req)
	if err != nil {
		h.respondError(c, err, "create article")
		return
	}
	response.Ok(c, dto.NewArticleData(article))
}

// UpdateAdminArticle godoc
// @Summary      更新文章
// @Description  全量更新；内容变更需要 article:update，发布时间变更需要 article:publish
// @ID           updateAdminArticle
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "文章 ID"
// @Param        request body dto.UpdateArticleRequest true "更新文章请求"
// @Success      200 {object} dto.ArticleDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "文章不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/articles/{id} [put]
func (h *ArticleHandler) UpdateAdminArticle(c *gin.Context) {
	id, ok := parseArticleID(c)
	if !ok {
		return
	}
	var req dto.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	article, err := h.articleService.Update(c.Request.Context(), actorID, id, &req)
	if err != nil {
		h.respondError(c, err, "update article")
		return
	}
	response.Ok(c, dto.NewArticleData(article))
}

// DeleteAdminArticle godoc
// @Summary      删除文章
// @Description  删除文章；需要 article:delete 权限
// @ID           deleteAdminArticle
// @Tags         admin
// @Produce      json
// @Param        id path int true "文章 ID"
// @Success      200 {object} response.MessageResponse "文章已删除"
// @Failure      404 {object} response.ErrorResponse "文章不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/articles/{id} [delete]
func (h *ArticleHandler) DeleteAdminArticle(c *gin.Context) {
	id, ok := parseArticleID(c)
	if !ok {
		return
	}
	if err := h.articleService.Delete(c.Request.Context(), id); err != nil {
		h.respondError(c, err, "delete article")
		return
	}
	response.OkWithMsg(c, "文章已删除")
}

func (h *ArticleHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrArticleNotFound):
		response.Error(c, appErrors.ErrNotFound("文章不存在"))
	case errors.Is(err, service.ErrInvalidArticleInput):
		response.Error(c, appErrors.ErrValidation("文章标题和正文不能为空"))
	case errors.Is(err, service.ErrInvalidArticleType):
		response.Error(c, appErrors.ErrValidation("文章类型不正确"))
	case errors.Is(err, service.ErrArticleForbidden):
		response.Error(c, appErrors.ErrForbidden("没有执行该操作的权限"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("文章操作失败"))
	}
}

func parseArticleID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("文章 ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}
