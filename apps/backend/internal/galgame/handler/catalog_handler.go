package handler

import (
	"errors"
	"strconv"
	"strings"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CatalogHandler struct {
	catalogService *service.CatalogService
}

func NewCatalogHandler(catalogService *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{catalogService: catalogService}
}

// ListGalgames godoc
// @Summary      查询 Galgame
// @Description  查询已发布 Galgame；多个 tag_ids 使用 AND 语义
// @ID           listGalgames
// @Tags         galgames
// @Produce      json
// @Param        keyword query string false "标题或别名关键词"
// @Param        developer_id query int false "开发商 ID"
// @Param        tag_ids query string false "Tag ID，逗号分隔" example(1,2)
// @Param        release_from query int false "最早发行年份"
// @Param        release_to query int false "最晚发行年份"
// @Param        age_rating query int false "年龄等级：0 未分级，1 全年龄，4 12+，2 15+，5 17+，3 18+"
// @Param        sort query string false "排序：latest、oldest、rating、favorite、popular" default(latest)
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.GalgameListResponse "Galgame 列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      500 {object} response.ErrorResponse "查询 Galgame 失败"
// @Router       /api/v1/galgames [get]
func (h *CatalogHandler) ListGalgames(c *gin.Context) {
	var query dto.GalgameQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	tagIDs, err := parseTagIDs(c.Query("tag_ids"))
	if err != nil {
		response.Error(c, appErrors.ErrValidation("tag_ids 格式不正确"))
		return
	}
	query.TagIDs = tagIDs

	galgames, total, page, limit, err := h.catalogService.ListPublishedGalgames(c.Request.Context(), &query)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidSort):
			response.Error(c, appErrors.ErrValidation("sort 不受支持"))
		case errors.Is(err, service.ErrInvalidReleaseRange):
			response.Error(c, appErrors.ErrValidation("发行年份范围不正确"))
		case errors.Is(err, service.ErrInvalidAgeRating):
			response.Error(c, appErrors.ErrValidation("年龄等级不正确"))
		default:
			logger.Error("list galgames", zap.Error(err))
			response.Error(c, appErrors.ErrInternal("查询 Galgame 失败"))
		}
		return
	}
	response.Ok(c, dto.GalgameListData{
		Items: dto.NewGalgameListItems(galgames),
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// ListMyGalgames godoc
// @Summary      查询我的 Galgame
// @Description  查询当前用户上传或收藏的 Galgame
// @ID           listMyGalgames
// @Tags         me
// @Produce      json
// @Param        type query string false "列表类型：uploaded、favorite" default(uploaded)
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.GalgameListResponse "Galgame 列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      500 {object} response.ErrorResponse "查询我的 Galgame 失败"
// @Security     BearerAuth
// @Router       /api/v1/me/galgames [get]
func (h *CatalogHandler) ListMyGalgames(c *gin.Context) {
	var query dto.MyGalgameQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	galgames, total, page, limit, err := h.catalogService.ListMyGalgames(
		c.Request.Context(), userID, &query,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidMyGalgameType) {
			response.Error(c, appErrors.ErrValidation("type 不受支持"))
			return
		}
		logger.Error("list my galgames", zap.Uint("user_id", userID), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询我的 Galgame 失败"))
		return
	}
	response.Ok(c, dto.GalgameListData{
		Items: dto.NewGalgameListItems(galgames),
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// GetGalgame godoc
// @Summary      查看 Galgame 详情
// @Description  按 ID 返回已发布 Galgame 详情
// @ID           getGalgame
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.GalgameDataResponse "Galgame 详情"
// @Failure      400 {object} response.ErrorResponse "Galgame ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "查询 Galgame 失败"
// @Router       /api/v1/galgames/{id} [get]
func (h *CatalogHandler) GetGalgame(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	galgame, err := h.catalogService.GetPublishedGalgame(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrGalgameNotFound) {
			response.Error(c, appErrors.ErrNotFound("Galgame 不存在"))
			return
		}
		logger.Error("get galgame", zap.Uint("galgame_id", id), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询 Galgame 失败"))
		return
	}
	response.Ok(c, dto.NewGalgameResponse(galgame))
}

// ListAdminGalgames godoc
// @Summary      管理端查询 Galgame
// @Description  查询全部状态（pending/published/rejected/hidden）的 Galgame；需要 galgame:review 权限
// @ID           listAdminGalgames
// @Tags         admin
// @Produce      json
// @Param        status query int false "状态过滤：0 待审，1 已发布，2 已拒绝，3 已隐藏"
// @Param        age_rating query int false "年龄等级过滤：0 未分级，1 全年龄，4 12+，2 15+，5 17+，3 18+"
// @Param        cover_sensitive query boolean false "敏感封面过滤：true 仅敏感，false 仅普通，不传为全部"
// @Param        keyword query string false "标题或别名关键词"
// @Param        sort query string false "排序：latest、oldest、rating、favorite、popular" default(latest)
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.GalgameListResponse "Galgame 列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询 Galgame 失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames [get]
func (h *CatalogHandler) ListAdminGalgames(c *gin.Context) {
	var query dto.AdminGalgameQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	galgames, total, page, limit, err := h.catalogService.ListAllGalgames(c.Request.Context(), &query)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidSort):
			response.Error(c, appErrors.ErrValidation("sort 不受支持"))
		case errors.Is(err, service.ErrInvalidStatus):
			response.Error(c, appErrors.ErrValidation("Galgame 状态不正确"))
		case errors.Is(err, service.ErrInvalidAgeRating):
			response.Error(c, appErrors.ErrValidation("年龄等级不正确"))
		default:
			logger.Error("list admin galgames", zap.Error(err))
			response.Error(c, appErrors.ErrInternal("查询 Galgame 失败"))
		}
		return
	}
	response.Ok(c, dto.GalgameListData{
		Items: dto.NewGalgameListItems(galgames),
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// GetAdminGalgame godoc
// @Summary      管理端查看 Galgame 详情
// @Description  按 ID 返回任意状态 Galgame 详情；需要 galgame:review 权限
// @ID           getAdminGalgame
// @Tags         admin
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.GalgameDataResponse "Galgame 详情"
// @Failure      400 {object} response.ErrorResponse "Galgame ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "查询 Galgame 失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id} [get]
func (h *CatalogHandler) GetAdminGalgame(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	galgame, err := h.catalogService.GetGalgame(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrGalgameNotFound) {
			response.Error(c, appErrors.ErrNotFound("Galgame 不存在"))
			return
		}
		logger.Error("get admin galgame", zap.Uint("galgame_id", id), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询 Galgame 失败"))
		return
	}
	response.Ok(c, dto.NewGalgameResponse(galgame))
}

// CreateGalgame godoc
// @Summary      创建 Galgame
// @Description  创建 Galgame、别名和 Tag 关联
// @ID           createGalgame
// @Tags         galgames
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateGalgameRequest true "创建 Galgame 请求"
// @Success      200 {object} dto.GalgameDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      409 {object} response.ErrorResponse "slug 已存在"
// @Failure      500 {object} response.ErrorResponse "创建 Galgame 失败"
// @Security     BearerAuth
// @Router       /api/v1/galgames [post]
func (h *CatalogHandler) CreateGalgame(c *gin.Context) {
	var req dto.CreateGalgameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	galgame, err := h.catalogService.CreateGalgame(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondCatalogError(c, err, "create galgame")
		return
	}
	response.Ok(c, dto.NewGalgameResponse(galgame))
}

// UpdateGalgame godoc
// @Summary      更新 Galgame
// @Description  全量更新 Galgame、别名和 Tag 关联
// @ID           updateGalgame
// @Tags         galgames
// @Accept       json
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        request body dto.UpdateGalgameRequest true "更新 Galgame 请求"
// @Success      200 {object} dto.GalgameDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      409 {object} response.ErrorResponse "slug 已存在"
// @Failure      500 {object} response.ErrorResponse "更新 Galgame 失败"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id} [put]
func (h *CatalogHandler) UpdateGalgame(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	var req dto.UpdateGalgameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	galgame, err := h.catalogService.UpdateGalgame(c.Request.Context(), id, &req, actorID)
	if err != nil {
		h.respondCatalogError(c, err, "update galgame")
		return
	}
	response.Ok(c, dto.NewGalgameResponse(galgame))
}

// ReviewGalgame godoc
// @Summary      管理端审核 Galgame
// @Description  将 Galgame 标记为已发布或已拒绝；需要 galgame:review 权限
// @ID           reviewGalgame
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        request body dto.ReviewGalgameRequest true "审核 Galgame 请求"
// @Success      200 {object} dto.GalgameDataResponse "审核结果"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "审核 Galgame 失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/review [put]
func (h *CatalogHandler) ReviewGalgame(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	var req dto.ReviewGalgameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	galgame, err := h.catalogService.ReviewGalgame(c.Request.Context(), actorID, id, &req)
	if err != nil {
		h.respondCatalogError(c, err, "review galgame")
		return
	}
	response.Ok(c, dto.NewGalgameResponse(galgame))
}

// BatchUpdateGalgames godoc
// @Summary      管理端批量更新 Galgame
// @Description  批量修改选中 Galgame 的年龄等级和/或敏感封面标记；两个字段至少提供一个，单次最多 500 条；需要 galgame:update 权限
// @ID           batchUpdateGalgames
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body dto.BatchUpdateGalgameRequest true "批量更新 Galgame 请求"
// @Success      200 {object} dto.BatchUpdateGalgameResponse "更新数量"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "批量更新失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/batch [patch]
func (h *CatalogHandler) BatchUpdateGalgames(c *gin.Context) {
	var req dto.BatchUpdateGalgameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数不正确"))
		return
	}
	updated, err := h.catalogService.BatchUpdateGalgame(c.Request.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAgeRating):
			response.Error(c, appErrors.ErrValidation("年龄等级不正确"))
		case errors.Is(err, service.ErrInvalidCatalogInput):
			response.Error(c, appErrors.ErrValidation("至少提供 age_rating 或 cover_sensitive 之一"))
		default:
			logger.Error("batch update galgames", zap.Error(err))
			response.Error(c, appErrors.ErrInternal("批量更新 Galgame 失败"))
		}
		return
	}
	response.Ok(c, dto.BatchUpdateGalgameData{Updated: updated})
}

// DeleteGalgame godoc
// @Summary      删除 Galgame
// @Description  删除 Galgame 及其别名和 Tag 关联
// @ID           deleteGalgame
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} response.MessageResponse "Galgame 已删除"
// @Failure      400 {object} response.ErrorResponse "Galgame ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "删除 Galgame 失败"
// @Security     BearerAuth
// @Router       /api/v1/galgames/{id} [delete]
func (h *CatalogHandler) DeleteGalgame(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	if err := h.catalogService.DeleteGalgame(c.Request.Context(), id); err != nil {
		h.respondCatalogError(c, err, "delete galgame")
		return
	}
	response.OkWithMsg(c, "Galgame 已删除")
}

// ListDevelopers godoc
// @Summary      查看开发商列表
// @Description  返回全部开发商
// @ID           listDevelopers
// @Tags         developers
// @Produce      json
// @Success      200 {object} dto.DeveloperListResponse "开发商列表"
// @Failure      500 {object} response.ErrorResponse "查询开发商失败"
// @Router       /api/v1/developers [get]
func (h *CatalogHandler) ListDevelopers(c *gin.Context) {
	developers, err := h.catalogService.ListDevelopers(c.Request.Context())
	if err != nil {
		logger.Error("list developers", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询开发商失败"))
		return
	}
	response.Ok(c, dto.NewDeveloperResponses(developers))
}

// GetDeveloper godoc
// @Summary      查看开发商详情
// @Description  按 ID 返回开发商详情
// @ID           getDeveloper
// @Tags         developers
// @Produce      json
// @Param        id path int true "开发商 ID"
// @Success      200 {object} dto.DeveloperDataResponse "开发商详情"
// @Failure      400 {object} response.ErrorResponse "开发商 ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "开发商不存在"
// @Failure      500 {object} response.ErrorResponse "查询开发商失败"
// @Router       /api/v1/developers/{id} [get]
func (h *CatalogHandler) GetDeveloper(c *gin.Context) {
	id, ok := parseID(c, "开发商")
	if !ok {
		return
	}
	developer, err := h.catalogService.GetDeveloper(c.Request.Context(), id)
	if err != nil {
		h.respondCatalogError(c, err, "get developer")
		return
	}
	response.Ok(c, dto.NewDeveloperResponse(developer))
}

// CreateDeveloper godoc
// @Summary      创建开发商
// @Description  创建 Galgame 开发商
// @ID           createDeveloper
// @Tags         developers
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateDeveloperRequest true "创建开发商请求"
// @Success      200 {object} dto.DeveloperDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      409 {object} response.ErrorResponse "slug 已存在"
// @Failure      500 {object} response.ErrorResponse "创建开发商失败"
// @Security     BearerAuth
// @Router       /api/v1/developers [post]
func (h *CatalogHandler) CreateDeveloper(c *gin.Context) {
	var req dto.CreateDeveloperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	developer, err := h.catalogService.CreateDeveloper(c.Request.Context(), &req)
	if err != nil {
		h.respondCatalogError(c, err, "create developer")
		return
	}
	response.Ok(c, dto.NewDeveloperResponse(developer))
}

// UpdateDeveloper godoc
// @Summary      更新开发商
// @Description  更新 Galgame 开发商
// @ID           updateDeveloper
// @Tags         developers
// @Accept       json
// @Produce      json
// @Param        id path int true "开发商 ID"
// @Param        request body dto.UpdateDeveloperRequest true "更新开发商请求"
// @Success      200 {object} dto.DeveloperDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "开发商不存在"
// @Failure      409 {object} response.ErrorResponse "slug 已存在"
// @Failure      500 {object} response.ErrorResponse "更新开发商失败"
// @Security     BearerAuth
// @Router       /api/v1/developers/{id} [put]
func (h *CatalogHandler) UpdateDeveloper(c *gin.Context) {
	id, ok := parseID(c, "开发商")
	if !ok {
		return
	}
	var req dto.UpdateDeveloperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	developer, err := h.catalogService.UpdateDeveloper(c.Request.Context(), id, &req)
	if err != nil {
		h.respondCatalogError(c, err, "update developer")
		return
	}
	response.Ok(c, dto.NewDeveloperResponse(developer))
}

// ListTags godoc
// @Summary      查看 Tag 列表
// @Description  返回全部 Galgame Tag
// @ID           listTags
// @Tags         tags
// @Produce      json
// @Success      200 {object} dto.TagListResponse "Tag 列表"
// @Failure      500 {object} response.ErrorResponse "查询 Tag 失败"
// @Router       /api/v1/tags [get]
func (h *CatalogHandler) ListTags(c *gin.Context) {
	tags, err := h.catalogService.ListTags(c.Request.Context())
	if err != nil {
		logger.Error("list tags", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询 Tag 失败"))
		return
	}
	response.Ok(c, dto.NewTagResponses(tags))
}

// CreateTag godoc
// @Summary      创建 Tag
// @Description  创建 Galgame Tag
// @ID           createTag
// @Tags         tags
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateTagRequest true "创建 Tag 请求"
// @Success      200 {object} dto.TagDataResponse "创建成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      409 {object} response.ErrorResponse "名称或 slug 已存在"
// @Failure      500 {object} response.ErrorResponse "创建 Tag 失败"
// @Security     BearerAuth
// @Router       /api/v1/tags [post]
func (h *CatalogHandler) CreateTag(c *gin.Context) {
	var req dto.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	tag, err := h.catalogService.CreateTag(c.Request.Context(), &req)
	if err != nil {
		h.respondCatalogError(c, err, "create tag")
		return
	}
	response.Ok(c, dto.NewTagResponse(tag))
}

// UpdateTag godoc
// @Summary      更新 Tag
// @Description  更新 Galgame Tag
// @ID           updateTag
// @Tags         tags
// @Accept       json
// @Produce      json
// @Param        id path int true "Tag ID"
// @Param        request body dto.UpdateTagRequest true "更新 Tag 请求"
// @Success      200 {object} dto.TagDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "Tag 不存在"
// @Failure      409 {object} response.ErrorResponse "名称或 slug 已存在"
// @Failure      500 {object} response.ErrorResponse "更新 Tag 失败"
// @Security     BearerAuth
// @Router       /api/v1/tags/{id} [put]
func (h *CatalogHandler) UpdateTag(c *gin.Context) {
	id, ok := parseID(c, "Tag")
	if !ok {
		return
	}
	var req dto.UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	tag, err := h.catalogService.UpdateTag(c.Request.Context(), id, &req)
	if err != nil {
		h.respondCatalogError(c, err, "update tag")
		return
	}
	response.Ok(c, dto.NewTagResponse(tag))
}

// GetTag godoc
// @Summary      查看 Tag 详情
// @Description  按 ID 返回 Tag 详情
// @ID           getTag
// @Tags         tags
// @Produce      json
// @Param        id path int true "Tag ID"
// @Success      200 {object} dto.TagDataResponse "Tag 详情"
// @Failure      400 {object} response.ErrorResponse "Tag ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "Tag 不存在"
// @Failure      500 {object} response.ErrorResponse "查询 Tag 失败"
// @Router       /api/v1/tags/{id} [get]
func (h *CatalogHandler) GetTag(c *gin.Context) {
	id, ok := parseID(c, "Tag")
	if !ok {
		return
	}
	tag, err := h.catalogService.GetTag(c.Request.Context(), id)
	if err != nil {
		h.respondCatalogError(c, err, "get tag")
		return
	}
	response.Ok(c, dto.NewTagResponse(tag))
}

func (h *CatalogHandler) respondCatalogError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrDeveloperNotFound):
		response.Error(c, appErrors.ErrNotFound("开发商不存在"))
	case errors.Is(err, service.ErrTagNotFound):
		response.Error(c, appErrors.ErrNotFound("Tag 不存在"))
	case errors.Is(err, service.ErrGalgameNotFound):
		response.Error(c, appErrors.ErrNotFound("Galgame 不存在"))
	case errors.Is(err, service.ErrDeveloperSlugExists):
		response.Error(c, appErrors.ErrConflict("开发商 slug 已存在"))
	case errors.Is(err, service.ErrTagNameExists):
		response.Error(c, appErrors.ErrConflict("Tag 名称已存在"))
	case errors.Is(err, service.ErrTagSlugExists):
		response.Error(c, appErrors.ErrConflict("Tag slug 已存在"))
	case errors.Is(err, service.ErrGalgameSlugExists):
		response.Error(c, appErrors.ErrConflict("Galgame slug 已存在"))
	case errors.Is(err, service.ErrUnknownTagIDs):
		response.Error(c, appErrors.ErrValidation("Tag 列表包含不存在的 Tag"))
	case errors.Is(err, service.ErrInvalidReleaseDate):
		response.Error(c, appErrors.ErrValidation("发行日期格式不正确"))
	case errors.Is(err, service.ErrInvalidAgeRating):
		response.Error(c, appErrors.ErrValidation("年龄等级不正确"))
	case errors.Is(err, service.ErrInvalidStatus):
		response.Error(c, appErrors.ErrValidation("Galgame 状态不正确"))
	case errors.Is(err, service.ErrInvalidCatalogInput):
		response.Error(c, appErrors.ErrValidation("名称、标题或 slug 不能为空"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("Galgame Catalog 操作失败"))
	}
}

func parseID(c *gin.Context, resource string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation(resource+" ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}

func parseTagIDs(value string) ([]uint, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	ids := make([]uint, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseUint(strings.TrimSpace(part), 10, 0)
		if err != nil || id == 0 {
			return nil, errors.New("invalid tag id")
		}
		ids = append(ids, uint(id))
	}
	return ids, nil
}
