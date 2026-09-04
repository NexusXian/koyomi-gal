package model

import "time"

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

type Galgame struct {
	ID               uint                 `gorm:"primaryKey" json:"id"`
	Title            string               `gorm:"size:255;not null" json:"title"`
	OriginalTitle    string               `gorm:"size:255;not null" json:"original_title"`
	RomajiTitle      string               `gorm:"size:255;not null" json:"romaji_title"`
	Slug             string               `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Description      string               `gorm:"not null" json:"description"`
	CoverURL         string               `gorm:"not null" json:"cover_url"`
	BannerURL        string               `gorm:"not null" json:"banner_url"`
	DeveloperID      *uint                `json:"developer_id"`
	ReleaseDate      *time.Time           `gorm:"type:date" json:"release_date"`
	AgeRating        int16                `gorm:"not null" json:"age_rating"`
	Status           int16                `gorm:"not null" json:"status"`
	RatingAverage    float64              `gorm:"type:numeric(4,2);not null" json:"rating_average"`
	RatingCount      int64                `gorm:"not null" json:"rating_count"`
	FavoriteCount    int64                `gorm:"not null" json:"favorite_count"`
	ResourceCount    int64                `gorm:"not null" json:"resource_count"`
	PostCount        int64                `gorm:"not null" json:"post_count"`
	CreatedBy        *uint                `json:"created_by"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
	Developer        *Developer           `gorm:"foreignKey:DeveloperID" json:"developer,omitempty"`
	Aliases          []Alias              `gorm:"foreignKey:GalgameID" json:"aliases,omitempty"`
	Tags             []Tag                `gorm:"many2many:galgame_tags" json:"tags,omitempty"`
	Contributors     []GalgameContributor `gorm:"-" json:"-"`
	ContributorCount int64                `gorm:"-" json:"-"`
}
