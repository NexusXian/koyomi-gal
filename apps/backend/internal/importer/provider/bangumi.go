package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	bangumiSource         = "bangumi"
	bangumiDefaultBaseURL = "https://api.bgm.tv"
	bangumiUserAgent      = "Koyomi-Gal/1.0 (koyomi-gal backend)"
	defaultBangumiLimit   = 20
	maxBangumiLimit       = 50
	// bangumiSubjectTypeGame is the Bangumi subject type for games (visual
	// novels are a subset; match quality filtering happens in the service).
	bangumiSubjectTypeGame = 4
)

var bangumiIDPattern = regexp.MustCompile(`^[1-9][0-9]*$`)

type BangumiProvider struct {
	client  *http.Client
	baseURL string
	token   string
	limiter *rate.Limiter
}

type bangumiSearchRequest struct {
	Keyword string              `json:"keyword"`
	Filter  bangumiSearchFilter `json:"filter"`
	Limit   int                 `json:"limit,omitempty"`
	Offset  int                 `json:"offset,omitempty"`
}

type bangumiSearchFilter struct {
	Type []int `json:"type,omitempty"`
}

type bangumiSearchResponse struct {
	Data []json.RawMessage `json:"data"`
}

type bangumiSubject struct {
	ID       int              `json:"id"`
	Type     int              `json:"type"`
	Name     string           `json:"name"`
	NameCN   string           `json:"name_cn"`
	Summary  string           `json:"summary"`
	Date     string           `json:"date"`
	Platform string           `json:"platform"`
	Images   *bangumiImages   `json:"images"`
	Rating   *bangumiRating   `json:"rating"`
	Tags     []bangumiTag     `json:"tags"`
	Infobox  []bangumiInfobox `json:"infobox"`
}

type bangumiImages struct {
	Large  string `json:"large"`
	Common string `json:"common"`
	Medium string `json:"medium"`
	Small  string `json:"small"`
	Grid   string `json:"grid"`
}

type bangumiRating struct {
	Rank  int     `json:"rank"`
	Total int     `json:"total"`
	Score float64 `json:"score"`
}

type bangumiTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// bangumiInfobox models infobox entries whose value is either a plain string
// or a list of {k, v} pairs, e.g. aliases.
type bangumiInfobox struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

func NewBangumiProvider(client *http.Client, baseURL, token string) *BangumiProvider {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = bangumiDefaultBaseURL
	}
	return &BangumiProvider{
		client:  client,
		baseURL: baseURL,
		token:   strings.TrimSpace(token),
		limiter: rate.NewLimiter(rate.Every(500*time.Millisecond), 1),
	}
}

func NewBangumiProviderWithEndpoint(client *http.Client, baseURL, token string) *BangumiProvider {
	provider := NewBangumiProvider(client, baseURL, token)
	provider.limiter = rate.NewLimiter(rate.Inf, 1)
	return provider
}

func (p *BangumiProvider) Search(ctx context.Context, query string, limit int) ([]ExternalGame, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("search query is required")
	}
	if limit <= 0 {
		limit = defaultBangumiLimit
	}
	if limit > maxBangumiLimit {
		limit = maxBangumiLimit
	}

	payload, err := json.Marshal(bangumiSearchRequest{
		Keyword: query,
		Filter:  bangumiSearchFilter{Type: []int{bangumiSubjectTypeGame}},
		Limit:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Bangumi search request: %w", err)
	}
	body, err := p.do(ctx, http.MethodPost, "/v0/search/subjects", payload)
	if err != nil {
		return nil, err
	}
	var result bangumiSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode Bangumi search response: %w", err)
	}
	return p.mapSubjects(result.Data)
}

func (p *BangumiProvider) Get(ctx context.Context, externalID string) (*ExternalGame, error) {
	externalID = strings.TrimSpace(externalID)
	if !bangumiIDPattern.MatchString(externalID) {
		return nil, errors.New("invalid Bangumi subject ID")
	}

	body, err := p.do(ctx, http.MethodGet, "/v0/subjects/"+externalID, nil)
	if err != nil {
		if errors.Is(err, errBangumiNotFound) {
			return nil, nil
		}
		return nil, err
	}
	games, err := p.mapSubjects([]json.RawMessage{body})
	if err != nil {
		return nil, err
	}
	if len(games) == 0 {
		return nil, nil
	}
	return &games[0], nil
}

func (p *BangumiProvider) do(ctx context.Context, method, path string, payload []byte) ([]byte, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("wait for Bangumi rate limit: %w", err)
	}
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create Bangumi request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", bangumiUserAgent)
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	response, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Bangumi: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, errBangumiNotFound
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("Bangumi returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(message)))
	}
	return io.ReadAll(io.LimitReader(response.Body, 10<<20))
}

