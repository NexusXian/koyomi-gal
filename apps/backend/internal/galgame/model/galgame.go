package model

import (
	"time"

	contributionModel "backend/internal/contribution/model"
	relationModel "backend/internal/relation/model"
)

const (
	GalgameStatusPending int16 = iota
	GalgameStatusPublished
	GalgameStatusRejected
	GalgameStatusHidden
)

const (
	AgeRatingUnknown int16 = iota
	AgeRatingAll
	AgeRatingR15
	AgeRatingR18
)

// Extended levels are appended after the original four so existing stored
// values stay stable.
const (
	AgeRatingR12 int16 = iota + 4
	AgeRatingR17
)

const (
	GalgameSourceManual int16 = iota
	GalgameSourceVNDB
	GalgameSourceBangumi
	GalgameSourceMixed
)

// Description sources ordered by enrichment priority: manual > bangumi >
// vndb > unknown. Automatic enrichment may only replace a description
// coming from a lower-priority source.
const (
	DescriptionSourceUnknown = "unknown"
	DescriptionSourceVNDB    = "vndb"
	DescriptionSourceBangumi = "bangumi"
	DescriptionSourceManual  = "manual"
)

type Galgame struct {
	ID                uint                                `gorm:"primaryKey" json:"id"`
	Title             string                              `gorm:"size:255;not null" json:"title"`
	OriginalTitle     string                              `gorm:"size:255;not null" json:"original_title"`
	RomajiTitle       string                              `gorm:"size:255;not null" json:"romaji_title"`
	Slug              string                              `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Description       string                              `gorm:"not null" json:"description"`
	DescriptionSource string                              `gorm:"size:32;not null;default:''" json:"description_source"`
	CoverURL          string                              `gorm:"not null" json:"cover_url"`
	BannerURL         string                              `gorm:"not null" json:"banner_url"`
	DeveloperID       *uint                               `json:"developer_id"`
	ReleaseDate       *time.Time                          `gorm:"type:date" json:"release_date"`
	OriginalLanguage  string                              `gorm:"size:16;not null" json:"original_language"`
	LengthMinutes     *int                                `json:"length_minutes"`
	SourceType        int16                               `gorm:"not null" json:"source_type"`
	MetadataUpdatedAt *time.Time                          `json:"metadata_updated_at"`
	AgeRating         int16                               `gorm:"not null" json:"age_rating"`
	CoverSensitive    bool                                `gorm:"not null;default:false;index" json:"cover_sensitive"`
	Status            int16                               `gorm:"not null" json:"status"`
	RatingAverage     float64                             `gorm:"type:numeric(4,2);not null" json:"rating_average"`
	RatingCount       int64                               `gorm:"not null" json:"rating_count"`
	FavoriteCount     int64                               `gorm:"not null" json:"favorite_count"`
	ResourceCount     int64                               `gorm:"not null" json:"resource_count"`
	PostCount         int64                               `gorm:"not null" json:"post_count"`
	CreatedBy         *uint                               `json:"created_by"`
	CreatedAt         time.Time                           `json:"created_at"`
	UpdatedAt         time.Time                           `json:"updated_at"`
	Developer         *Developer                          `gorm:"foreignKey:DeveloperID" json:"developer,omitempty"`
	Aliases           []Alias                             `gorm:"foreignKey:GalgameID" json:"aliases,omitempty"`
	Tags              []Tag                               `gorm:"many2many:galgame_tags" json:"tags,omitempty"`
	Contributors      []contributionModel.WorkContributor `gorm:"-" json:"-"`
	ContributorCount  int64                               `gorm:"-" json:"-"`
	RelatedNovels     []relationModel.RelatedWork         `gorm:"-" json:"-"`

	// Transient read-only projection of the game's latest AI classification
	// row, populated only by the admin listing query.
	AIClassification string  `gorm:"->;-:migration" json:"-"`
	AIConfidence     float64 `gorm:"->;-:migration" json:"-"`
	AIStatus         string  `gorm:"->;-:migration" json:"-"`
	AIConflict       bool    `gorm:"->;-:migration" json:"-"`
}
