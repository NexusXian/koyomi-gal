package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestVNDBServer(t *testing.T, handler http.HandlerFunc) *VNDBProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewVNDBProviderWithEndpoint(server.Client(), server.URL)
}

const vndbGameJSON = `{
  "id": "v17",
  "title": "Ever17 -the out of infinity-",
  "alttitle": "エバーセブンティーン",
  "titles": [
    {"lang": "ja", "title": "エバーセブンティーン", "latin": null, "official": true, "main": true},
    {"lang": "zh-Hans", "title": "时光的羁绊", "latin": null, "official": true, "main": false},
    {"lang": "en", "title": "Ever17", "latin": "Ever17 -the out of infinity-", "official": true, "main": false}
  ],
  "aliases": ["E17", "エバーセブンティーン"],
  "description": "A sci-fi visual novel.",
  "image": {"url": "https://t.vndb.org/cv/17.jpg"},
  "released": "2002-08-29",
  "languages": ["ja", "en"],
  "length_minutes": 1800,
  "developers": [{"id": "p9", "name": "KID"}],
  "rating": 85.23,
  "votecount": 5000,
  "tags": [
    {"id": "g1", "name": "Sci-Fi", "spoiler": 0, "rating": 2.9},
    {"id": "g2", "name": "Plot Twist", "spoiler": 2, "rating": 3.0}
  ]
}`

func TestVNDBSearchMapsResponse(t *testing.T) {
	var capturedRequest map[string]any
	vndb := newTestVNDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results": [` + vndbGameJSON + `], "more": false, "count": 1}`))
	})

	games, err := vndb.Search(context.Background(), "ever17", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("games = %d, want 1", len(games))
	}

	if filters, _ := capturedRequest["filters"].([]any); len(filters) != 3 || filters[0] != "search" {
		t.Errorf("filters = %v, want search predicate", capturedRequest["filters"])
	}
	if capturedRequest["sort"] != "searchrank" {
		t.Errorf("sort = %v, want searchrank", capturedRequest["sort"])
	}
	if capturedRequest["results"] != float64(5) {
		t.Errorf("results = %v, want 5", capturedRequest["results"])
	}

	game := games[0]
	if game.Source != "vndb" || game.ExternalID != "v17" {
		t.Errorf("identity = %s/%s", game.Source, game.ExternalID)
	}
	// zh-Hans official title wins display title.
	if game.Title != "时光的羁绊" {
		t.Errorf("title = %q, want zh-Hans official title", game.Title)
	}
	if game.OriginalTitle != "エバーセブンティーン" {
		t.Errorf("original title = %q", game.OriginalTitle)
	}
	if game.RomajiTitle != "Ever17 -the out of infinity-" {
		t.Errorf("romaji title = %q", game.RomajiTitle)
	}
	if game.OriginalLanguage != "ja" {
		t.Errorf("original language = %q, want main title language ja", game.OriginalLanguage)
	}
	if game.Description != "A sci-fi visual novel." {
		t.Errorf("description = %q", game.Description)
	}
	if game.CoverURL != "https://t.vndb.org/cv/17.jpg" {
		t.Errorf("cover url = %q", game.CoverURL)
	}
	wantDate := time.Date(2002, time.August, 29, 0, 0, 0, 0, time.UTC)
	if game.ReleaseDate == nil || !game.ReleaseDate.Equal(wantDate) {
		t.Errorf("release date = %v, want %v", game.ReleaseDate, wantDate)
	}
	if game.LengthMinutes == nil || *game.LengthMinutes != 1800 {
		t.Errorf("length minutes = %v", game.LengthMinutes)
	}
	if game.Rating == nil || *game.Rating < 8.52 || *game.Rating > 8.53 {
		t.Errorf("rating = %v, want 85.23 normalized to ~8.523", game.Rating)
	}
	if game.RatingCount == nil || *game.RatingCount != 5000 {
		t.Errorf("rating count = %v", game.RatingCount)
	}
	if game.Developer == nil || game.Developer.Name != "KID" || game.Developer.ExternalID != "p9" {
		t.Errorf("developer = %+v", game.Developer)
	}
	if len(game.Tags) != 2 {
		t.Fatalf("tags = %d, want 2 (provider keeps filtering to the mapper)", len(game.Tags))
	}
	if len(game.Raw) == 0 || !strings.Contains(string(game.Raw), `"v17"`) {
		t.Errorf("raw metadata not preserved")
	}

	aliasSet := make(map[string]bool)
	for _, alias := range game.Aliases {
		aliasSet[strings.ToLower(alias)] = true
	}
	for _, want := range []string{"e17", "ever17", "时光的羁绊"} {
		if !aliasSet[want] {
			t.Errorf("aliases missing %q: %v", want, game.Aliases)
		}
	}
}

