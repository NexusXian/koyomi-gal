package dto

import (
	"time"

	"backend/internal/image/model"
)

type PresignImageRequest struct {
	Filename    string `json:"filename" binding:"required,max=255" example:"cover.png"`
	ContentType string `json:"content_type" binding:"required" example:"image/png"`
	Size        int64  `json:"size" binding:"required,gt=0" example:"204800"`
	Category    string `json:"category" binding:"required,oneof=avatars posts comments galgames backgrounds banners profile-banners admin" example:"posts"`
}

type PresignImageData struct {
	ID        uint   `json:"id" example:"123"`
	ObjectKey string `json:"object_key" example:"posts/10001/2026/09/01991e1d-a1b2-7c8d-9e0f-a1b2c3d4e5f6.png"`
	UploadURL string `json:"upload_url" example:"https://example.r2.cloudflarestorage.com/koyomi-gal-assets/posts/...?X-Amz-Signature=..."`
	ExpiresIn int    `json:"expires_in" example:"300"`
}

type PresignImageResponse struct {
	Code int              `json:"code" example:"0"`
	Data PresignImageData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

type CompleteUploadRequest struct {
	Width  *int `json:"width" binding:"omitempty,gte=1,lte=100000" example:"1920"`
	Height *int `json:"height" binding:"omitempty,gte=1,lte=100000" example:"1080"`
}

type ImageData struct {
	ID           uint       `json:"id" example:"123"`
	URL          string     `json:"url" example:"https://img.example.com/posts/10001/2026/09/uuid.png"`
	ObjectKey    string     `json:"object_key" example:"posts/10001/2026/09/uuid.png"`
	OriginalName string     `json:"original_name" example:"cover.png"`
	MimeType     string     `json:"mime_type" example:"image/png"`
	Extension    string     `json:"extension" example:"png"`
	Size         int64      `json:"size" example:"204800"`
	Width        *int       `json:"width"`
	Height       *int       `json:"height"`
	Category     string     `json:"category" example:"posts"`
	UserID       *uint      `json:"user_id" example:"10001"`
	Status       int16      `json:"status" example:"1"`
	CreatedAt    time.Time  `json:"created_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

type ImageDataResponse struct {
	Code int       `json:"code" example:"0"`
	Data ImageData `json:"data"`
	Msg  string    `json:"msg" example:"success"`
}

type AdminImageQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Category string `form:"category" binding:"omitempty,oneof=avatars posts comments galgames backgrounds banners profile-banners admin" example:"posts"`
	UserID   *uint  `form:"user_id" binding:"omitempty,min=1"`
	Status   *int16 `form:"status" binding:"omitempty,oneof=0 1 2 3" example:"1"`
}

type AdminImageListData struct {
	Items []ImageData `json:"items"`
	Total int64       `json:"total" example:"100"`
	Page  int         `json:"page" example:"1"`
	Limit int         `json:"limit" example:"20"`
}

type AdminImageListResponse struct {
	Code int                `json:"code" example:"0"`
	Data AdminImageListData `json:"data"`
	Msg  string             `json:"msg" example:"success"`
}

func NewImageData(asset *model.ImageAsset, publicURL string) ImageData {
	return ImageData{
		ID: asset.ID, URL: publicURL, ObjectKey: asset.ObjectKey,
		OriginalName: asset.OriginalName, MimeType: asset.MimeType,
		Extension: asset.Extension, Size: asset.Size,
		Width: asset.Width, Height: asset.Height,
		Category: asset.Category, UserID: asset.UserID, Status: asset.Status,
		CreatedAt: asset.CreatedAt, DeletedAt: asset.DeletedAt,
	}
}
