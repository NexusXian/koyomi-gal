package handler

import (
	"backend/internal/home/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type HomeHandler struct {
	homeService *service.HomeService
}

func NewHomeHandler(homeService *service.HomeService) *HomeHandler {
	return &HomeHandler{homeService: homeService}
}

// GetHome godoc
// @Summary      获取首页数据
// @Description  返回首页轮播、公告、Galgame 和帖子板块
// @ID           getHome
// @Tags         home
// @Produce      json
// @Success      200 {object} dto.HomeResponse "首页数据"
// @Failure      500 {object} response.ErrorResponse "查询首页数据失败"
// @Router       /api/v1/home [get]
func (h *HomeHandler) GetHome(c *gin.Context) {
	data, err := h.homeService.Get(c.Request.Context())
	if err != nil {
		logger.Error("get home", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("查询首页数据失败"))
		return
	}
	response.Ok(c, data)
}
