package dto

import (
	"time"

	"backend/internal/galgame/model"
	imageModel "backend/internal/image/model"
	imageService "backend/internal/image/service"
)

type CreateGalleryImageRequest struct {
	AssetID     *uint  `json:"asset_id" example:"10086"`
	ExternalURL string `json:"external_url" binding:"omitempty,max=2048" example:"https://source-a.com/images/123.jpg"`
	Title       string `json:"title" binding:"max=255" example:"标题画面"`
	Description string `json:"description" example:"游戏标题界面截图"`
	ImageType   int16  `json:"image_type" binding:"oneof=0 1 2 3 4" example:"0"`
	IsSpoiler   bool   `json:"is_spoiler" example:"false"`
}

type BatchGalleryImageItem struct {
	ExternalURL string `json:"external_url" binding:"required,max=2048" example:"https://source-a.com/images/123.jpg"`
	Title       string `json:"title" binding:"max=255" example:"标题画面"`
	ImageType   int16  `json:"image_type" binding:"oneof=0 1 2 3 4" example:"0"`
	IsSpoiler   bool   `json:"is_spoiler" example:"false"`
}

type BatchCreateGalleryRequest struct {
	Items []BatchGalleryImageItem `json:"items" binding:"required,min=1,max=100,dive"`
}

