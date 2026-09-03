package model

import (
	"time"

	imageModel "backend/internal/image/model"
)

const (
	GalleryImageTypeScreenshot int16 = iota
	GalleryImageTypeCG
	GalleryImageTypeCharacter
	GalleryImageTypeUI
	GalleryImageTypePromotional
)

// GalleryImage associates an uploaded image asset with a galgame's gallery.
// Binary content stays in object storage; this table only stores the relation.
type GalleryImage struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	GalgameID   uint   `gorm:"not null;uniqueIndex:uk_galgame_gallery_images_galgame_asset,priority:1" json:"galgame_id"`
	AssetID     uint   `gorm:"not null;uniqueIndex:uk_galgame_gallery_images_galgame_asset,priority:2" json:"asset_id"`
	Title       string `gorm:"size:255" json:"title"`
	Description string `gorm:"type:text" json:"description"`
	SortOrder   int    `gorm:"not null;default:0" json:"sort_order"`
	ImageType   int16  `gorm:"not null;default:0" json:"image_type"`
	IsSpoiler   bool   `gorm:"not null;default:false" json:"is_spoiler"`
	CreatedBy   *uint  `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Asset imageModel.ImageAsset `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (GalleryImage) TableName() string {
	return "galgame_gallery_images"
}
