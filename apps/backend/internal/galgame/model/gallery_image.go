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

// Source of a gallery image: an R2-backed image asset or a third-party URL.
const (
	GallerySourceAsset int16 = iota
	GallerySourceExternal
)

// Independent review state of a gallery image; decoupled from Galgame.Status
// so publishing a game never has to rewind for its images or vice versa.
const (
	GalleryImageStatusPending int16 = iota
	GalleryImageStatusPublished
	GalleryImageStatusRejected
)

// GalleryImage associates an image with a galgame's gallery. The image is
// either an uploaded asset (binary content stays in object storage) or an
// external URL stored verbatim; this table only stores the relation plus its
// own review state.
type GalleryImage struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	GalgameID    uint       `gorm:"not null;uniqueIndex:uk_galgame_gallery_images_galgame_asset,priority:1" json:"galgame_id"`
	SourceType   int16      `gorm:"not null;default:0" json:"source_type"`
	AssetID      *uint      `gorm:"uniqueIndex:uk_galgame_gallery_images_galgame_asset,priority:2" json:"asset_id"`
	ExternalURL  string     `gorm:"type:text;not null;default:''" json:"external_url"`
	Title        string     `gorm:"size:255" json:"title"`
	Description  string     `gorm:"type:text" json:"description"`
	SortOrder    int        `gorm:"not null;default:0" json:"sort_order"`
	ImageType    int16      `gorm:"not null;default:0" json:"image_type"`
	IsSpoiler    bool       `gorm:"not null;default:false" json:"is_spoiler"`
	Status       int16      `gorm:"not null;default:0" json:"status"`
	CreatedBy    *uint      `json:"created_by"`
	ReviewedBy   *uint      `json:"reviewed_by"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	RejectReason string     `gorm:"type:text;not null;default:''" json:"reject_reason"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	Asset *imageModel.ImageAsset `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (GalleryImage) TableName() string {
	return "galgame_gallery_images"
}
