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

func newTestBangumiServer(t *testing.T, handler http.HandlerFunc) *BangumiProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewBangumiProviderWithEndpoint(server.Client(), server.URL, "test-token")
}

const bangumiSubjectJSON = `{
  "id": 200763,
  "type": 4,
  "name": "Summer Pockets",
  "name_cn": "夏日口袋",
  "summary": "鳥白島での夏休みを描く作品。",
  "date": "2018-06-29",
  "platform": "游戏",
  "images": {
    "large": "https://lain.bgm.tv/pic/cover/l/200763.jpg",
    "common": "https://lain.bgm.tv/r/400/pic/cover/l/200763.jpg",
    "medium": "", "small": "", "grid": ""
  },
  "rating": {"rank": 198, "total": 5819, "score": 8.2, "count": {}},
  "tags": [
    {"name": "key", "count": 2362},
    {"name": "Galgame", "count": 2092},
    {"name": "催泪", "count": 328}
  ],
  "infobox": [
    {"key": "中文名", "value": "夏日口袋"},
    {"key": "别名", "value": [
      {"v": "サマーポケッツ"},
      {"v": "サマポケ、SP"},
      {"v": "Summer Pokemon、夏日岛可梦、夏日蒐集"}
    ]},
    {"key": "平台", "value": "PC"},
    {"key": "开发", "value": "VISUAL ARTS / Key"}
  ],
  "collection": {"wish": 2336, "collect": 7363, "doing": 765, "on_hold": 361, "dropped": 117}
}`

func TestBangumiSearchMapsResponse(t *testing.T) {
	var capturedRequest map[string]any
	var capturedMethod string
	var capturedAuth string
	bgm := newTestBangumiServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": [` + bangumiSubjectJSON + `], "total": 1, "limit": 5, "offset": 0}`))
	})

	games, err := bgm.Search(context.Background(), "summer pockets", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("games = %d, want 1", len(games))
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", capturedMethod)
	}
	if capturedAuth != "Bearer test-token" {
		t.Errorf("authorization = %q, want bearer token", capturedAuth)
	}
	if capturedRequest["keyword"] != "summer pockets" {
		t.Errorf("keyword = %v", capturedRequest["keyword"])
	}
	filter, _ := capturedRequest["filter"].(map[string]any)
	types, _ := filter["type"].([]any)
	if len(types) != 1 || types[0] != float64(4) {
		t.Errorf("filter.type = %v, want [4] (game only)", filter["type"])
	}
	if capturedRequest["limit"] != float64(5) {
		t.Errorf("limit = %v, want 5", capturedRequest["limit"])
	}

	game := games[0]
	if game.Source != "bangumi" || game.ExternalID != "200763" {
		t.Errorf("identity = %s/%s", game.Source, game.ExternalID)
	}
	if game.Title != "夏日口袋" {
		t.Errorf("title = %q, want Chinese name", game.Title)
	}
	if game.OriginalTitle != "Summer Pockets" {
		t.Errorf("original title = %q", game.OriginalTitle)
	}
	if game.Description == "" || !strings.Contains(game.Description, "鳥白島") {
		t.Errorf("description = %q", game.Description)
	}
	if game.CoverURL != "https://lain.bgm.tv/pic/cover/l/200763.jpg" {
		t.Errorf("cover url = %q", game.CoverURL)
	}
	wantDate := time.Date(2018, time.June, 29, 0, 0, 0, 0, time.UTC)
	if game.ReleaseDate == nil || !game.ReleaseDate.Equal(wantDate) {
		t.Errorf("release date = %v, want %v", game.ReleaseDate, wantDate)
	}
	if game.Rating == nil || *game.Rating != 8.2 {
		t.Errorf("rating = %v, want 8.2", game.Rating)
	}
	if game.RatingCount == nil || *game.RatingCount != 5819 {
		t.Errorf("rating count = %v, want 5819", game.RatingCount)
	}
	if game.Developer == nil || game.Developer.Name != "VISUAL ARTS / Key" {
		t.Errorf("developer = %+v, want infobox 开发", game.Developer)
	}
	if len(game.Raw) == 0 || !strings.Contains(string(game.Raw), "200763") {
		t.Errorf("raw metadata not preserved")
	}

	aliasSet := make(map[string]bool)
	for _, alias := range game.Aliases {
		aliasSet[alias] = true
	}
	for _, want := range []string{
		"Summer Pockets", "夏日口袋",
		"サマーポケッツ", "サマポケ", "SP",
		"Summer Pokemon", "夏日岛可梦", "夏日蒐集",
	} {
		if !aliasSet[want] {
			t.Errorf("aliases missing %q: %v", want, game.Aliases)
		}
	}

	if len(game.Tags) != 3 {
		t.Fatalf("tags = %d, want 3", len(game.Tags))
	}
	if game.Tags[0].Name != "key" || game.Tags[0].Rating != 2362 {
		t.Errorf("first tag = %+v, want key with count as rating", game.Tags[0])
	}
}

