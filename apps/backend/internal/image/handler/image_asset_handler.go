package handler

import (
	"errors"
	"strconv"

	"backend/internal/image/dto"
	"backend/internal/image/service"
	"backend/internal/infrastructures/storage"
	"backend/internal/middleware"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ImageHandler struct {
	imageService *service.ImageAssetService
}

func NewImageHandler(imageService *service.ImageAssetService) *ImageHandler {
	return &ImageHandler{imageService: imageService}
}

// PresignUpload godoc
// @Summary      申请图片上传凭证
// @Description  校验类型、大小和分类权限后，创建 pending 图片记录并返回 R2 Presigned PUT URL；该 URL 仅用于上传，与访问图片的 CDN URL 不同
// @ID           presignImageUpload
// @Tags         images
// @Accept       json
// @Produce      json
// @Param        request body dto.PresignImageRequest true "上传申请"
// @Success      200 {object} dto.PresignImageResponse "上传凭证"
// @Failure      400 {object} response.ErrorResponse "图片类型、大小或分类不合法"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有该分类的上传权限"
// @Failure      429 {object} response.ErrorResponse "上传请求过于频繁或超出每日配额"
// @Failure      500 {object} response.ErrorResponse "生成上传凭证失败"
// @Security     BearerAuth
// @Router       /api/v1/images/presign [post]
func (h *ImageHandler) PresignUpload(c *gin.Context) {
	userID, ok := currentImageUserID(c)
	if !ok {
		return
	}
	var req dto.PresignImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("上传申请参数不正确"))
		return
	}
	data, err := h.imageService.CreatePresignedUpload(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondError(c, err, "presign image upload")
		return
	}
	response.Ok(c, data)
}

// CompleteUpload godoc
// @Summary      确认图片上传完成
// @Description  通过 R2 HeadObject 验证对象存在且与申请一致后将图片置为 active；重复确认已 active 的图片是幂等的
// @ID           completeImageUpload
// @Tags         images
// @Accept       json
// @Produce      json
// @Param        id path int true "图片 ID"
// @Param        request body dto.CompleteUploadRequest false "图片尺寸"
// @Success      200 {object} dto.ImageDataResponse "图片详情，包含 CDN 访问 URL"
// @Failure      400 {object} response.ErrorResponse "图片 ID 或参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "只能操作自己上传的图片"
// @Failure      404 {object} response.ErrorResponse "图片不存在"
// @Failure      409 {object} response.ErrorResponse "对象尚未上传或与申请不一致"
// @Security     BearerAuth
// @Router       /api/v1/images/{id}/complete [post]
func (h *ImageHandler) CompleteUpload(c *gin.Context) {
	userID, ok := currentImageUserID(c)
	if !ok {
		return
	}
	id, ok := parseImageID(c)
	if !ok {
		return
	}
	var req dto.CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("确认上传参数不正确"))
		return
	}
	asset, err := h.imageService.CompleteUpload(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.respondError(c, err, "complete image upload")
		return
	}
	response.Ok(c, dto.NewImageData(asset, h.imageService.BuildPublicURL(asset.ObjectKey)))
}

// GetImage godoc
// @Summary      查看图片
// @Description  返回 active 状态图片的元数据和 CDN 访问 URL
// @ID           getImage
// @Tags         images
// @Produce      json
// @Param        id path int true "图片 ID"
// @Success      200 {object} dto.ImageDataResponse "图片详情"
// @Failure      400 {object} response.ErrorResponse "图片 ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "图片不存在"
// @Router       /api/v1/images/{id} [get]
func (h *ImageHandler) GetImage(c *gin.Context) {
	id, ok := parseImageID(c)
	if !ok {
		return
	}
	asset, err := h.imageService.GetImage(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "get image")
		return
	}
	response.Ok(c, dto.NewImageData(asset, h.imageService.BuildPublicURL(asset.ObjectKey)))
}

// DeleteImage godoc
// @Summary      删除图片
// @Description  删除 R2 对象并将记录标记为 deleted；用户只能删除自己上传的图片，拥有 image:delete 权限可以删除任意图片
// @ID           deleteImage
// @Tags         images
// @Produce      json
// @Param        id path int true "图片 ID"
// @Success      200 {object} response.MessageResponse "图片已删除"
// @Failure      400 {object} response.ErrorResponse "图片 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "只能删除自己上传的图片"
// @Failure      404 {object} response.ErrorResponse "图片不存在"
// @Security     BearerAuth
// @Router       /api/v1/images/{id} [delete]
func (h *ImageHandler) DeleteImage(c *gin.Context) {
	userID, ok := currentImageUserID(c)
	if !ok {
		return
	}
	id, ok := parseImageID(c)
	if !ok {
		return
	}
	if err := h.imageService.DeleteImage(c.Request.Context(), userID, id); err != nil {
		h.respondError(c, err, "delete image")
		return
	}
	response.OkWithMsg(c, "图片已删除")
}