var errBangumiNotFound = errors.New("bangumi subject not found")

func (p *BangumiProvider) mapSubjects(results []json.RawMessage) ([]ExternalGame, error) {
	games := make([]ExternalGame, 0, len(results))
	for _, raw := range results {
		var subject bangumiSubject
		if err := json.Unmarshal(raw, &subject); err != nil {
			return nil, fmt.Errorf("decode Bangumi subject: %w", err)
		}
		if subject.ID == 0 || subject.Type != bangumiSubjectTypeGame {
			continue
		}
		games = append(games, mapBangumiSubject(subject, raw))
	}
	return games, nil
}

func mapBangumiSubject(subject bangumiSubject, raw json.RawMessage) ExternalGame {
	name := strings.TrimSpace(subject.Name)
	nameCN := strings.TrimSpace(subject.NameCN)

	var rating *float64
	var ratingCount *int
	if subject.Rating != nil && subject.Rating.Score > 0 {
		score := subject.Rating.Score
		rating = &score
		count := subject.Rating.Total
		ratingCount = &count
	}

	return ExternalGame{
		Source:           bangumiSource,
		ExternalID:       strconv.Itoa(subject.ID),
		Title:            bangumiTitle(nameCN, name),
		OriginalTitle:    name,
		RomajiTitle:      "",
		Aliases:          bangumiAliases(name, nameCN, subject.Infobox),
		Description:      strings.TrimSpace(subject.Summary),
		CoverURL:         bangumiCoverURL(subject.Images),
		ReleaseDate:      parseBangumiDate(subject.Date),
		OriginalLanguage: "",
		Rating:           rating,
		RatingCount:      ratingCount,
		Developer:        bangumiDeveloper(subject.Infobox),
		Tags:             bangumiTags(subject.Tags),
		Raw:              append(json.RawMessage(nil), raw...),
	}
}

func bangumiTitle(nameCN, name string) string {
	if nameCN != "" {
		return nameCN
	}
	return name
}

func bangumiCoverURL(images *bangumiImages) string {
	if images == nil {
		return ""
	}
	for _, candidate := range []string{images.Large, images.Common, images.Medium, images.Small} {
		if strings.TrimSpace(candidate) != "" {
			return strings.TrimSpace(candidate)
		}
	}
	return ""
}

func parseBangumiDate(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	layout := "2006-01-02"
	switch len(value) {
	case 4:
		layout = "2006"
	case 7:
		layout = "2006-01"
	}
	parsed, err := time.Parse(layout, value)
	if err != nil {
		return nil
	}
	return &parsed
}

// bangumiInfoboxKeys carries the infobox keys that identify developer names.
var bangumiDeveloperKeys = []string{"开发", "开发商", "游戏开发商", "制作"}

func bangumiDeveloper(infobox []bangumiInfobox) *ExternalDeveloper {
	for _, item := range infobox {
		for _, key := range bangumiDeveloperKeys {
			if item.Key != key {
				continue
			}
			if value := infoboxStringValue(item.Value); value != "" {
				return &ExternalDeveloper{ExternalID: "", Name: value}
			}
		}
	}
	return nil
}

func bangumiAliases(name, nameCN string, infobox []bangumiInfobox) []string {
	values := make([]string, 0, 8)
	values = append(values, name, nameCN)
	for _, item := range infobox {
		if item.Key != "别名" && item.Key != "中文名" {
			continue
		}
		if value := infoboxStringValue(item.Value); value != "" {
			values = append(values, value)
			continue
		}
		values = append(values, infoboxListValues(item.Value)...)
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		for _, alias := range strings.FieldsFunc(value, func(r rune) bool {
			return r == '、' || r == '，' || r == ',' || r == '\n'
		}) {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			key := strings.ToLower(alias)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, alias)
		}
	}
	return result
}

func bangumiTags(tags []bangumiTag) []ExternalTag {
	result := make([]ExternalTag, 0, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}
		result = append(result, ExternalTag{
			ExternalID: "",
			Name:       name,
			Spoiler:    0,
			Rating:     float64(tag.Count),
		})
	}
	return result
}

// infoboxStringValue extracts a plain string infobox value.
func infoboxStringValue(raw json.RawMessage) string {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

// infoboxListValues extracts {v} entries from list-shaped infobox values.
func infoboxListValues(raw json.RawMessage) []string {
	var items []struct {
		Key   string `json:"k"`
		Value string `json:"v"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		if value := strings.TrimSpace(item.Value); value != "" {
			values = append(values, value)
		}
	}
	return values
}