func TestVNDBSearchDeduplicatesTitleAliases(t *testing.T) {
	vndb := newTestVNDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results": [{
			"id": "v1",
			"title": "Game",
			"titles": [{"lang": "ja", "title": "ゲーム", "official": true, "main": true}],
			"aliases": ["Game", " GAMES "],
			"languages": ["ja"],
			"votecount": 1
		}]}`))
	})
	games, err := vndb.Search(context.Background(), "game", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	counts := make(map[string]int)
	for _, alias := range games[0].Aliases {
		counts[strings.ToLower(strings.TrimSpace(alias))]++
	}
	for alias, count := range counts {
		if count > 1 {
			t.Errorf("alias %q appears %d times in %v", alias, count, games[0].Aliases)
		}
	}
}

func TestVNDBPartialReleaseDates(t *testing.T) {
	cases := map[string]*time.Time{
		`"2018-06"`:    timePtr(time.Date(2018, time.June, 1, 0, 0, 0, 0, time.UTC)),
		`"2018"`:       timePtr(time.Date(2018, time.January, 1, 0, 0, 0, 0, time.UTC)),
		`"TBA"`:        nil,
		`null`:         nil,
		`"2020-02-29"`: timePtr(time.Date(2020, time.February, 29, 0, 0, 0, 0, time.UTC)),
	}
	for input, want := range cases {
		vndb := newTestVNDBServer(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"results": [{
				"id": "v1", "title": "Game", "released": ` + input + `,
				"titles": [{"lang": "ja", "title": "げーむ", "official": true, "main": true}],
				"languages": ["ja"], "votecount": 0
			}]}`))
		})
		game, err := vndb.Get(context.Background(), "v1")
		if err != nil {
			t.Fatalf("get with released %s: %v", input, err)
		}
		if (game.ReleaseDate == nil) != (want == nil) {
			t.Errorf("released %s: got %v, want %v", input, game.ReleaseDate, want)
			continue
		}
		if want != nil && !game.ReleaseDate.Equal(*want) {
			t.Errorf("released %s: got %v, want %v", input, game.ReleaseDate, *want)
		}
	}
}

func TestVNDBInvalidReleaseDateIsError(t *testing.T) {
	vndb := newTestVNDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results": [{
			"id": "v1", "title": "Game", "released": "sometime",
			"titles": [], "languages": [], "votecount": 0
		}]}`))
	})
	if _, err := vndb.Get(context.Background(), "v1"); err == nil {
		t.Fatal("expected error for invalid release date")
	}
}

func TestVNDBGetValidatesID(t *testing.T) {
	vndb := newTestVNDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called for an invalid ID")
	})
	if _, err := vndb.Get(context.Background(), "17"); err == nil {
		t.Error("expected error for ID without v prefix")
	}
	if _, err := vndb.Get(context.Background(), "v0"); err == nil {
		t.Error("expected error for v0")
	}
}

func TestVNDBGetNotFoundReturnsNil(t *testing.T) {
	vndb := newTestVNDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results": [], "more": false}`))
	})
	game, err := vndb.Get(context.Background(), "v999999")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if game != nil {
		t.Errorf("game = %+v, want nil", game)
	}
}

func TestVNDBNon2xxIsError(t *testing.T) {
	vndb := newTestVNDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream down"))
	})
	_, err := vndb.Search(context.Background(), "game", 10)
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err = %v, want HTTP 502 failure", err)
	}
}

func TestVNDBListBatchBuildsFilters(t *testing.T) {
	var capturedRequest map[string]any
	vndb := newTestVNDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"results": [], "more": false, "count": 42}`))
	})

	result, err := vndb.ListBatch(context.Background(), BatchFilters{
		MinRating:        7.5,
		MinVoteCount:     100,
		FromYear:         2010,
		ToYear:           2020,
		OriginalLanguage: "ja",
	}, 2, 100, true)
	if err != nil {
		t.Fatalf("list batch: %v", err)
	}
	if result.Count != 42 {
		t.Errorf("count = %d, want 42", result.Count)
	}

	filters, _ := capturedRequest["filters"].([]any)
	if len(filters) < 3 || filters[0] != "and" {
		t.Fatalf("filters = %v, want and expression", capturedRequest["filters"])
	}
	rendered := fmtFilters(filters)
	if !strings.Contains(rendered, "rating") || !strings.Contains(rendered, "75") {
		t.Errorf("filters missing rating >= 75: %v", rendered)
	}
	if !strings.Contains(rendered, "votecount") || !strings.Contains(rendered, "100") {
		t.Errorf("filters missing votecount >= 100: %v", rendered)
	}
	if !strings.Contains(rendered, "2010-01-01") || !strings.Contains(rendered, "2021-01-01") {
		t.Errorf("filters missing released year range: %v", rendered)
	}
	if !strings.Contains(rendered, "lang") || !strings.Contains(rendered, "ja") {
		t.Errorf("filters missing lang = ja: %v", rendered)
	}
	if capturedRequest["page"] != float64(2) {
		t.Errorf("page = %v, want 2", capturedRequest["page"])
	}
	if capturedRequest["count"] != true {
		t.Errorf("count flag = %v, want true", capturedRequest["count"])
	}
}

func fmtFilters(filters []any) string {
	encoded, _ := json.Marshal(filters)
	return string(encoded)
}

func timePtr(value time.Time) *time.Time {
	return &value
}
