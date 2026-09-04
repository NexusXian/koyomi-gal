package provider

import (
	"context"
	"encoding/json"
	"time"
)

type Provider interface {
	Search(ctx context.Context, query string, limit int) ([]ExternalGame, error)
	Get(ctx context.Context, externalID string) (*ExternalGame, error)
}

// BatchFilters describes catalog-wide batch query filters for providers that
// support listing without a search keyword.
type BatchFilters struct {
	MinRating        float64 // 0-10 scale, 0 means unset
	MinVoteCount     int
	FromYear         int // 0 means unset
	ToYear           int // 0 means unset
	OriginalLanguage string
}

type BatchListResult struct {
	Games []ExternalGame
	More  bool
	Count int64 // -1 when the provider did not return a total
}

// BatchLister is implemented by providers that support filtered batch listing.
type BatchLister interface {
	ListBatch(
		ctx context.Context,
		filters BatchFilters,
		page, results int,
		withCount bool,
	) (*BatchListResult, error)
}

type ExternalGame struct {
	Source     string `json:"source"`
	ExternalID string `json:"external_id"`

	Title         string   `json:"title"`
	OriginalTitle string   `json:"original_title"`
	RomajiTitle   string   `json:"romaji_title"`
	Aliases       []string `json:"aliases"`

	Description string `json:"description"`
	CoverURL    string `json:"cover_url"`

	ReleaseDate *time.Time `json:"release_date"`

	OriginalLanguage string `json:"original_language"`
	LengthMinutes    *int   `json:"length_minutes"`

	Rating      *float64 `json:"rating"`
	RatingCount *int     `json:"rating_count"`

	Developer *ExternalDeveloper `json:"developer"`
	Tags      []ExternalTag      `json:"tags"`

	Raw json.RawMessage `json:"-"`
}

type ExternalDeveloper struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
}

type ExternalTag struct {
	ExternalID string  `json:"external_id"`
	Name       string  `json:"name"`
	Spoiler    int     `json:"spoiler"`
	Rating     float64 `json:"rating"`
}
