package model

import (
	"encoding/json"
	"time"
)

type GalgameExternalSource struct {
	ID                  uint            `gorm:"primaryKey" json:"id"`
	GalgameID           uint            `gorm:"not null;index" json:"galgame_id"`
	Source              string          `gorm:"size:32;not null;uniqueIndex:idx_external_source_identity" json:"source"`
	ExternalID          string          `gorm:"size:128;not null;uniqueIndex:idx_external_source_identity" json:"external_id"`
	URL                 string          `gorm:"not null" json:"url"`
	ExternalRating      *float64        `gorm:"type:numeric(4,2)" json:"external_rating"`
	ExternalRatingCount *int            `json:"external_rating_count"`
	RawMetadata         json.RawMessage `gorm:"type:jsonb" json:"raw_metadata,omitempty"`
	LastSyncedAt        *time.Time      `json:"last_synced_at"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

func (GalgameExternalSource) TableName() string {
	return "galgame_external_sources"
}
