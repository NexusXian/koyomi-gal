package handler

import (
	"errors"
	"strconv"
	"strings"

	"backend/internal/middleware"
	"backend/internal/novel/dto"
	"backend/internal/novel/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type NovelHandler struct {
	novelService    *service.NovelService
	relationService *service.RelationService
}

func NewNovelHandler(
	novelService *service.NovelService,
	relationService *service.RelationService,
) *NovelHandler {
	return &NovelHandler{novelService: novelService, relationService: relationService}
}

// ListNovels godoc
// @Summary      查询小说列表
// @Description  分页返回已发布的小说，支持关键字、标签、作者、出版社、文库、语言和连载状态筛选
// @ID           listNovels
// @Tags         novels
// @Produce      json
// @Param        keyword query string false "标题 / 原文标题 / 作者关键字"
// @Param        tag_ids query string false "Tag ID 列表（逗号分隔，AND 语义）"
// @Param        author query string false "作者"
// @Param        publisher query string false "出版社"
// @Param        label query string false "文库 / Label"
// @Param        release_status query string false "连载状态" Enums(ongoing,completed,hiatus,cancelled,unknown)
// @Param        language query string false "语言，如 ja、zh-CN"
// @Param        sort query string false "排序" Enums(latest,oldest,updated,release,release_asc)
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.NovelListResponse "小说列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      500 {object} response.ErrorResponse "查询小说失败"
// @Router       /api/v1/novels [get]
func (h *NovelHandler) ListNovels(c *gin.Context) {
	var query dto.NovelQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	tagIDs, err := parseTagIDs(c.Query("tag_ids"))
	if err != nil {
		response.Error(c, appErrors.ErrValidation("tag_ids 参数格式不正确"))
		return
	}
	query.TagIDs = tagIDs
	novels, total, page, limit, err := h.novelService.ListPublishedNovels(c.Request.Context(), &query)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidSort):
			response.Error(c, appErrors.ErrValidation("排序参数不正确"))
		case errors.Is(err, service.ErrInvalidReleaseState):
			response.Error(c, appErrors.ErrValidation("连载状态不正确"))
		default:
			logger.Error("list novels", zap.Error(err))
			response.Error(c, appErrors.ErrInternal("查询小说失败"))
		}
		return
	}
	response.Ok(c, dto.NovelListData{
		Items: dto.NewNovelListItems(novels),
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// GetNovel godoc
// @Summary      查看小说详情
// @Description  按 ID 返回已发布小说详情，包含标签、卷册、关联视觉小说、资源和贡献者概要
// @ID           getNovel
// @Tags         novels
// @Produce      json
// @Param        id path int true "Novel ID"
// @Success      200 {object} dto.NovelDataResponse "小说详情"
// @Failure      400 {object} response.ErrorResponse "Novel ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "小说不存在"
// @Failure      500 {object} response.ErrorResponse "查询小说失败"
// @Router       /api/v1/novels/{id} [get]
func (h *NovelHandler) GetNovel(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	novel, err := h.novelService.GetPublishedNovel(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNovelNotFound) {
			response.Error(c, appErrors.ErrNotFound("小说不存在"))
			return
		}
		logger.Error("get novel", zap.Uint("novel_id", id), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询小说失败"))
		return
	}
	response.Ok(c, dto.NewNovelResponse(novel))
}

// CreateNovel godoc
// @Summary      创建小说
// @Description  创建小说条目；需要 novel:create 权限，默认进入待审核状态
// @ID           createNovel
// @Tags         novels
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateNovelRequest true "创建小说请求"
// @Success      200 {object} dto.NovelDataResponse "创建的小说"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      409 {object} response.ErrorResponse "slug 已存在"
// @Failure      500 {object} response.ErrorResponse "创建小说失败"
// @Security     BearerAuth
// @Router       /api/v1/novels [post]
func (h *NovelHandler) CreateNovel(c *gin.Context) {
	var req dto.CreateNovelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	novel, err := h.novelService.CreateNovel(c.Request.Context(), userID, &req)
	if err != nil {
		respondNovelError(c, err, "create novel")
		return
	}
	response.Ok(c, dto.NewNovelResponse(novel))
}

// UpdateNovel godoc
// @Summary      更新小说
// @Description  全量更新小说资料；需要 novel:update 权限
// @ID           updateNovel
// @Tags         novels
// @Accept       json
// @Produce      json
// @Param        id path int true "Novel ID"
// @Param        request body dto.UpdateNovelRequest true "更新小说请求"
// @Success      200 {object} dto.NovelDataResponse "更新后的小说"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "小说不存在"
// @Failure      409 {object} response.ErrorResponse "slug 已存在"
// @Failure      500 {object} response.ErrorResponse "更新小说失败"
// @Security     BearerAuth
// @Router       /api/v1/novels/{id} [put]
func (h *NovelHandler) UpdateNovel(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	var req dto.UpdateNovelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	novel, err := h.novelService.UpdateNovel(c.Request.Context(), userID, id, &req)
	if err != nil {
		respondNovelError(c, err, "update novel")
		return
	}
	response.Ok(c, dto.NewNovelResponse(novel))
}

// DeleteNovel godoc
// @Summary      删除小说
// @Description  软删除小说及其卷册并清理关联数据；需要 novel:delete 权限
// @ID           deleteNovel
// @Tags         novels
// @Produce      json
// @Param        id path int true "Novel ID"
// @Success      200 {object} response.MessageResponse "删除成功"
// @Failure      400 {object} response.ErrorResponse "Novel ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "小说不存在"
// @Failure      500 {object} response.ErrorResponse "删除小说失败"
// @Security     BearerAuth
// @Router       /api/v1/novels/{id} [delete]
func (h *NovelHandler) DeleteNovel(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	if err := h.novelService.DeleteNovel(c.Request.Context(), id); err != nil {
		respondNovelError(c, err, "delete novel")
		return
	}
	response.OkWithMsg(c, "删除成功")
}

// ListNovelRelations godoc
// @Summary      查看小说关联作品
// @Description  返回小说的原始关联关系列表（双向）；需要 novel:update 权限
// @ID           listNovelRelations
// @Tags         novels
// @Produce      json
// @Param        id path int true "Novel ID"
// @Success      200 {object} dto.RelationListResponse "关联关系列表"
// @Failure      400 {object} response.ErrorResponse "Novel ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询关联作品失败"
// @Security     BearerAuth
// @Router       /api/v1/novels/{id}/relations [get]
func (h *NovelHandler) ListNovelRelations(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	relations, err := h.relationService.ListRelations(c.Request.Context(), id)
	if err != nil {
		logger.Error("list novel relations", zap.Uint("novel_id", id), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询关联作品失败"))
		return
	}
	items := make([]dto.RelationData, 0, len(relations))
	for _, relation := range relations {
		items = append(items, dto.RelationData{
			ID:           relation.ID,
			SourceType:   relation.SourceType,
			SourceID:     relation.SourceID,
			TargetType:   relation.TargetType,
			TargetID:     relation.TargetID,
			RelationType: string(relation.RelationType),
			CreatedBy:    relation.CreatedBy,
			CreatedAt:    relation.CreatedAt,
		})
	}
	response.Ok(c, dto.RelationListData{Items: items, Total: int64(len(items))})
}

// CreateNovelRelation godoc
// @Summary      新增小说关联
// @Description  将小说关联到视觉小说或其他小说；需要 novel:update 权限
// @ID           createNovelRelation
// @Tags         novels
// @Accept       json
// @Produce      json
// @Param        id path int true "Novel ID"
// @Param        request body dto.CreateRelationRequest true "新增关联请求"
// @Success      200 {object} dto.RelationDataResponse "创建的关联关系"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "关联对象不存在"
// @Failure      409 {object} response.ErrorResponse "关联关系已存在"
// @Failure      500 {object} response.ErrorResponse "创建关联失败"
// @Security     BearerAuth
// @Router       /api/v1/novels/{id}/relations [post]
func (h *NovelHandler) CreateNovelRelation(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	var req dto.CreateRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	relation, err := h.relationService.CreateRelation(c.Request.Context(), userID, id, &req)
	if err != nil {
		respondRelationError(c, err, "create relation")
		return
	}
	response.Ok(c, dto.RelationData{
		ID:           relation.ID,
		SourceType:   relation.SourceType,
		SourceID:     relation.SourceID,
		TargetType:   relation.TargetType,
		TargetID:     relation.TargetID,
		RelationType: string(relation.RelationType),
		CreatedBy:    relation.CreatedBy,
		CreatedAt:    relation.CreatedAt,
	})
}

// DeleteNovelRelation godoc
// @Summary      删除小说关联
// @Description  删除小说（任一侧）的关联关系；需要 novel:update 权限
// @ID           deleteNovelRelation
// @Tags         novels
// @Produce      json
// @Param        id path int true "Novel ID"
// @Param        relationId path int true "Relation ID"
// @Success      200 {object} response.MessageResponse "删除成功"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "关联关系不存在"
// @Failure      500 {object} response.ErrorResponse "删除关联失败"
// @Security     BearerAuth
// @Router       /api/v1/novels/{id}/relations/{relationId} [delete]
func (h *NovelHandler) DeleteNovelRelation(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	relationID, ok := parseUintParam(c, "relationId", "Relation")
	if !ok {
		return
	}
	if err := h.relationService.DeleteRelation(c.Request.Context(), id, relationID); err != nil {
		respondRelationError(c, err, "delete relation")
		return
	}
	response.OkWithMsg(c, "删除成功")
}

func respondNovelError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrNovelNotFound):
		response.Error(c, appErrors.ErrNotFound("小说不存在"))
	case errors.Is(err, service.ErrNovelSlugExists):
		response.Error(c, appErrors.ErrConflict("小说 slug 已存在"))
	case errors.Is(err, service.ErrUnknownTagIDs):
		response.Error(c, appErrors.ErrValidation("Tag 列表包含不存在的 Tag"))
	case errors.Is(err, service.ErrInvalidNovelInput):
		response.Error(c, appErrors.ErrValidation("标题或 slug 不能为空"))
	case errors.Is(err, service.ErrInvalidNovelURL):
		response.Error(c, appErrors.ErrValidation("URL 格式不正确"))
	case errors.Is(err, service.ErrInvalidReleaseDate):
		response.Error(c, appErrors.ErrValidation("日期格式不正确"))
	case errors.Is(err, service.ErrInvalidAgeRating):
		response.Error(c, appErrors.ErrValidation("年龄等级不正确"))
	case errors.Is(err, service.ErrInvalidStatus):
		response.Error(c, appErrors.ErrValidation("小说状态不正确"))
	case errors.Is(err, service.ErrInvalidReleaseState):
		response.Error(c, appErrors.ErrValidation("连载状态不正确"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("小说操作失败"))
	}
}

func respondRelationError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrRelationNotFound):
		response.Error(c, appErrors.ErrNotFound("关联关系不存在"))
	case errors.Is(err, service.ErrRelationExists):
		response.Error(c, appErrors.ErrConflict("关联关系已存在"))
	case errors.Is(err, service.ErrInvalidRelationInput):
		response.Error(c, appErrors.ErrValidation("关联参数不正确"))
	case errors.Is(err, service.ErrRelationTargetAbsent):
		response.Error(c, appErrors.ErrNotFound("关联对象不存在"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("关联操作失败"))
	}
}

func parseNovelID(c *gin.Context) (uint, bool) {
	return parseUintParam(c, "id", "Novel")
}

func parseUintParam(c *gin.Context, param, resource string) (uint, bool) {
	parsed, err := strconv.ParseUint(c.Param(param), 10, 0)
	if err != nil || parsed == 0 {
		response.Error(c, appErrors.ErrValidation(resource+" ID 格式不正确"))
		return 0, false
	}
	return uint(parsed), true
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
