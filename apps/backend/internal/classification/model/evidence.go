package model

import "time"

// Evidence source types, ordered by trustworthiness in agent prompts.
const (
	SourceTypeOfficial  = "official"
	SourceTypeSteam     = "steam"
	SourceTypeVNDB      = "vndb"
	SourceTypeBangumi   = "bangumi"
	SourceTypeCERO      = "cero"
	SourceTypeESRB      = "esrb"
	SourceTypePEGI      = "pegi"
	SourceTypeWikipedia = "wikipedia"
	SourceTypeOther     = "other"
)

func ValidSourceType(value string) bool {
	switch value {
	case SourceTypeOfficial, SourceTypeSteam, SourceTypeVNDB, SourceTypeBangumi,
		SourceTypeCERO, SourceTypeESRB, SourceTypePEGI, SourceTypeWikipedia, SourceTypeOther:
		return true
	default:
		return false
	}
}

type GameClassificationEvidence struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	ClassificationID uint      `gorm:"not null;index" json:"classification_id"`
	SourceType       string    `gorm:"size:32;not null;default:''" json:"source_type"`
	Title            string    `gorm:"not null;default:''" json:"title"`
	URL              string    `gorm:"not null;default:''" json:"url"`
	Evidence         string    `gorm:"not null;default:''" json:"evidence"`
	Weight           int       `gorm:"not null;default:0" json:"weight"`
	CreatedAt        time.Time `json:"created_at"`
}

func (GameClassificationEvidence) TableName() string {
	return "game_classification_evidences"
}
