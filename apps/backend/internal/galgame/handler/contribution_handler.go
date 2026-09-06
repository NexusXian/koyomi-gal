package handler

import (
	"errors"
	"strconv"

	"backend/internal/contribution/service"
	"backend/internal/galgame/dto"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ContributionHandler struct {
	service *service.ContributionService
	levels  service.LevelSummarizer
}

func NewContributionHandler(service *service.ContributionService, levels ...service.LevelSummarizer) *ContributionHandler {
	handler := &ContributionHandler{service: service}
	if len(levels) > 0 {
		handler.levels = levels[0]
	}
	return handler
}

// ListGalgameContributors godoc
// @Summary      查看 Galgame 贡献者
// @Description  按贡献次数和最近贡献时间分页返回已发布 Galgame 的贡献者
// @ID           listGalgameContributors
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.ContributorListResponse "贡献者列表"
// @Failure      400 {object} response.ErrorResponse "参数格式不正确"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "查询贡献者失败"
// @Router       /api/v1/galgames/{id}/contributors [get]
func (h *ContributionHandler) ListGalgameContributors(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("Galgame ID 格式不正确"))
		return
	}
	var query dto.ContributorQuery
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
		pageSize = 20
	}
	contributors, total, err := h.service.ListGalgameContributorRows(c.Request.Context(), uint(id), page, pageSize)
	if err != nil {
		if errors.Is(err, service.ErrGalgameNotFound) {
			response.Error(c, appErrors.ErrNotFound("Galgame 不存在"))
			return
		}
		logger.Error("list galgame contributors", zap.Uint64("galgame_id", id), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询贡献者失败"))
		return
	}
	items := dto.NewContributorData(contributors)
	if h.levels != nil && len(items) > 0 {
		userIDs := make([]uint, 0, len(items))
		for i := range items {
			userIDs = append(userIDs, items[i].UserID)
		}
		if summaries, summaryErr := h.levels.Summaries(c.Request.Context(), userIDs); summaryErr == nil {
			for i := range items {
				if summary, ok := summaries[items[i].UserID]; ok {
					summary := summary
					items[i].Level = &summary
				}
			}
		} else {
			logger.Error("load contributor levels", zap.Uint64("galgame_id", id), zap.Error(summaryErr))
		}
	}
	response.Ok(c, dto.ContributorListData{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
