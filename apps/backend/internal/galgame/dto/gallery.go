package dto

import (
	"backend/internal/galgame/model"
	imageModel "backend/internal/image/model"
	imageService "backend/internal/image/service"
)

type CreateGalleryImageRequest struct {
	AssetID     uint   `json:"asset_id" binding:"required,gt=0" example:"10086"`
	Title       string `json:"title" binding:"max=255" example:"标题画面"`
	Description string `json:"description" example:"游戏标题界面截图"`
	ImageType   int16  `json:"image_type" binding:"oneof=0 1 2 3 4" example:"0"`
	IsSpoiler   bool   `json:"is_spoiler" example:"false"`
}

type UpdateGalleryImageRequest struct {
	Title       *string `json:"title" binding:"omitempty,max=255" example:"游戏实际画面"`
	Description *string `json:"description" example:"共通线结尾画面"`
	ImageType   *int16  `json:"image_type" binding:"omitempty,oneof=0 1 2 3 4" example:"1"`
	IsSpoiler   *bool   `json:"is_spoiler" example:"true"`
}

type ReorderGalleryRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1,dive,gt=0" example:"103,101,105,102"`
}

type GalleryImageData struct {
	ID          uint   `json:"id" example:"1001"`
	AssetID     uint   `json:"asset_id" example:"3001"`
	URL         string `json:"url" example:"https://img.example.com/galgames/10001/2026/09/uuid.webp"`
	Width       *int   `json:"width" example:"1920"`
	Height      *int   `json:"height" example:"1080"`
	Title       string `json:"title" example:"标题画面"`
	Description string `json:"description" example:"游戏标题界面截图"`
	ImageType   int16  `json:"image_type" example:"0"`
	IsSpoiler   bool   `json:"is_spoiler" example:"false"`
	SortOrder   int    `json:"sort_order" example:"0"`
}

type GalleryListData struct {
	Items []GalleryImageData `json:"items"`
	Total int64              `json:"total" example:"12"`
}

type GalleryListResponse struct {
	Code int             `json:"code" example:"0"`
	Data GalleryListData `json:"data"`
	Msg  string          `json:"msg" example:"success"`
}

type GalleryDataResponse struct {
	Code int              `json:"code" example:"0"`
	Data GalleryImageData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

// NewGalleryImageList builds response items, skipping entries whose asset is
// no longer active so a soft-deleted asset disappears from the gallery.
func NewGalleryImageList(images []model.GalleryImage, assets *imageService.ImageAssetService) GalleryListData {
	items := make([]GalleryImageData, 0, len(images))
	for i := range images {
		image := &images[i]
		if image.Asset.Status != imageModel.ImageStatusActive {
			continue
		}
		items = append(items, GalleryImageData{
			ID:          image.ID,
			AssetID:     image.AssetID,
			URL:         assets.BuildPublicURL(image.Asset.ObjectKey),
			Width:       image.Asset.Width,
			Height:      image.Asset.Height,
			Title:       image.Title,
			Description: image.Description,
			ImageType:   image.ImageType,
			IsSpoiler:   image.IsSpoiler,
			SortOrder:   image.SortOrder,
		})
	}
	return GalleryListData{Items: items, Total: int64(len(items))}
}

func NewGalleryImageData(image *model.GalleryImage, assets *imageService.ImageAssetService) GalleryImageData {
	return GalleryImageData{
		ID:          image.ID,
		AssetID:     image.AssetID,
		URL:         assets.BuildPublicURL(image.Asset.ObjectKey),
		Width:       image.Asset.Width,
		Height:      image.Asset.Height,
		Title:       image.Title,
		Description: image.Description,
		ImageType:   image.ImageType,
		IsSpoiler:   image.IsSpoiler,
		SortOrder:   image.SortOrder,
	}
}
