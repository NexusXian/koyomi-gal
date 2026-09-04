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
// @Description  返回已发布 Galgame 的已通过（published）游戏截图 / CG 画廊，按 sort_order 排序，不分页
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
// @Description  返回任意状态 Galgame 的全部游戏画面（含待审核 / 已拒绝）；需要 galgame_gallery:manage 权限
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
// @Description  添加本站图片资源（asset_id）或外部图片链接（external_url），二选一；创建后进入待审核状态，需审核通过后才会公开展示；需要 galgame_gallery:manage 权限
// @ID           createGalgameGalleryImage
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        request body dto.CreateGalleryImageRequest true "添加游戏画面请求"
// @Success      200 {object} dto.GalleryDataResponse "添加成功（status=0 待审核）"
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

// BatchCreateGalgameGalleryImages godoc
// @Summary      批量导入外部图片链接
// @Description  一次导入多个外部图片 URL，重复（本批次内或画廊中已存在）自动跳过，无效 URL 计入 failed；全部创建为待审核；需要 galgame_gallery:manage 权限
// @ID           batchCreateGalgameGalleryImages
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Galgame ID"
// @Param        request body dto.BatchCreateGalleryRequest true "批量导入请求"
// @Success      200 {object} dto.GalleryBatchResponse "导入结果统计"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "Galgame 不存在"
// @Failure      409 {object} response.ErrorResponse "数量已达上限"
// @Failure      500 {object} response.ErrorResponse "批量导入失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/galgames/{id}/gallery/batch [post]
func (h *GalleryHandler) BatchCreateGalgameGalleryImages(c *gin.Context) {
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	var req dto.BatchCreateGalleryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	data, err := h.galleryService.BatchCreateGalleryImages(c.Request.Context(), id, actorID, &req)
	if err != nil {
		h.respondGalleryError(c, err, "batch create gallery images", zap.Uint("galgame_id", id))
		return
	}
	response.Ok(c, data)
}

// ListGalleryReviews godoc
// @Summary      插图审核队列
// @Description  跨 Galgame 的游戏画面审核列表，可按状态 / 游戏 / 来源过滤；需要 galgame_gallery:review 权限
// @ID           listGalleryReviews
// @Tags         admin
// @Produce      json
// @Param        status query int false "审核状态：0 待审核 / 1 已通过 / 2 已拒绝"
// @Param        galgame_id query int false "按 Galgame 过滤"
// @Param        source_type query int false "来源：0 本站上传 / 1 外部链接"
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量（1-100）" default(20)
// @Success      200 {object} dto.GalleryReviewListResponse "审核列表"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询审核列表失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/gallery-images [get]
func (h *GalleryHandler) ListGalleryReviews(c *gin.Context) {
	var query dto.GalleryReviewListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	data, err := h.galleryService.ListGalleryReviews(c.Request.Context(), service.GalleryReviewQuery{
		Status:     query.Status,
		GalgameID:  query.GalgameID,
		SourceType: query.SourceType,
		Page:       query.Page,
		Limit:      query.Limit,
	})
	if err != nil {
		h.respondGalleryError(c, err, "list gallery reviews")
		return
	}
	response.Ok(c, data)
}

// ApproveGalleryImage godoc
// @Summary      通过游戏画面审核
// @Description  将游戏画面标记为已通过（published），游戏已发布时会为提交者记录一次贡献；需要 galgame_gallery:review 权限
// @ID           approveGalleryImage
// @Tags         admin
// @Produce      json
// @Param        id path int true "画廊图片 ID"
// @Success      200 {object} dto.GalleryDataResponse "审核后的游戏画面"
// @Failure      400 {object} response.ErrorResponse "ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "画廊图片不存在"
// @Failure      500 {object} response.ErrorResponse "审核失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/gallery-images/{id}/approve [post]
func (h *GalleryHandler) ApproveGalleryImage(c *gin.Context) {
	h.reviewGalleryImage(c, true)
}

// RejectGalleryImage godoc
// @Summary      拒绝游戏画面审核
// @Description  将游戏画面标记为已拒绝（rejected），可附带拒绝理由；需要 galgame_gallery:review 权限
// @ID           rejectGalleryImage
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "画廊图片 ID"
// @Param        request body dto.ReviewGalleryImageRequest false "拒绝理由"
// @Success      200 {object} dto.GalleryDataResponse "审核后的游戏画面"
// @Failure      400 {object} response.ErrorResponse "ID 或请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "画廊图片不存在"
// @Failure      500 {object} response.ErrorResponse "审核失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/gallery-images/{id}/reject [post]
func (h *GalleryHandler) RejectGalleryImage(c *gin.Context) {
	h.reviewGalleryImage(c, false)
}

func (h *GalleryHandler) reviewGalleryImage(c *gin.Context, approve bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("画廊图片 ID 格式不正确"))
		return
	}
	reason := ""
	if !approve {
		var req dto.ReviewGalleryImageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
			return
		}
		reason = req.Reason
	}
	adminID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	if _, err := h.galleryService.ReviewGalleryImages(c.Request.Context(), service.ReviewGalleryImagesInput{
		IDs:     []uint{uint(id)},
		Approve: approve,
		Reason:  reason,
		AdminID: adminID,
	}); err != nil {
		h.respondGalleryError(c, err, "review gallery image", zap.Uint("gallery_id", uint(id)))
		return
	}
	image, err := h.galleryService.FindGalleryImageForReview(c.Request.Context(), uint(id))
	if err != nil {
		h.respondGalleryError(c, err, "reload gallery image", zap.Uint("gallery_id", uint(id)))
		return
	}
	if image == nil {
		response.Error(c, appErrors.ErrNotFound("画廊图片不存在"))
		return
	}
	response.Ok(c, image)
}

// BatchReviewGalleryImages godoc
// @Summary      批量审核游戏画面
// @Description  一次通过或拒绝多个游戏画面，事务执行；action=approve 或 reject，reject 可附带理由；需要 galgame_gallery:review 权限
// @ID           batchReviewGalleryImages
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body dto.BatchReviewGalleryRequest true "批量审核请求"
// @Success      200 {object} dto.GalleryReviewBatchResponse "实际审核数量"
// @Failure      400 {object} response.ErrorResponse "请求参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "批量审核失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/gallery-images/batch-review [post]
func (h *GalleryHandler) BatchReviewGalleryImages(c *gin.Context) {
	var req dto.BatchReviewGalleryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("请求参数格式不正确"))
		return
	}
	adminID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return
	}
	reviewed, err := h.galleryService.ReviewGalleryImages(c.Request.Context(), service.ReviewGalleryImagesInput{
		IDs:     req.IDs,
		Approve: req.Action == "approve",
		Reason:  req.Reason,
		AdminID: adminID,
	})
	if err != nil {
		h.respondGalleryError(c, err, "batch review gallery images")
		return
	}
	response.Ok(c, reviewed)
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
	case errors.Is(err, service.ErrGalleryURLDuplicate):
		response.Error(c, appErrors.ErrConflict("该图片链接已在画廊中"))
	case errors.Is(err, service.ErrGalleryLimitExceeded):
		response.Error(c, appErrors.ErrConflict("游戏画面数量已达上限"))
	case errors.Is(err, service.ErrGalleryInvalidSource):
		response.Error(c, appErrors.ErrValidation("asset_id 与 external_url 必须二选一"))
	case errors.Is(err, service.ErrGalleryInvalidURL):
		response.Error(c, appErrors.ErrValidation("图片链接无效：仅支持 http/https，长度不超过 2048"))
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
