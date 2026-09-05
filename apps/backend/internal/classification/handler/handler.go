package handler

import (
	"errors"
	"strconv"

	"backend/internal/classification/dto"
	"backend/internal/classification/model"
	classificationService "backend/internal/classification/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ClassificationHandler struct {
	service *classificationService.Service
}

func NewClassificationHandler(service *classificationService.Service) *ClassificationHandler {
	return &ClassificationHandler{service: service}
}

// StartClassification godoc
// @Summary      启动 AI 年龄分级
// @Description  异步运行 Eino Agent 研究并产出 R18 / 17+ / 15+ / 12+ / 非 R18 / unknown 建议；已在进行中的游戏不会重复入队
// @ID           startClassification
// @Tags         galgame_classification
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.ClassificationDetailResponse "最新分级建议"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确 / 功能未启用 / 正在进行中"
// @Failure      404 {object} response.ErrorResponse "游戏不存在"
// @Failure      500 {object} response.ErrorResponse "启动失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/classification [post]
func (h *ClassificationHandler) StartClassification(c *gin.Context) {
	gameID, ok := parseClassificationID(c)
	if !ok {
		return
	}
	if _, err := h.service.StartClassification(c.Request.Context(), gameID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.respondDetail(c, gameID)
}

// RetryClassification godoc
// @Summary      重试 AI 年龄分级
// @Description  将最新一条 failed 记录重新入队
// @ID           retryClassification
// @Tags         galgame_classification
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.ClassificationDetailResponse "最新分级建议"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确 / 功能未启用"
// @Failure      404 {object} response.ErrorResponse "没有可重试的失败记录"
// @Failure      500 {object} response.ErrorResponse "重试失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/classification/retry [post]
func (h *ClassificationHandler) RetryClassification(c *gin.Context) {
	gameID, ok := parseClassificationID(c)
	if !ok {
		return
	}
	if _, err := h.service.RetryClassification(c.Request.Context(), gameID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.respondDetail(c, gameID)
}

// ListClassifications godoc
// @Summary      查看 AI 分级任务队列
// @Description  分页返回每个游戏最新一条 AI 判断任务，可按状态与游戏名/原名关键字过滤
// @ID           listClassifications
// @Tags         galgame_classification
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Param        status query string false "状态" Enums(queued,processing,pending_review,approved,rejected,failed,cancelled)
// @Param        keyword query string false "游戏标题 / 原名关键字" maxlength(255)
// @Success      200 {object} dto.ClassificationListResponse "队列列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/classification [get]
func (h *ClassificationHandler) ListClassifications(c *gin.Context) {
	var query dto.ClassificationListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	tasks, total, page, limit, err := h.service.ListQueue(
		c.Request.Context(), query.Page, query.Limit, query.Status, query.Keyword,
	)
	if err != nil {
		logger.Error("list classification queue failed", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询失败"))
		return
	}
	items := make([]dto.ClassificationListItem, 0, len(tasks))
	for i := range tasks {
		items = append(items, dto.NewClassificationListItem(&tasks[i]))
	}
	response.Ok(c, dto.ClassificationListData{Items: items, Total: total, Page: page, Limit: limit})
}

// GetClassification godoc
// @Summary      查看 AI 年龄分级
// @Description  返回游戏最新一条 AI 分级建议与全部证据；从未判断过时 classification 为 null
// @ID           getClassification
// @Tags         galgame_classification
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.ClassificationDetailResponse "最新分级建议"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "游戏不存在"
// @Failure      500 {object} response.ErrorResponse "查询失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/classification [get]
func (h *ClassificationHandler) GetClassification(c *gin.Context) {
	gameID, ok := parseClassificationID(c)
	if !ok {
		return
	}
	h.respondDetail(c, gameID)
}

// CancelClassification godoc
// @Summary      取消 AI 年龄分级
// @Description  停止排队中或进行中的 AI 判断任务并标记为 cancelled；已完成的判断无法取消
// @ID           cancelClassification
// @Tags         galgame_classification
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.ClassificationDetailResponse "最新分级建议"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "该游戏还没有 AI 判断记录"
// @Failure      409 {object} response.ErrorResponse "当前没有排队或进行中的任务"
// @Failure      500 {object} response.ErrorResponse "操作失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/classification/cancel [post]
func (h *ClassificationHandler) CancelClassification(c *gin.Context) {
	gameID, ok := parseClassificationID(c)
	if !ok {
		return
	}
	if err := h.service.CancelClassification(c.Request.Context(), gameID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.respondDetail(c, gameID)
}

// ApproveClassification godoc
// @Summary      采用 AI 年龄分级
// @Description  只有此接口会修改游戏正式年龄字段：r18 → age_rating 3，r17 → 5，r15 → 2，r12 → 4，non_r18 → 1；unknown 无法采用
// @ID           approveClassification
// @Tags         galgame_classification
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.ClassificationDetailResponse "最新分级建议"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "没有可采用的 AI 结果"
// @Failure      409 {object} response.ErrorResponse "结果不是待审核状态或为 unknown"
// @Failure      500 {object} response.ErrorResponse "操作失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/classification/approve [post]
func (h *ClassificationHandler) ApproveClassification(c *gin.Context) {
	gameID, ok := parseClassificationID(c)
	if !ok {
		return
	}
	reviewerID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if err := h.service.Approve(c.Request.Context(), gameID, reviewerID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.respondDetail(c, gameID)
}

// RejectClassification godoc
// @Summary      拒绝 AI 年龄分级
// @Description  将待审核建议标记为 rejected，不改动游戏正式数据
// @ID           rejectClassification
// @Tags         galgame_classification
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.ClassificationDetailResponse "最新分级建议"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "没有可拒绝的 AI 结果"
// @Failure      409 {object} response.ErrorResponse "结果不是待审核状态"
// @Failure      500 {object} response.ErrorResponse "操作失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/classification/reject [post]
func (h *ClassificationHandler) RejectClassification(c *gin.Context) {
	gameID, ok := parseClassificationID(c)
	if !ok {
		return
	}
	reviewerID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if err := h.service.Reject(c.Request.Context(), gameID, reviewerID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.respondDetail(c, gameID)
}

// OverrideClassification godoc
// @Summary      人工覆盖 AI 年龄分级
// @Description  用人工结论替换最新建议并回到待审核状态，仍需 Approve 才会写入正式数据
// @ID           overrideClassification
// @Tags         galgame_classification
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        body body dto.OverrideClassificationRequest true "人工分级结论"
// @Success      200 {object} dto.ClassificationDetailResponse "最新分级建议"
// @Failure      400 {object} response.ErrorResponse "参数不正确"
// @Failure      404 {object} response.ErrorResponse "游戏还没有 AI 判断记录"
// @Failure      409 {object} response.ErrorResponse "已通过或进行中的结果无法覆盖"
// @Failure      500 {object} response.ErrorResponse "操作失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/classification/override [post]
func (h *ClassificationHandler) OverrideClassification(c *gin.Context) {
	gameID, ok := parseClassificationID(c)
	if !ok {
		return
	}
	var request dto.OverrideClassificationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, appErrors.ErrValidation("classification 必须是 r18 / r17 / r15 / r12 / non_r18 / unknown"))
		return
	}
	reason := request.Reason
	if reason == "" {
		reason = "管理员人工指定"
	}
	if err := h.service.Override(c.Request.Context(), gameID,
		model.ClassificationValue(request.Classification), reason); err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.respondDetail(c, gameID)
}

// BatchClassification godoc
// @Summary      批量启动 AI 年龄分级
// @Description  一次最多 500 款游戏，每款独立入队
// @ID           batchClassification
// @Tags         galgame_classification
// @Produce      json
// @Param        body body dto.BatchClassificationRequest true "游戏 ID 列表"
// @Success      200 {object} dto.BatchResponse "批量入队结果"
// @Failure      400 {object} response.ErrorResponse "参数不正确 / 功能未启用"
// @Failure      500 {object} response.ErrorResponse "操作失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/classification/batch [post]
func (h *ClassificationHandler) BatchClassification(c *gin.Context) {
	var request dto.BatchClassificationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, appErrors.ErrValidation("game_ids 必须为 1-500 个正整数"))
		return
	}
	result, err := h.service.BatchStart(c.Request.Context(), request.GameIDs)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Ok(c, dto.BatchData{
		Enqueued:       result.Enqueued,
		AlreadyRunning: result.AlreadyRunning,
		Failed:         toBatchItems(result.Failed),
	})
}

// BatchApproveClassification godoc
// @Summary      批量采用 AI 高置信度结果
// @Description  仅采用 confidence >= 0.7 且无证据冲突且结论明确的待审核建议；其余保持待审核
// @ID           batchApproveClassification
// @Tags         galgame_classification
// @Produce      json
// @Param        body body dto.BatchClassificationRequest true "游戏 ID 列表"
// @Success      200 {object} dto.BatchResponse "批量采用结果"
// @Failure      400 {object} response.ErrorResponse "参数不正确"
// @Failure      500 {object} response.ErrorResponse "操作失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/classification/batch-approve [post]
func (h *ClassificationHandler) BatchApproveClassification(c *gin.Context) {
	var request dto.BatchClassificationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, appErrors.ErrValidation("game_ids 必须为 1-500 个正整数"))
		return
	}
	reviewerID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	result, err := h.service.BatchApprove(c.Request.Context(), request.GameIDs, reviewerID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Ok(c, dto.BatchData{
		Approved: result.Approved,
		Skipped:  toBatchItems(result.Skipped),
		Failed:   toBatchItems(result.Failed),
	})
}

func (h *ClassificationHandler) respondDetail(c *gin.Context, gameID uint) {
	game, row, err := h.service.GetDetail(c.Request.Context(), gameID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Ok(c, dto.ClassificationDetailData{
		Game: dto.GameSummary{
			ID:             game.ID,
			Title:          game.Title,
			OriginalTitle:  game.OriginalTitle,
			CoverURL:       game.CoverURL,
			AgeRating:      game.AgeRating,
			CoverSensitive: game.CoverSensitive,
		},
		Classification: dto.NewClassificationResponse(row),
	})
}

func (h *ClassificationHandler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, classificationService.ErrAgentDisabled):
		response.Error(c, appErrors.ErrBadRequest("AI 分级功能未启用"))
	case errors.Is(err, classificationService.ErrAlreadyRunning):
		response.Error(c, appErrors.ErrConflict("该游戏已有 AI 判断正在进行"))
	case errors.Is(err, classificationService.ErrGameNotFound),
		errors.Is(err, classificationService.ErrNoClassification),
		errors.Is(err, classificationService.ErrNotFailed):
		response.Error(c, appErrors.ErrNotFound(err.Error()))
	case errors.Is(err, classificationService.ErrInvalidState):
		response.Error(c, appErrors.ErrConflict(err.Error()))
	default:
		logger.Error("classification operation failed", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("操作失败"))
	}
}

func parseClassificationID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("Galgame ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}

func toBatchItems(failures []classificationService.BatchFailure) []dto.BatchItem {
	items := make([]dto.BatchItem, 0, len(failures))
	for _, failure := range failures {
		items = append(items, dto.BatchItem{GameID: failure.GameID, Reason: failure.Reason})
	}
	return items
}
