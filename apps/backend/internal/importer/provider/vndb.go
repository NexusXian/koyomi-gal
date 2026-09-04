package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	vndbSource       = "vndb"
	vndbEndpoint     = "https://api.vndb.org/kana/vn"
	vndbFields       = "title,alttitle,titles{lang,title,latin,official,main},aliases,description,image.url,released,languages,length_minutes,developers{id,name},rating,votecount,tags{id,name,spoiler,rating}"
	defaultVNDBLimit = 20
	maxVNDBLimit     = 100
)

var vndbIDPattern = regexp.MustCompile(`^v[1-9][0-9]*$`)

type VNDBProvider struct {
	client   *http.Client
	endpoint string
	limiter  *rate.Limiter
}

type vndbRequest struct {
	Filters any    `json:"filters"`
	Fields  string `json:"fields"`
	Sort    string `json:"sort,omitempty"`
	Reverse bool   `json:"reverse,omitempty"`
	Results int    `json:"results"`
	Page    int    `json:"page,omitempty"`
	Count   bool   `json:"count,omitempty"`
}

type vndbResponse struct {
	Results []json.RawMessage `json:"results"`
	More    bool              `json:"more"`
	Count   int               `json:"count,omitempty"`
}

type vndbGame struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	AltTitle    *string         `json:"alttitle"`
	Titles      []vndbTitle     `json:"titles"`
	Aliases     []string        `json:"aliases"`
	Description *string         `json:"description"`
	Image       *vndbImage      `json:"image"`
	Released    *string         `json:"released"`
	Languages   []string        `json:"languages"`
	Length      *int            `json:"length_minutes"`
	Developers  []vndbDeveloper `json:"developers"`
	Rating      *float64        `json:"rating"`
	VoteCount   int             `json:"votecount"`
	Tags        []vndbTag       `json:"tags"`
}

type vndbTitle struct {
	Language string  `json:"lang"`
	Title    string  `json:"title"`
	Latin    *string `json:"latin"`
	Official bool    `json:"official"`
	Main     bool    `json:"main"`
}

type vndbImage struct {
	URL string `json:"url"`
}

type vndbDeveloper struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type vndbTag struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Spoiler int     `json:"spoiler"`
	Rating  float64 `json:"rating"`
}

func NewVNDBProvider(client *http.Client) *VNDBProvider {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &VNDBProvider{
		client:   client,
		endpoint: vndbEndpoint,
		limiter:  rate.NewLimiter(rate.Every(1500*time.Millisecond), 1),
	}
}

func NewVNDBProviderWithEndpoint(client *http.Client, endpoint string) *VNDBProvider {
	provider := NewVNDBProvider(client)
	provider.endpoint = endpoint
	provider.limiter = rate.NewLimiter(rate.Inf, 1)
	return provider
}

func (p *VNDBProvider) Search(ctx context.Context, query string, limit int) ([]ExternalGame, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("search query is required")
	}
	if limit <= 0 {
		limit = defaultVNDBLimit
	}
	if limit > maxVNDBLimit {
		limit = maxVNDBLimit
	}

	result, err := p.request(ctx, vndbRequest{
		Filters: []any{"search", "=", query},
		Fields:  vndbFields,
		Sort:    "searchrank",
		Results: limit,
	})
	if err != nil {
		return nil, err
	}
	return mapVNDBResults(result.Results)
}

func (p *VNDBProvider) Get(ctx context.Context, externalID string) (*ExternalGame, error) {
	externalID = strings.TrimSpace(externalID)
	if !vndbIDPattern.MatchString(externalID) {
		return nil, errors.New("invalid VNDB ID")
	}

	result, err := p.request(ctx, vndbRequest{
		Filters: []any{"id", "=", externalID},
		Fields:  vndbFields,
		Results: 1,
	})
	if err != nil {
		return nil, err
	}
	games, err := mapVNDBResults(result.Results)
	if err != nil {
		return nil, err
	}
	if len(games) == 0 {
		return nil, nil
	}
	return &games[0], nil
}

func (p *VNDBProvider) ListBatch(
	ctx context.Context,
	filters BatchFilters,
	page, results int,
	withCount bool,
) (*BatchListResult, error) {
	if page < 1 {
		page = 1
	}
	if results <= 0 || results > maxVNDBLimit {
		results = maxVNDBLimit
	}

	predicates := make([][]any, 0, 5)
	if filters.MinRating > 0 {
		scaled := int(math.Round(filters.MinRating * 10))
		if scaled < 10 {
			scaled = 10
		}
		predicates = append(predicates, []any{"rating", ">=", scaled})
	}
	if filters.MinVoteCount > 0 {
		predicates = append(predicates, []any{"votecount", ">=", filters.MinVoteCount})
	}
	if filters.FromYear > 0 && filters.FromYear <= 9999 {
		predicates = append(predicates, []any{"released", ">=", fmt.Sprintf("%04d-01-01", filters.FromYear)})
	}
	if filters.ToYear > 0 && filters.ToYear <= 9999 {
		predicates = append(predicates, []any{"released", "<", fmt.Sprintf("%04d-01-01", filters.ToYear+1)})
	}
	if strings.TrimSpace(filters.OriginalLanguage) != "" {
		predicates = append(predicates, []any{"lang", "=", strings.TrimSpace(filters.OriginalLanguage)})
	}
	// VNDB requires at least two predicates inside an "and" expression.
	for len(predicates) < 2 {
		predicates = append(predicates, []any{"votecount", ">=", 0})
	}

	filter := make([]any, 0, len(predicates)+1)
	filter = append(filter, "and")
	for _, predicate := range predicates {
		filter = append(filter, predicate)
	}

	result, err := p.request(ctx, vndbRequest{
		Filters: filter,
		Fields:  vndbFields,
		Sort:    "rating",
		Reverse: true,
		Results: results,
		Page:    page,
		Count:   withCount,
	})
	if err != nil {
		return nil, err
	}
	games, err := mapVNDBResults(result.Results)
	if err != nil {
		return nil, err
	}
	count := int64(-1)
	if withCount {
		count = int64(result.Count)
	}
	return &BatchListResult{Games: games, More: result.More, Count: count}, nil
}

