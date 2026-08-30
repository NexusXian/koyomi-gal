package handler

import (
	"errors"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/resource/dto"
	"backend/internal/resource/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ReportHandler struct {
	reportService *service.ReportService
}

func NewReportHandler(reportService *service.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

// CreateReport godoc
// @Summary      举报资源
// @Description  登录用户举报已发布资源；同一用户对同一资源只能举报一次
// @ID           createResourceReport
// @Tags         resources
// @Accept       json
// @Produce      json
// @Param        id path int true "资源 ID"
// @Param        request body dto.CreateResourceReportRequest true "举报请求"
// @Success      200 {object} dto.ResourceReportDataResponse "举报成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      404 {object} response.ErrorResponse "资源不存在"
// @Failure      409 {object} response.ErrorResponse "已举报过该资源"
// @Failure      500 {object} response.ErrorResponse "举报失败"
// @Security     BearerAuth
// @Router       /api/v1/resources/{id}/reports [post]
func (h *ReportHandler) CreateReport(c *gin.Context) {
	resourceID, ok := parseResourceID(c, "资源")
	if !ok {
		return
	}
	var req dto.CreateResourceReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	report, err := h.reportService.Create(c.Request.Context(), userID, resourceID, &req)
	if err != nil {
		h.respondReportError(c, err, "create resource report")
		return
	}
	response.Ok(c, dto.NewResourceReportData(report))
}

// ListReports godoc
// @Summary      管理端查询资源举报
// @Description  分页查询资源举报；需要 resource_report:list 权限
// @ID           listResourceReports
// @Tags         admin
// @Produce      json
// @Param        status query int false "状态过滤：0 待处理，1 已解决，2 已驳回"
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.ResourceReportListResponse "举报列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询举报失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/resource-reports [get]
func (h *ReportHandler) ListReports(c *gin.Context) {
	var query struct {
		Status *int16 `form:"status" binding:"omitempty,oneof=0 1 2"`
		Page   int    `form:"page" binding:"omitempty,min=1,max=1000000"`
		Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	reports, total, page, limit, err := h.reportService.List(
		c.Request.Context(), query.Status, query.Page, query.Limit,
	)
	if err != nil {
		h.respondReportError(c, err, "list resource reports")
		return
	}
	response.Ok(c, dto.ResourceReportListData{
		Items: dto.NewResourceReportListItems(reports),
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// HandleReport godoc
// @Summary      管理端处理资源举报
// @Description  将举报标记为已解决或已驳回并记录处理人；需要 resource_report:handle 权限
// @ID           handleResourceReport
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "举报 ID"
// @Param        request body dto.HandleResourceReportRequest true "处理举报请求"
// @Success      200 {object} dto.ResourceReportDataResponse "处理结果"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "举报不存在"
// @Failure      500 {object} response.ErrorResponse "处理举报失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/resource-reports/{id}/handle [put]
func (h *ReportHandler) HandleReport(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("举报 ID 格式不正确"))
		return
	}
	var req dto.HandleResourceReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	adminID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	report, err := h.reportService.Handle(c.Request.Context(), adminID, uint(id), &req)
	if err != nil {
		h.respondReportError(c, err, "handle resource report")
		return
	}
	response.Ok(c, dto.NewResourceReportData(report))
}

func (h *ReportHandler) respondReportError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrReportNotFound):
		response.Error(c, appErrors.ErrNotFound("举报不存在"))
	case errors.Is(err, service.ErrResourceNotFound):
		response.Error(c, appErrors.ErrNotFound("资源不存在"))
	case errors.Is(err, service.ErrAlreadyReported):
		response.Error(c, appErrors.ErrConflict("已举报过该资源"))
	case errors.Is(err, service.ErrInvalidReportReason):
		response.Error(c, appErrors.ErrValidation("举报原因不正确"))
	case errors.Is(err, service.ErrInvalidReportStatus):
		response.Error(c, appErrors.ErrValidation("举报状态不正确"))
	case errors.Is(err, service.ErrInvalidReportHandle):
		response.Error(c, appErrors.ErrValidation("处理状态只能是 1 已解决或 2 已驳回"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("资源举报操作失败"))
	}
}
