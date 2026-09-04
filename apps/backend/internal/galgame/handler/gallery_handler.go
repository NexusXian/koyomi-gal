package handler

import (
	"errors"
	"strconv"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/service"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GalleryHandler struct {
	galleryService *service.GalleryService
}

func NewGalleryHandler(galleryService *service.GalleryService) *GalleryHandler {
	return &GalleryHandler{galleryService: galleryService}
}

// ListGalgameGallery godoc
// @Summary      查询 Galgame 游戏画面
// @Description  返回已发布 Galgame 的游戏截图 / CG 画廊，按 sort_order 排序，不分页
// @ID           listGalgameGallery
// @Tags         galgames
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.GalleryListResponse "游戏画面列表"
// @Failure      400 {object} response.ErrorResponse "Galgame ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "查询游戏画面失败"
// @Router       /api/v1/galgames/{id}/gallery [get]
func (h *GalleryHandler) ListGalgameGallery(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	data, err := h.galleryService.ListPublishedGallery(c.Request.Context(), id)
	if err != nil {
		h.respondGalleryError(c, err, "list galgame gallery", zap.Uint("galgame_id", id))
		return
	}
	response.Ok(c, data)
}

// ListAdminGalgameGallery godoc
// @Summary      管理端查询 Galgame 游戏画面
// @Description  返回任意状态 Galgame 的游戏画面；需要 galgame_gallery:manage 权限
// @ID           listAdminGalgameGallery
// @Tags         admin
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Success      200 {object} dto.GalleryListResponse "游戏画面列表"
// @Failure      400 {object} response.ErrorResponse "Galgame ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      500 {object} response.ErrorResponse "查询游戏画面失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/gallery [get]
func (h *GalleryHandler) ListAdminGalgameGallery(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	data, err := h.galleryService.ListAdminGallery(c.Request.Context(), id)
	if err != nil {
		h.respondGalleryError(c, err, "list admin galgame gallery", zap.Uint("galgame_id", id))
		return
	}
	response.Ok(c, data)
}

// CreateGalgameGalleryImage godoc
// @Summary      添加游戏画面
// @Description  把已有图片资源（image_assets）加入 Galgame 画廊，追加到末尾；需要 galgame_gallery:manage 权限
// @ID           createGalgameGalleryImage
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        request body dto.CreateGalleryImageRequest true "添加游戏画面请求"
// @Success      200 {object} dto.GalleryDataResponse "添加成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "Galgame 或图片资源不存在"
// @Failure      409 {object} response.ErrorResponse "图片已在画廊中或数量已达上限"
// @Failure      500 {object} response.ErrorResponse "添加游戏画面失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/gallery [post]
func (h *GalleryHandler) CreateGalgameGalleryImage(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	var req dto.CreateGalleryImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	data, err := h.galleryService.CreateGalleryImage(c.Request.Context(), id, actorID, &req)
	if err != nil {
		h.respondGalleryError(c, err, "create gallery image", zap.Uint("galgame_id", id))
		return
	}
	response.Ok(c, data)
}

// UpdateGalgameGalleryImage godoc
// @Summary      编辑游戏画面
// @Description  更新游戏画面的标题、描述、类型和剧透标记；需要 galgame_gallery:manage 权限
// @ID           updateGalgameGalleryImage
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        galleryId path int true "画廊图片 ID"
// @Param        request body dto.UpdateGalleryImageRequest true "编辑游戏画面请求"
// @Success      200 {object} dto.GalleryDataResponse "更新成功"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "画廊图片不存在"
// @Failure      500 {object} response.ErrorResponse "更新游戏画面失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/gallery/{galleryId} [patch]
func (h *GalleryHandler) UpdateGalgameGalleryImage(c *gin.Context) {
	id, galleryID, ok := parseGalleryIDs(c)
	if !ok {
		return
	}
	var req dto.UpdateGalleryImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	data, err := h.galleryService.UpdateGalleryImage(c.Request.Context(), id, galleryID, &req, actorID)
	if err != nil {
		h.respondGalleryError(c, err, "update gallery image", zap.Uint("galgame_id", id), zap.Uint("gallery_id", galleryID))
		return
	}
	response.Ok(c, data)
}

