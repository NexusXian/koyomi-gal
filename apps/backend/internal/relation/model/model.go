package model

import "time"

// Work types usable as relation endpoints. Kept in one place so the relation,
// resource, and contribution modules share identical values.
const (
	WorkTypeGalgame = "galgame"
	WorkTypeNovel   = "novel"
)

type RelationType string

const (
	RelationTypeAdaptation RelationType = "adaptation"
	RelationTypeOriginal   RelationType = "original"
	RelationTypeSpinOff    RelationType = "spin_off"
	RelationTypeSequel     RelationType = "sequel"
	RelationTypePrequel    RelationType = "prequel"
	RelationTypeSameSeries RelationType = "same_series"
	RelationTypeRelated    RelationType = "related"
)

func ValidRelationType(value RelationType) bool {
	switch value {
	case RelationTypeAdaptation,
		RelationTypeOriginal,
		RelationTypeSpinOff,
		RelationTypeSequel,
		RelationTypePrequel,
		RelationTypeSameSeries,
		RelationTypeRelated:
		return true
	default:
		return false
	}
}

func ValidWorkType(value string) bool {
	return value == WorkTypeGalgame || value == WorkTypeNovel
}

type WorkRelation struct {
	ID           uint         `gorm:"primaryKey" json:"id"`
	SourceType   string       `gorm:"size:32;not null" json:"source_type"`
	SourceID     uint         `gorm:"not null" json:"source_id"`
	TargetType   string       `gorm:"size:32;not null" json:"target_type"`
	TargetID     uint         `gorm:"not null" json:"target_id"`
	RelationType RelationType `gorm:"size:32;not null" json:"relation_type"`
	CreatedBy    *uint        `json:"created_by"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

func (WorkRelation) TableName() string { return "work_relations" }

// RelatedWork is a SQL scan projection joining the opposite side of a
// relation with the published work's summary fields.
type RelatedWork struct {
	RelationID     uint   `json:"relation_id"`
	RelationType   string `json:"relation_type"`
	WorkID         uint   `json:"work_id"`
	Title          string `json:"title"`
	OriginalTitle  string `json:"original_title"`
	Slug           string `json:"slug"`
	CoverURL       string `json:"cover_url"`
	CoverSensitive bool   `json:"cover_sensitive"`
	AgeRating      int16  `json:"age_rating"`
}