// ListAdminImages godoc
// @Summary      管理端查询图片
// @Description  分页返回全部图片资源；需要 image:read 权限
// @ID           listAdminImages
// @Tags         admin
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        limit query int false "每页数量，最大 100" default(20)
// @Param        category query string false "分类" Enums(avatars,posts,comments,galgames,backgrounds,banners,admin)
// @Param        user_id query int false "上传用户 ID"
// @Param        status query int false "状态" Enums(0,1,2,3)
// @Success      200 {object} dto.AdminImageListResponse "图片列表"
// @Failure      400 {object} response.ErrorResponse "查询参数格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      500 {object} response.ErrorResponse "查询图片失败"
// @Security     BearerAuth
// @Router       /api/v1/admin/images [get]
func (h *ImageHandler) ListAdminImages(c *gin.Context) {
	var query dto.AdminImageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("查询参数格式不正确"))
		return
	}
	assets, total, page, limit, err := h.imageService.ListAdmin(c.Request.Context(), query)
	if err != nil {
		h.respondError(c, err, "list admin images")
		return
	}
	items := make([]dto.ImageData, 0, len(assets))
	for i := range assets {
		items = append(items, dto.NewImageData(&assets[i], h.imageService.BuildPublicURL(assets[i].ObjectKey)))
	}
	response.Ok(c, dto.AdminImageListData{Items: items, Total: total, Page: page, Limit: limit})
}

// GetAdminImage godoc
// @Summary      管理端查看图片
// @Description  按 ID 返回任意状态的图片元数据；需要 image:read 权限
// @ID           getAdminImage
// @Tags         admin
// @Produce      json
// @Param        id path int true "图片 ID"
// @Success      200 {object} dto.ImageDataResponse "图片详情"
// @Failure      400 {object} response.ErrorResponse "图片 ID 格式不正确"
// @Failure      404 {object} response.ErrorResponse "图片不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/images/{id} [get]
func (h *ImageHandler) GetAdminImage(c *gin.Context) {
	id, ok := parseImageID(c)
	if !ok {
		return
	}
	asset, err := h.imageService.GetAdmin(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err, "get admin image")
		return
	}
	response.Ok(c, dto.NewImageData(asset, h.imageService.BuildPublicURL(asset.ObjectKey)))
}

// DeleteAdminImage godoc
// @Summary      管理端删除图片
// @Description  删除任意用户的图片；需要 image:delete 权限
// @ID           deleteAdminImage
// @Tags         admin
// @Produce      json
// @Param        id path int true "图片 ID"
// @Success      200 {object} response.MessageResponse "图片已删除"
// @Failure      400 {object} response.ErrorResponse "图片 ID 格式不正确"
// @Failure      401 {object} response.ErrorResponse "用户登录失效"
// @Failure      403 {object} response.ErrorResponse "没有执行该操作的权限"
// @Failure      404 {object} response.ErrorResponse "图片不存在"
// @Security     BearerAuth
// @Router       /api/v1/admin/images/{id} [delete]
func (h *ImageHandler) DeleteAdminImage(c *gin.Context) {
	userID, ok := currentImageUserID(c)
	if !ok {
		return
	}
	id, ok := parseImageID(c)
	if !ok {
		return
	}
	if err := h.imageService.DeleteImage(c.Request.Context(), userID, id); err != nil {
		h.respondError(c, err, "delete admin image")
		return
	}
	response.OkWithMsg(c, "图片已删除")
}

func (h *ImageHandler) respondError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, service.ErrInvalidImageType):
		response.Error(c, appErrors.ErrValidation("不支持的图片类型"))
	case errors.Is(err, service.ErrImageTooLarge):
		response.Error(c, appErrors.ErrValidation("图片大小超出限制"))
	case errors.Is(err, service.ErrInvalidImageCategory):
		response.Error(c, appErrors.ErrForbidden("没有该分类的上传权限"))
	case errors.Is(err, service.ErrImageNotFound):
		response.Error(c, appErrors.ErrNotFound("图片不存在"))
	case errors.Is(err, service.ErrImageForbidden):
		response.Error(c, appErrors.ErrForbidden("只能操作自己上传的图片"))
	case errors.Is(err, service.ErrImageInvalidState):
		response.Error(c, appErrors.ErrConflict("图片当前状态不允许该操作"))
	case errors.Is(err, service.ErrImageUploadIncomplete), errors.Is(err, storage.ErrObjectNotFound):
		response.Error(c, appErrors.ErrConflict("图片尚未上传完成或与申请不一致"))
	case errors.Is(err, service.ErrPresignLimitExceeded), errors.Is(err, service.ErrImageQuotaExceeded):
		response.Error(c, appErrors.ErrTooManyRequests("上传请求过于频繁或超出每日配额"))
	default:
		logger.Error(operation, zap.Error(err))
		response.Error(c, appErrors.ErrInternal("图片操作失败"))
	}
}

func parseImageID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("图片 ID 格式不正确"))
		return 0, false
	}
	return uint(id), true
}

func currentImageUserID(c *gin.Context) (uint, bool) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, appErrors.ErrAuthExpired())
		return 0, false
	}
	return userID, true
}