type BatchGalleryResultData struct {
	Created int `json:"created" example:"18"`
	Skipped int `json:"skipped" example:"2"`
	Failed  int `json:"failed" example:"1"`
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

type ReviewGalleryImageRequest struct {
	Reason string `json:"reason" binding:"max=500" example:"图片与游戏无关"`
}

type BatchReviewGalleryRequest struct {
	IDs    []uint `json:"ids" binding:"required,min=1,max=100,dive,gt=0" example:"1,2,3"`
	Action string `json:"action" binding:"required,oneof=approve reject" example:"approve"`
	Reason string `json:"reason" binding:"omitempty,max=500" example:"重复图片"`
}

type GalleryReviewListQuery struct {
	Status     *int16 `form:"status" binding:"omitempty,oneof=0 1 2" example:"0"`
	GalgameID  uint   `form:"galgame_id" binding:"omitempty,gt=0" example:"101"`
	SourceType *int16 `form:"source_type" binding:"omitempty,oneof=0 1" example:"1"`
	Page       int    `form:"page" binding:"omitempty,min=1,max=1000000" example:"1"`
	Limit      int    `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
}

type GalleryImageData struct {
	ID           uint   `json:"id" example:"1001"`
	SourceType   int16  `json:"source_type" example:"0"`
	AssetID      *uint  `json:"asset_id,omitempty" example:"3001"`
	URL          string `json:"url" example:"https://img.example.com/galgames/10001/2026/09/uuid.webp"`
	ExternalURL  string `json:"external_url,omitempty" example:"https://source-a.com/images/123.jpg"`
	Width        *int   `json:"width,omitempty" example:"1920"`
	Height       *int   `json:"height,omitempty" example:"1080"`
	Title        string `json:"title" example:"标题画面"`
	Description  string `json:"description" example:"游戏标题界面截图"`
	ImageType    int16  `json:"image_type" example:"0"`
	IsSpoiler    bool   `json:"is_spoiler" example:"false"`
	Status       int16  `json:"status" example:"1"`
	RejectReason string `json:"reject_reason,omitempty" example:"图片与游戏无关"`
	SortOrder    int    `json:"sort_order" example:"0"`
}

type GalleryListData struct {
	Items []GalleryImageData `json:"items"`
	Total int64              `json:"total" example:"12"`
}

type GalleryReviewItemData struct {
	ID                 uint       `json:"id" example:"1001"`
	GalgameID          uint       `json:"galgame_id" example:"101"`
	GalgameTitle       string     `json:"galgame_title" example:"Summer Pockets"`
	GalgameSlug        string     `json:"galgame_slug" example:"summer-pockets"`
	SourceType         int16      `json:"source_type" example:"1"`
	URL                string     `json:"url" example:"https://source-a.com/images/123.jpg"`
	ExternalURL        string     `json:"external_url,omitempty" example:"https://source-a.com/images/123.jpg"`
	Title              string     `json:"title" example:"标题画面"`
	ImageType          int16      `json:"image_type" example:"0"`
	IsSpoiler          bool       `json:"is_spoiler" example:"false"`
	Status             int16      `json:"status" example:"0"`
	RejectReason       string     `json:"reject_reason,omitempty" example:"图片与游戏无关"`
	CreatedByUsername  string     `json:"created_by_username" example:"NexusXian"`
	ReviewedByUsername string     `json:"reviewed_by_username,omitempty" example:"admin"`
	CreatedAt          time.Time  `json:"created_at"`
	ReviewedAt         *time.Time `json:"reviewed_at"`
}

type GalleryReviewListData struct {
	Items []GalleryReviewItemData `json:"items"`
	Total int64                   `json:"total" example:"12"`
	Page  int                     `json:"page" example:"1"`
	Limit int                     `json:"limit" example:"20"`
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

type GalleryBatchResponse struct {
	Code int                    `json:"code" example:"0"`
	Data BatchGalleryResultData `json:"data"`
	Msg  string                 `json:"msg" example:"success"`
}

type GalleryReviewListResponse struct {
	Code int                   `json:"code" example:"0"`
	Data GalleryReviewListData `json:"data"`
	Msg  string                `json:"msg" example:"success"`
}

type GalleryReviewBatchResponse struct {
	Code int    `json:"code" example:"0"`
	Data int    `json:"data" example:"18"`
	Msg  string `json:"msg" example:"success"`
}

// GalleryImageURL unifies both sources into the URL a browser should load.
// External images return the stored URL verbatim; asset images are built from
// the asset object key.
func GalleryImageURL(image *model.GalleryImage, assets *imageService.ImageAssetService) string {
	if image.SourceType == model.GallerySourceExternal {
		return image.ExternalURL
	}
	if image.Asset != nil {
		return assets.BuildPublicURL(image.Asset.ObjectKey)
	}
	return ""
}

// NewGalleryImageList builds public response items: only published entries
// whose asset is still active are returned.
func NewGalleryImageList(images []model.GalleryImage, assets *imageService.ImageAssetService) GalleryListData {
	items := make([]GalleryImageData, 0, len(images))
	for i := range images {
		image := &images[i]
		if image.Status != model.GalleryImageStatusPublished {
			continue
		}
		if image.SourceType == model.GallerySourceAsset &&
			image.Asset.Status != imageModel.ImageStatusActive {
			continue
		}
		items = append(items, NewGalleryImageData(image, assets))
	}
	return GalleryListData{Items: items, Total: int64(len(items))}
}

// NewAdminGalleryImageList builds admin response items with every review
// status visible; asset-backed entries with a missing/inactive asset keep an
// empty URL instead of disappearing so they can still be managed.
func NewAdminGalleryImageList(images []model.GalleryImage, assets *imageService.ImageAssetService) GalleryListData {
	items := make([]GalleryImageData, 0, len(images))
	for i := range images {
		items = append(items, NewGalleryImageData(&images[i], assets))
	}
	return GalleryListData{Items: items, Total: int64(len(items))}
}

func NewGalleryImageData(image *model.GalleryImage, assets *imageService.ImageAssetService) GalleryImageData {
	data := GalleryImageData{
		ID:           image.ID,
		SourceType:   image.SourceType,
		AssetID:      image.AssetID,
		URL:          GalleryImageURL(image, assets),
		ExternalURL:  image.ExternalURL,
		Title:        image.Title,
		Description:  image.Description,
		ImageType:    image.ImageType,
		IsSpoiler:    image.IsSpoiler,
		Status:       image.Status,
		RejectReason: image.RejectReason,
		SortOrder:    image.SortOrder,
	}
	if image.Asset != nil {
		data.Width = image.Asset.Width
		data.Height = image.Asset.Height
	}
	if image.SourceType == model.GallerySourceExternal {
		data.ExternalURL = image.ExternalURL
	}
	return data
}