// DeleteGalgameGalleryImage godoc
// @Summary      删除游戏画面
// @Description  仅删除画廊关联关系，不删除图片资源和 R2 对象；需要 galgame_gallery:manage 权限
// @ID           deleteGalgameGalleryImage
// @Tags         admin
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        galleryId path int true "画廊图片 ID"
// @Success      200 {object} response.MessageResponse "游戏画面已删除"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "画廊图片不存在"
// @Failure      500 {object} response.ErrorResponse "删除游戏画面失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/gallery/{galleryId} [delete]
func (h *GalleryHandler) DeleteGalgameGalleryImage(c *gin.Context) {
	id, galleryID, ok := parseGalleryIDs(c)
	if !ok {
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if err := h.galleryService.DeleteGalleryImage(c.Request.Context(), id, galleryID, actorID); err != nil {
		h.respondGalleryError(c, err, "delete gallery image", zap.Uint("galgame_id", id), zap.Uint("gallery_id", galleryID))
		return
	}
	response.OkWithMsg(c, "游戏画面已删除")
}

// ReorderGalgameGallery godoc
// @Summary      调整游戏画面排序
// @Description  按传入的完整 ID 顺序重写 sort_order（0..n-1）；ID 集合必须恰好覆盖该 Galgame 的全部画廊图片；需要 galgame_gallery:manage 权限
// @ID           reorderGalgameGallery
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        request body dto.ReorderGalleryRequest true "排序请求"
// @Success      200 {object} response.MessageResponse "排序已保存"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      422 {object} response.ErrorResponse "ID 集合与画廊不匹配"
// @Failure      500 {object} response.ErrorResponse "保存排序失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/gallery/order [put]
func (h *GalleryHandler) ReorderGalgameGallery(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	var req dto.ReorderGalleryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if err := h.galleryService.ReorderGallery(c.Request.Context(), id, &req, actorID); err != nil {
		h.respondGalleryError(c, err, "reorder gallery", zap.Uint("galgame_id", id))
		return
	}
	response.OkWithMsg(c, "排序已保存")
}

func (h *GalleryHandler) respondGalleryError(c *gin.Context, err error, operation string, fields ...zap.Field) {
	switch {
	case errors.Is(err, service.ErrGalleryGalgameNotFound):
		response.Error(c, appErrors.ErrNotFound("Galgame 不存在"))
	case errors.Is(err, service.ErrGalleryImageNotFound):
		response.Error(c, appErrors.ErrNotFound("画廊图片不存在"))
	case errors.Is(err, service.ErrGalleryAssetNotFound):
		response.Error(c, appErrors.ErrNotFound("图片资源不存在或不可用"))
	case errors.Is(err, service.ErrGalleryAssetDuplicate):
		response.Error(c, appErrors.ErrConflict("该图片已在画廊中"))
	case errors.Is(err, service.ErrGalleryLimitExceeded):
		response.Error(c, appErrors.ErrConflict("游戏画面数量已达上限"))
	case errors.Is(err, service.ErrGalleryInvalidReorder):
		response.Error(c, appErrors.ErrValidation("排序 ID 集合必须恰好覆盖该 Galgame 的全部游戏画面"))
	default:
		logger.Error(operation, append(fields, zap.Error(err))...)
		response.Error(c, appErrors.ErrInternal("游戏画面操作失败"))
	}
}

func parseGalleryIDs(c *gin.Context) (uint, uint, bool) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return 0, 0, false
	}
	galleryID, err := strconv.ParseUint(c.Param("galleryId"), 10, 0)
	if err != nil || galleryID == 0 {
		response.Error(c, appErrors.ErrValidation("画廊图片 ID 格式不正确"))
		return 0, 0, false
	}
	return id, uint(galleryID), true
}
