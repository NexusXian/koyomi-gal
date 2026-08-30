package handler

import (
	"backend/internal/health/service"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	healthService *service.HealthService
}

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

func NewHealthHandler(healthService *service.HealthService) *HealthHandler {
	return &HealthHandler{
        healthService: healthService,
    }
}

// HealthCheck godoc
// @Summary      健康检查
// @Description  返回服务存活状态
// @ID           healthCheck
// @Tags         health
// @Produce      json
// @Success      200 {object} HealthResponse "服务正常"
// @Router       /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(200, HealthResponse{Status: "ok"})
}
