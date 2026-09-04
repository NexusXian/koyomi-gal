package model

import (
	"encoding/json"
	"time"
)

const (
	MatchCandidateStatusPending int16 = iota
	MatchCandidateStatusApproved
	MatchCandidateStatusRejected
)

type ExternalMatchCandidate struct {
	ID         uint64          `gorm:"primaryKey" json:"id"`
	GalgameID  uint            `gorm:"not null;uniqueIndex:idx_external_match_identity" json:"galgame_id"`
	Provider   string          `gorm:"size:32;not null;uniqueIndex:idx_external_match_identity" json:"provider"`
	ExternalID string          `gorm:"size:128;not null;uniqueIndex:idx_external_match_identity" json:"external_id"`
	Confidence float64         `gorm:"type:numeric(5,4);not null" json:"confidence"`
	Reasons    json.RawMessage `gorm:"type:jsonb;not null" json:"reasons"`
	Preview    json.RawMessage `gorm:"type:jsonb" json:"preview,omitempty"`
	Status     int16           `gorm:"not null" json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	ReviewedAt *time.Time      `json:"reviewed_at"`
	ReviewedBy *uint           `json:"reviewed_by"`

	// Review context joined from galgames, never persisted on this table.
	GalgameTitle         string `gorm:"-" json:"galgame_title,omitempty"`
	GalgameOriginalTitle string `gorm:"-" json:"galgame_original_title,omitempty"`
}

func (ExternalMatchCandidate) TableName() string {
	return "external_match_candidates"
}
