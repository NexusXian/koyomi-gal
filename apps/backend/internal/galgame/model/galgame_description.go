package model

import "time"

// Description languages stored per galgame, using BCP-47 style tags.
const (
	LanguageZhCN = "zh-CN"
	LanguageEnUS = "en-US"
	LanguageJaJP = "ja-JP"
)

// Additional description sources for the per-language table. Priority for
// automatic enrichment: manual > official > nextmoe > bangumi > steam >
// vndb > unknown.
const (
	DescriptionSourceNextMoe  = "nextmoe"
	DescriptionSourceOfficial = "official"
	DescriptionSourceSteam    = "steam"
)

// GalgameDescription stores one language's description for a galgame,
// together with structured provenance so the UI can attribute the source
// without parsing it out of the content.
type GalgameDescription struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	GalgameID  uint      `gorm:"not null;uniqueIndex:idx_galgame_descriptions_galgame_id_language" json:"galgame_id"`
	Language   string    `gorm:"size:16;not null;uniqueIndex:idx_galgame_descriptions_galgame_id_language" json:"language"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	SourceType string    `gorm:"size:32;not null;default:'unknown'" json:"source_type"`
	SourceName string    `gorm:"size:128;not null" json:"source_name"`
	SourceURL  string    `gorm:"type:text;not null" json:"source_url"`
	IsOfficial bool      `gorm:"not null;default:false" json:"is_official"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (GalgameDescription) TableName() string { return "galgame_descriptions" }

// SupportedDescriptionLanguages lists the accepted language tags.
var SupportedDescriptionLanguages = []string{LanguageZhCN, LanguageEnUS, LanguageJaJP}

// ValidDescriptionLanguage reports whether the tag is supported.
func ValidDescriptionLanguage(language string) bool {
	switch language {
	case LanguageZhCN, LanguageEnUS, LanguageJaJP:
		return true
	default:
		return false
	}
}

// ValidDescriptionSourceType reports whether the source type is accepted.
func ValidDescriptionSourceType(source string) bool {
	switch source {
	case DescriptionSourceUnknown,
		DescriptionSourceVNDB,
		DescriptionSourceBangumi,
		DescriptionSourceNextMoe,
		DescriptionSourceOfficial,
		DescriptionSourceSteam,
		DescriptionSourceManual:
		return true
	default:
		return false
	}
}

// DefaultDescriptionSourceName returns the display name used when a row has
// no explicit source_name.
func DefaultDescriptionSourceName(sourceType string) string {
	switch sourceType {
	case DescriptionSourceVNDB:
		return "VNDB"
	case DescriptionSourceBangumi:
		return "Bangumi"
	case DescriptionSourceNextMoe:
		return "NextMoe 资料库"
	case DescriptionSourceOfficial:
		return "游戏官网"
	case DescriptionSourceSteam:
		return "Steam"
	case DescriptionSourceManual:
		return "手动录入"
	default:
		return "未知来源"
	}
}
