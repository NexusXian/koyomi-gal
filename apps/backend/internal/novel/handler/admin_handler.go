package handler

import (
	"errors"

	"backend/internal/middleware"
	"backend/internal/novel/dto"
	"backend/internal/novel/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AdminHandler struct {
	novelService  *service.NovelService
	volumeService *service.VolumeService
}

func NewAdminHandler(
	novelService *service.NovelService,
	volumeService *service.VolumeService,
) *AdminHandler {
	return &AdminHandler{novelService: novelService, volumeService: volumeService}
}

// ListAdminNovels godoc
// @Summary      管理端查询小说列表
// @Description  返回全部状态的小说；需要 novel:review 权限
// @ID           listAdminNovels
// @Tags         admin
// @Produce      json
// @Param        status query int false "状态筛选" Enums(0,1,2,3)
// @Param        keyword query string false "标题 / 原文标题 / 作者关键字"
// @Param        release_status query string false "连载状态" Enums(ongoing,completed,hiatus,cancelled,unknown)
// @Param        sort query string false "排序" Enums(latest,oldest,updated,release,release_asc)
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.NovelListResponse "小说列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询小说失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/novels [get]
func (h *AdminHandler) ListAdminNovels(c *gin.Context) {
	var query dto.AdminNovelQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	novels, total, page, limit, err := h.novelService.ListAdminNovels(c.Request.Context(), &query)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidSort):
			response.Error(c, appErrors.ErrValidation("排序参数不正确"))
		case errors.Is(err, service.ErrInvalidStatus):
			response.Error(c, appErrors.ErrValidation("小说状态不正确"))
		case errors.Is(err, service.ErrInvalidReleaseState):
			response.Error(c, appErrors.ErrValidation("连载状态不正确"))
		default:
			logger.Error("list admin novels", zap.Error(err))
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

// GetAdminNovel godoc
// @Summary      管理端查看小说详情
// @Description  按 ID 返回任意状态的小说详情；需要 novel:review 权限
// @ID           getAdminNovel
// @Tags         admin
// @Produce      json
// @Param        id path int true "Novel ID"
// @Success      200 {object} dto.NovelDataResponse "小说详情"
// @Failure      400 {object} response.ErrorResponse "Novel ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "小说不存在"
// @Failure      500 {object} response.ErrorResponse "查询小说失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/novels/{id} [get]
func (h *AdminHandler) GetAdminNovel(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	novel, err := h.novelService.GetNovel(c.Request.Context(), id)
	if err != nil {
		respondNovelError(c, err, "get admin novel")
		return
	}
	response.Ok(c, dto.NewNovelResponse(novel))
}

// ReviewNovel godoc
// @Summary      管理端审核小说
// @Description  将小说标记为已发布 (1) 或已拒绝 (2)，拒绝时可附原因；需要 novel:review 权限
// @ID           reviewNovel
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Novel ID"
// @Param        request body dto.ReviewNovelRequest true "审核小说请求"
// @Success      200 {object} dto.NovelDataResponse "审核结果"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "小说不存在"
// @Failure      500 {object} response.ErrorResponse "审核小说失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/novels/{id}/review [put]
func (h *AdminHandler) ReviewNovel(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	var req dto.ReviewNovelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	novel, err := h.novelService.ReviewNovel(c.Request.Context(), actorID, id, &req)
	if err != nil {
		respondNovelError(c, err, "review novel")
		return
	}
	response.Ok(c, dto.NewNovelResponse(novel))
}

// ListAdminNovelVolumes godoc
// @Summary      管理端查询小说卷册列表
// @Description  返回全部状态的卷册并附带所属小说标题；需要 novel:review 权限
// @ID           listAdminNovelVolumes
// @Tags         admin
// @Produce      json
// @Param        status query int false "状态筛选" Enums(0,1,2,3)
// @Param        novel_id query int false "小说 ID 筛选"
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.AdminVolumeListResponse "卷册列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询卷册失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/novel-volumes [get]
func (h *AdminHandler) ListAdminNovelVolumes(c *gin.Context) {
	var query dto.AdminVolumeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	var novelID *uint
	if query.NovelID != nil {
		novelID = query.NovelID
	}
	volumes, total, page, limit, err := h.volumeService.ListAdminVolumes(
		c.Request.Context(), query.Status, novelID, query.Page, query.Limit,
	)
	if err != nil {
		respondVolumeError(c, err, "list admin volumes")
		return
	}
	response.Ok(c, dto.AdminVolumeListData{
		Items: dto.NewAdminVolumeListItems(volumes),
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// ReviewNovelVolume godoc
// @Summary      管理端审核小说卷册
// @Description  将卷册标记为已发布 (1) 或已拒绝 (2)，拒绝时可附原因；需要 novel:review 权限
// @ID           reviewNovelVolume
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Volume ID"
// @Param        request body dto.ReviewVolumeRequest true "审核卷册请求"
// @Success      200 {object} dto.VolumeDataResponse "审核结果"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "卷册不存在"
// @Failure      500 {object} response.ErrorResponse "审核卷册失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/novel-volumes/{id}/review [put]
func (h *AdminHandler) ReviewNovelVolume(c *gin.Context) {
	id, ok := parseUintParam(c, "id", "Volume")
	if !ok {
		return
	}
	var req dto.ReviewVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	volume, err := h.volumeService.ReviewVolume(c.Request.Context(), actorID, id, &req)
	if err != nil {
		respondVolumeError(c, err, "review volume")
		return
	}
	response.Ok(c, dto.NewVolumeData(volume))
}
