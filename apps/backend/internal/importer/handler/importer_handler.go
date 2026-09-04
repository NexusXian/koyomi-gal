package handler

import (
	"errors"
	"net/http"
	"strconv"

	"backend/internal/importer/dto"
	"backend/internal/importer/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const defaultSearchLimit = 20

type ImporterHandler struct {
	importerService *service.Service
}

func NewImporterHandler(importerService *service.Service) *ImporterHandler {
	return &ImporterHandler{importerService: importerService}
}

// ListImportProviders godoc
// @Summary      查询外部数据源
// @Description  返回当前可用的外部视觉小说数据源列表
// @ID           listImportProviders
// @Tags         adminImport
// @Produce      json
// @Success      200 {object} dto.ImportProvidersResponse "外部数据源列表"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "无权限"
// @Security     BearerAuth
// @Router       /api/v1/admin/import/providers [get]
func (h *ImporterHandler) ListImportProviders(c *gin.Context) {
	response.Ok(c, dto.NewProvidersData(h.importerService.Providers()))
}

// SearchImportGames godoc
// @Summary      搜索外部视觉小说
// @Description  通过外部数据源搜索视觉小说，并附带站内重复状态
// @ID           searchImportGames
// @Tags         adminImport
// @Produce      json
// @Param        provider query string true "数据源，例如 vndb" example(vndb)
// @Param        q query string true "搜索关键词" example(Summer Pockets)
// @Param        limit query int false "返回数量，最大 100" default(20)
// @Success      200 {object} dto.ImportSearchResponse "外部作品搜索结果"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "无权限"
// @Failure      502 {object} response.ErrorResponse "外部数据源查询失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/import/games/search [get]
func (h *ImporterHandler) SearchImportGames(c *gin.Context) {
	var query dto.ImportSearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	if query.Limit <= 0 {
		query.Limit = defaultSearchLimit
	}
	previews, err := h.importerService.Search(c.Request.Context(), query.Provider, query.Q, query.Limit)
	if err != nil {
		h.respondSearchError(c, "search import games", err)
		return
	}
	items := dto.NewImportSearchItems(previews)
	response.Ok(c, dto.ImportSearchData{Items: items, Total: len(items)})
}

// PreviewImportGame godoc
// @Summary      预览外部视觉小说详情
// @Description  返回外部作品完整元数据以及站内重复检测结果
// @ID           previewImportGame
// @Tags         adminImport
// @Produce      json
// @Param        provider path string true "数据源，例如 vndb" example(vndb)
// @Param        external_id path string true "外部作品 ID" example(v17)
// @Success      200 {object} dto.ImportPreviewResponse "外部作品详情与重复状态"
// @Failure      400 {object} response.ErrorResponse "参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "无权限"
// @Failure      404 {object} response.ErrorResponse "外部作品不存在"
// @Failure      502 {object} response.ErrorResponse "外部数据源查询失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/import/games/{provider}/{external_id} [get]
func (h *ImporterHandler) PreviewImportGame(c *gin.Context) {
	providerName := c.Param("provider")
	externalID := c.Param("external_id")
	if providerName == "" || externalID == "" {
		response.Error(c, appErrors.ErrValidation("参数格式不正确"))
		return
	}
	preview, err := h.importerService.Get(c.Request.Context(), providerName, externalID)
	if err != nil {
		h.respondSearchError(c, "preview import game", err)
		return
	}
	response.Ok(c, dto.NewImportPreviewData(preview))
}

// ImportGame godoc
// @Summary      导入外部视觉小说
// @Description  导入单条外部作品；duplicate_action=error 时疑似重复会返回候选列表而不导入
// @ID           importGame
// @Tags         adminImport
// @Accept       json
// @Produce      json
// @Param        request body dto.ImportGameRequest true "导入请求"
// @Success      200 {object} dto.ImportGameResponse "导入结果"
// @Failure      400 {object} response.ErrorResponse "请求格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "无权限"
// @Failure      404 {object} response.ErrorResponse "外部作品或站内作品不存在"
// @Failure      502 {object} response.ErrorResponse "外部数据源查询失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/import/games [post]
func (h *ImporterHandler) ImportGame(c *gin.Context) {
	var request dto.ImportGameRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, appErrors.ErrValidation("请求格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	result, err := h.importerService.Import(c.Request.Context(), service.ImportInput{
		Provider:            request.Provider,
		ExternalID:          request.ExternalID,
		DuplicateAction:     request.DuplicateAction,
		ExistingGalgameID:   request.ExistingGalgameID,
		ForceMetadataUpdate: request.ForceMetadataUpdate,
		CreatedBy:           &userID,
		RecordContribution:  true,
	})
	if err != nil {
		h.respondImportError(c, "import game", err)
		return
	}
	response.Ok(c, dto.NewImportResultData(result))
}