func TestBangumiSearchFallsBackToOriginalName(t *testing.T) {
	bgm := newTestBangumiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data": [{
			"id": 34032, "type": 4, "name": "_summer##", "name_cn": "",
			"date": "2006-08-24", "tags": [], "infobox": []
		}], "total": 1}`))
	})
	games, err := bgm.Search(context.Background(), "_summer", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if games[0].Title != "_summer##" {
		t.Errorf("title = %q, want fallback to name", games[0].Title)
	}
}

func TestBangumiSearchDropsNonGameSubjects(t *testing.T) {
	bgm := newTestBangumiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data": [
			{"id": 195311, "type": 1, "name": "神とよばれた吸血鬼 (4)", "date": "2016-09-21", "tags": [], "infobox": []},
			{"id": 200763, "type": 4, "name": "Summer Pockets", "name_cn": "夏日口袋", "date": "2018-06-29", "tags": [], "infobox": []}
		], "total": 2}`))
	})
	games, err := bgm.Search(context.Background(), "summer", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(games) != 1 || games[0].ExternalID != "200763" {
		t.Fatalf("games = %+v, want only the type-4 game", games)
	}
}

func TestBangumiSearchRequiresQuery(t *testing.T) {
	bgm := newTestBangumiServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called for an empty query")
	})
	if _, err := bgm.Search(context.Background(), "  ", 5); err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestBangumiGetMapsSubject(t *testing.T) {
	var capturedPath string
	bgm := newTestBangumiServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = w.Write([]byte(bangumiSubjectJSON))
	})
	game, err := bgm.Get(context.Background(), "200763")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if capturedPath != "/v0/subjects/200763" {
		t.Errorf("path = %q", capturedPath)
	}
	if game == nil || game.Title != "夏日口袋" || game.RatingCount == nil || *game.RatingCount != 5819 {
		t.Fatalf("game = %+v", game)
	}
}

func TestBangumiGetValidatesID(t *testing.T) {
	bgm := newTestBangumiServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called for an invalid ID")
	})
	if _, err := bgm.Get(context.Background(), "abc"); err == nil {
		t.Error("expected error for non-numeric ID")
	}
	if _, err := bgm.Get(context.Background(), "0"); err == nil {
		t.Error("expected error for zero ID")
	}
	if _, err := bgm.Get(context.Background(), "-1"); err == nil {
		t.Error("expected error for negative ID")
	}
}

func TestBangumiGetNotFoundReturnsNil(t *testing.T) {
	bgm := newTestBangumiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":404,"error":"Not Found"}`))
	})
	game, err := bgm.Get(context.Background(), "999999")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if game != nil {
		t.Errorf("game = %+v, want nil", game)
	}
}

func TestBangumiRateLimitIsSent(t *testing.T) {
	bgm := newTestBangumiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data": [], "total": 0}`))
	})
	if bgm.limiter.Limit() <= 0 {
		t.Error("production provider must rate limit requests")
	}
	if _, err := bgm.Search(context.Background(), "x", 1); err != nil {
		t.Fatalf("search: %v", err)
	}
}

func TestBangumiNon2xxIsError(t *testing.T) {
	bgm := newTestBangumiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"code":429,"error":"Too Many Requests"}`))
	})
	_, err := bgm.Search(context.Background(), "game", 10)
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("err = %v, want HTTP 429 failure", err)
	}
}

func TestBangumiUserAgentIsSent(t *testing.T) {
	var capturedUA string
	bgm := newTestBangumiServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{"data": [], "total": 0}`))
	})
	if _, err := bgm.Search(context.Background(), "x", 1); err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.HasPrefix(capturedUA, "Koyomi-Gal/") {
		t.Errorf("user agent = %q", capturedUA)
	}
}

func TestBangumiPartialDates(t *testing.T) {
	cases := map[string]*time.Time{
		"2018-06-29": timePtr(time.Date(2018, time.June, 29, 0, 0, 0, 0, time.UTC)),
		"2018-06":    timePtr(time.Date(2018, time.June, 1, 0, 0, 0, 0, time.UTC)),
		"2018":       timePtr(time.Date(2018, time.January, 1, 0, 0, 0, 0, time.UTC)),
		"":           nil,
	}
	for input, want := range cases {
		got := parseBangumiDate(input)
		if (got == nil) != (want == nil) {
			t.Errorf("parseBangumiDate(%q) = %v, want %v", input, got, want)
			continue
		}
		if want != nil && !got.Equal(*want) {
			t.Errorf("parseBangumiDate(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestBangumiZeroRatingIsOmitted(t *testing.T) {
	bgm := newTestBangumiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id": 34032, "type": 4, "name": "_summer##", "date": "2006-08-24",
			"rating": {"rank": 0, "total": 0, "score": 0}, "tags": [], "infobox": []}`))
	})
	game, err := bgm.Get(context.Background(), "34032")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if game.Rating != nil || game.RatingCount != nil {
		t.Errorf("rating = %v/%v, want nil for unscored subject", game.Rating, game.RatingCount)
	}
}
