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

type VolumeHandler struct {
	volumeService *service.VolumeService
}

func NewVolumeHandler(volumeService *service.VolumeService) *VolumeHandler {
	return &VolumeHandler{volumeService: volumeService}
}

// ListNovelVolumes godoc
// @Summary      查询小说卷册列表
// @Description  分页返回已发布小说的已发布卷册
// @ID           listNovelVolumes
// @Tags         novels
// @Produce      json
// @Param        id path int true "Novel ID"
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Success      200 {object} dto.VolumeListResponse "卷册列表"
// @Failure      400 {object} response.ErrorResponse "参数格式不正确"
// @Failure      404 {object} response.ErrorResponse "小说不存在"
// @Failure      500 {object} response.ErrorResponse "查询卷册失败"
// @Router       /api/v1/novels/{id}/volumes [get]
func (h *VolumeHandler) ListNovelVolumes(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	var query dto.VolumeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	volumes, total, page, limit, err := h.volumeService.ListVolumes(c.Request.Context(), id, true, query.Page, query.Limit)
	if err != nil {
		respondVolumeError(c, err, "list volumes")
		return
	}
	response.Ok(c, dto.VolumeListData{
		Items: dto.NewVolumeListData(volumes),
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// GetNovelVolume godoc
// @Summary      查看小说卷册详情
// @Description  返回已发布小说的单个已发布卷册
// @ID           getNovelVolume
// @Tags         novels
// @Produce      json
// @Param        id path int true "Novel ID"
// @Param        volumeId path int true "Volume ID"
// @Success      200 {object} dto.VolumeDataResponse "卷册详情"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "卷册不存在"
// @Failure      500 {object} response.ErrorResponse "查询卷册失败"
// @Router       /api/v1/novels/{id}/volumes/{volumeId} [get]
func (h *VolumeHandler) GetNovelVolume(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	volumeID, ok := parseUintParam(c, "volumeId", "Volume")
	if !ok {
		return
	}
	volume, err := h.volumeService.GetVolume(c.Request.Context(), id, volumeID, true)
	if err != nil {
		respondVolumeError(c, err, "get volume")
		return
	}
	response.Ok(c, dto.NewVolumeData(volume))
}

// CreateNovelVolume godoc
// @Summary      新增小说卷册
// @Description  为小说添加卷册；需要 novel:update 权限，默认进入待审核状态
// @ID           createNovelVolume
// @Tags         novels
// @Accept       json
// @Produce      json
// @Param        id path int true "Novel ID"
// @Param        request body dto.CreateVolumeRequest true "新增卷册请求"
// @Success      200 {object} dto.VolumeDataResponse "创建的卷册"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "小说不存在"
// @Failure      500 {object} response.ErrorResponse "新增卷册失败"
// @Security     BearerAuth
// @Router       /api/v1/novels/{id}/volumes [post]
func (h *VolumeHandler) CreateNovelVolume(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	var req dto.CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	volume, err := h.volumeService.CreateVolume(c.Request.Context(), userID, id, &req)
	if err != nil {
		respondVolumeError(c, err, "create volume")
		return
	}
	response.Ok(c, dto.NewVolumeData(volume))
}

// UpdateNovelVolume godoc
// @Summary      更新小说卷册
// @Description  全量更新卷册资料；卷册必须属于该小说，需要 novel:update 权限
// @ID           updateNovelVolume
// @Tags         novels
// @Accept       json
// @Produce      json
// @Param        id path int true "Novel ID"
// @Param        volumeId path int true "Volume ID"
// @Param        request body dto.UpdateVolumeRequest true "更新卷册请求"
// @Success      200 {object} dto.VolumeDataResponse "更新后的卷册"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "卷册不存在"
// @Failure      500 {object} response.ErrorResponse "更新卷册失败"
// @Security     BearerAuth
// @Router       /api/v1/novels/{id}/volumes/{volumeId} [put]
func (h *VolumeHandler) UpdateNovelVolume(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	volumeID, ok := parseUintParam(c, "volumeId", "Volume")
	if !ok {
		return
	}
	var req dto.UpdateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	volume, err := h.volumeService.UpdateVolume(c.Request.Context(), userID, id, volumeID, &req)
	if err != nil {
		respondVolumeError(c, err, "update volume")
		return
	}
	response.Ok(c, dto.NewVolumeData(volume))
}

// DeleteNovelVolume godoc
// @Summary      删除小说卷册
// @Description  软删除卷册；卷册必须属于该小说，需要 novel:update 权限
// @ID           deleteNovelVolume
// @Tags         novels
// @Produce      json
// @Param        id path int true "Novel ID"
// @Param        volumeId path int true "Volume ID"
// @Success      200 {object} response.MessageResponse "删除成功"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "卷册不存在"
// @Failure      500 {object} response.ErrorResponse "删除卷册失败"
// @Security     BearerAuth
// @Router       /api/v1/novels/{id}/volumes/{volumeId} [delete]
func (h *VolumeHandler) DeleteNovelVolume(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	volumeID, ok := parseUintParam(c, "volumeId", "Volume")
	if !ok {
		return
	}
	if err := h.volumeService.DeleteVolume(c.Request.Context(), id, volumeID); err != nil {
		respondVolumeError(c, err, "delete volume")
		return
	}
	response.OkWithMsg(c, "删除成功")
}

// ReorderNovelVolumes godoc
// @Summary      排序小说卷册
// @Description  按传入 ID 顺序重写卷册 sort_order；ID 集合必须与小说卷册完全一致，需要 novel:update 权限
// @ID           reorderNovelVolumes
// @Tags         novels
// @Accept       json
// @Produce      json
// @Param        id path int true "Novel ID"
// @Param        request body dto.ReorderVolumesRequest true "卷册排序请求"
// @Success      200 {object} response.MessageResponse "排序成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "小说不存在"
// @Failure      500 {object} response.ErrorResponse "排序卷册失败"
// @Security     BearerAuth
// @Router       /api/v1/novels/{id}/volumes/order [put]
func (h *VolumeHandler) ReorderNovelVolumes(c *gin.Context) {
	id, ok := parseNovelID(c)
	if !ok {
		return
	}
	var req dto.ReorderVolumesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	if err := h.volumeService.ReorderVolumes(c.Request.Context(), id, &req); err != nil {
		respondVolumeError(c, err, "reorder volumes")
		return
	}
	response.OkWithMsg(c, "排序成功")
}

func respondVolumeError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrVolumeNotFound):
		response.Error(c, appErrors.ErrNotFound("卷册不存在"))
	case errors.Is(err, service.ErrNovelNotFound):
		response.Error(c, appErrors.ErrNotFound("小说不存在"))
	case errors.Is(err, service.ErrInvalidVolumeInput):
		response.Error(c, appErrors.ErrValidation("卷册参数不正确"))
	case errors.Is(err, service.ErrInvalidVolumeURL):
		response.Error(c, appErrors.ErrValidation("URL 格式不正确"))
	case errors.Is(err, service.ErrInvalidISBN):
		response.Error(c, appErrors.ErrValidation("ISBN 格式不正确"))
	case errors.Is(err, service.ErrInvalidVolumeStatus):
		response.Error(c, appErrors.ErrValidation("卷册状态不正确"))
	case errors.Is(err, service.ErrInvalidVolumeReorder):
		response.Error(c, appErrors.ErrValidation("卷册排序参数不正确"))
	case errors.Is(err, service.ErrInvalidReleaseDate):
		response.Error(c, appErrors.ErrValidation("日期格式不正确"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("卷册操作失败"))
	}
}