// CreateImportBatch godoc
// @Summary      创建批量导入任务
// @Description  按 VNDB 筛选条件创建异步批量导入任务
// @ID           createImportBatch
// @Tags         adminImport
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateImportBatchRequest true "批量导入请求"
// @Success      200 {object} dto.ImportJobDataResponse "已创建的导入任务"
// @Failure      400 {object} response.ErrorResponse "请求格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "无权限"
// @Security     BearerAuth
// @Router       /api/v1/admin/import/batches [post]
func (h *ImporterHandler) CreateImportBatch(c *gin.Context) {
	var request dto.CreateImportBatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, appErrors.ErrValidation("请求格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	job, err := h.importerService.CreateBatchJob(c.Request.Context(), request.Provider, service.BatchParams{
		MinRating:        request.MinRating,
		MinVoteCount:     request.MinVoteCount,
		FromYear:         request.FromYear,
		ToYear:           request.ToYear,
		OriginalLanguage: request.OriginalLanguage,
		Limit:            request.Limit,
	}, &userID)
	if err != nil {
		h.respondBatchError(c, "create import batch", err)
		return
	}
	response.Ok(c, dto.NewImportJobData(job))
}

// ListImportBatches godoc
// @Summary      查询批量导入任务
// @Description  分页返回批量导入任务列表
// @ID           listImportBatches
// @Tags         adminImport
// @Produce      json
// @Param        status query int false "状态：0 待处理，1 运行中，2 成功，3 失败，4 已取消"
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.ImportJobListResponse "导入任务列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "无权限"
// @Security     BearerAuth
// @Router       /api/v1/admin/import/batches [get]
func (h *ImporterHandler) ListImportBatches(c *gin.Context) {
	var query dto.ImportJobListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = defaultSearchLimit
	}
	jobs, total, err := h.importerService.ListImportJobs(c.Request.Context(), query.Status, query.Page, query.Limit)
	if err != nil {
		logger.Error("list import batches", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询导入任务失败"))
		return
	}
	response.Ok(c, dto.ImportJobListData{
		Items: dto.NewImportJobListItems(jobs),
		Total: total,
		Page:  query.Page,
		Limit: query.Limit,
	})
}

// GetImportBatch godoc
// @Summary      查询批量导入任务详情
// @Description  返回单个导入任务的进度与统计
// @ID           getImportBatch
// @Tags         adminImport
// @Produce      json
// @Param        id path int true "任务 ID"
// @Success      200 {object} dto.ImportJobDataResponse "导入任务详情"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "无权限"
// @Failure      404 {object} response.ErrorResponse "导入任务不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/import/batches/{id} [get]
func (h *ImporterHandler) GetImportBatch(c *gin.Context) {
	id, ok := parseImportID(c, "id")
	if !ok {
		return
	}
	job, err := h.importerService.GetImportJob(c.Request.Context(), int64(id))
	if err != nil {
		if errors.Is(err, service.ErrImportJobNotFound) {
			response.Error(c, appErrors.ErrNotFound("导入任务不存在"))
			return
		}
		logger.Error("get import batch", zap.Int64("job_id", int64(id)), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询导入任务失败"))
		return
	}
	response.Ok(c, dto.NewImportJobData(job))
}

func (h *ImporterHandler) respondBatchError(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, service.ErrProviderNotFound):
		response.Error(c, appErrors.ErrValidation("数据源不受支持"))
	case errors.Is(err, service.ErrBatchUnsupported):
		response.Error(c, appErrors.ErrValidation("数据源不支持批量导入"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("创建批量导入任务失败"))
	}
}

func (h *ImporterHandler) respondSearchError(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, service.ErrProviderNotFound):
		response.Error(c, appErrors.ErrValidation("数据源不受支持"))
	case errors.Is(err, service.ErrExternalGameNotFound):
		response.Error(c, appErrors.ErrNotFound("外部作品不存在"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.New(appErrors.CodeBiz, "外部数据源查询失败", http.StatusBadGateway))
	}
}

func (h *ImporterHandler) respondImportError(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, service.ErrProviderNotFound):
		response.Error(c, appErrors.ErrValidation("数据源不受支持"))
	case errors.Is(err, service.ErrExternalGameNotFound):
		response.Error(c, appErrors.ErrNotFound("外部作品不存在"))
	case errors.Is(err, service.ErrInvalidDuplicateAction):
		response.Error(c, appErrors.ErrValidation("duplicate_action 不受支持"))
	case errors.Is(err, service.ErrExistingGalgameRequired):
		response.Error(c, appErrors.ErrValidation("link_existing 需要 existing_galgame_id"))
	case errors.Is(err, service.ErrExistingGalgameNotFound):
		response.Error(c, appErrors.ErrNotFound("站内作品不存在"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("导入外部作品失败"))
	}
}

// parseImportID parses a numeric path parameter.
func parseImportID(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		response.Error(c, appErrors.ErrValidation("ID 格式不正确"))
		return 0, false
	}
	return uint(value), true
}
