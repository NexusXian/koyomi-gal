package handler

import (
	"errors"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ContributionHandler struct {
	service *service.ContributionService
}

func NewContributionHandler(service *service.ContributionService) *ContributionHandler {
	return &ContributionHandler{service: service}
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
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	var query dto.ContributorQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	data, err := h.service.ListContributorsByGalgameID(c.Request.Context(), id, query.Page, query.PageSize)
	if err != nil {
		if errors.Is(err, service.ErrGalgameNotFound) {
			response.Error(c, appErrors.ErrNotFound("Galgame 不存在"))
			return
		}
		logger.Error("list galgame contributors", zap.Uint("galgame_id", id), zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询贡献者失败"))
		return
	}
	response.Ok(c, data)
}