func (p *VNDBProvider) request(ctx context.Context, payload vndbRequest) (*vndbResponse, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("wait for VNDB rate limit: %w", err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode VNDB request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create VNDB request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Koyomi-Gal/1.0")

	response, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request VNDB: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("VNDB returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(message)))
	}

	var result vndbResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, 10<<20))
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("decode VNDB response: %w", err)
	}
	return &result, nil
}

func mapVNDBResults(results []json.RawMessage) ([]ExternalGame, error) {
	games := make([]ExternalGame, 0, len(results))
	for _, raw := range results {
		var item vndbGame
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("decode VNDB game: %w", err)
		}
		game, err := mapVNDBGame(item, raw)
		if err != nil {
			return nil, fmt.Errorf("map VNDB game %s: %w", item.ID, err)
		}
		games = append(games, game)
	}
	return games, nil
}

func mapVNDBGame(item vndbGame, raw json.RawMessage) (ExternalGame, error) {
	releaseDate, err := parseVNDBDate(item.Released)
	if err != nil {
		return ExternalGame{}, err
	}

	title := strings.TrimSpace(item.Title)
	originalTitle := ""
	originalLanguage := ""
	aliases := append([]string(nil), item.Aliases...)
	for _, candidate := range item.Titles {
		candidateTitle := strings.TrimSpace(candidate.Title)
		if candidateTitle == "" {
			continue
		}
		aliases = append(aliases, candidateTitle)
		if candidate.Latin != nil {
			aliases = append(aliases, strings.TrimSpace(*candidate.Latin))
		}
		if candidate.Main {
			originalLanguage = candidate.Language
		}
		if strings.HasPrefix(candidate.Language, "zh") && candidate.Official {
			title = candidateTitle
		}
		if candidate.Language == "ja" && originalTitle == "" {
			originalTitle = candidateTitle
		}
	}
	if originalTitle == "" && item.AltTitle != nil {
		originalTitle = strings.TrimSpace(*item.AltTitle)
	}
	if originalLanguage == "" && len(item.Languages) > 0 {
		originalLanguage = item.Languages[0]
	}

	var coverURL string
	if item.Image != nil {
		coverURL = item.Image.URL
	}
	var description string
	if item.Description != nil {
		description = *item.Description
	}
	var developer *ExternalDeveloper
	for _, candidate := range item.Developers {
		if strings.TrimSpace(candidate.Name) != "" {
			developer = &ExternalDeveloper{ExternalID: candidate.ID, Name: strings.TrimSpace(candidate.Name)}
			break
		}
	}
	tags := make([]ExternalTag, 0, len(item.Tags))
	for _, candidate := range item.Tags {
		tags = append(tags, ExternalTag{
			ExternalID: candidate.ID,
			Name:       strings.TrimSpace(candidate.Name),
			Spoiler:    candidate.Spoiler,
			Rating:     candidate.Rating,
		})
	}
	var rating *float64
	if item.Rating != nil {
		normalized := *item.Rating / 10
		rating = &normalized
	}
	ratingCount := item.VoteCount

	return ExternalGame{
		Source:           vndbSource,
		ExternalID:       item.ID,
		Title:            title,
		OriginalTitle:    originalTitle,
		RomajiTitle:      strings.TrimSpace(item.Title),
		Aliases:          aliases,
		Description:      description,
		CoverURL:         coverURL,
		ReleaseDate:      releaseDate,
		OriginalLanguage: originalLanguage,
		LengthMinutes:    item.Length,
		Rating:           rating,
		RatingCount:      &ratingCount,
		Developer:        developer,
		Tags:             tags,
		Raw:              append(json.RawMessage(nil), raw...),
	}, nil
}

func parseVNDBDate(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" || strings.EqualFold(strings.TrimSpace(*value), "TBA") {
		return nil, nil
	}
	date := strings.TrimSpace(*value)
	layout := "2006-01-02"
	switch len(date) {
	case 4:
		layout = "2006"
	case 7:
		layout = "2006-01"
	}
	parsed, err := time.Parse(layout, date)
	if err != nil {
		return nil, fmt.Errorf("invalid release date %q: %w", date, err)
	}
	return &parsed, nil
}
