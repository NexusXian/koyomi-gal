package nextmoe

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testKey = "nmk_test_secret_value"

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewClient(server.URL+"/v2/", testKey, Options{
		HTTPClient:        server.Client(),
		RequestsPerMinute: 600000,
	})
	client.sleep = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
	return client
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func writeProblem(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"status":` + strconv.Itoa(status) + `,"code":"` + code + `","request_id":"req_01J0000000000000000000000A"}`))
}

func TestFindWorkByVNDBIDExactMatchWithIntros(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/catalog/works" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("refs") != "vndb:v17" || query.Get("include") != "refs,intros" || query.Get("nsfw") != "true" {
			t.Errorf("unexpected query %s", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer "+testKey {
			t.Errorf("missing bearer token")
		}
		writeJSON(w, http.StatusOK, `{"object":"list","items":[{"object":"work","id":"42",
			"refs":[{"source":"bangumi","external_id":"v17"},{"source":"vndb","external_id":"v17"}],
			"intros":[{"lang":"en","value":"English","is_machine":false,"source":"vndb"},
			          {"lang":"zh-Hans","value":"  中文简介\r\n第二行 ","is_machine":false,"source":"bangumi"}]}]}`)
	})
	intro, err := client.GetChineseIntro(context.Background(), "v17")
	if err != nil {
		t.Fatalf("GetChineseIntro: %v", err)
	}
	if intro.WorkID != "42" || intro.Lang != "zh-Hans" || intro.Source != "bangumi" {
		t.Fatalf("unexpected intro %+v", intro)
	}
	if !strings.Contains(intro.Content, "中文简介") {
		t.Fatalf("unexpected content %q", intro.Content)
	}
}

func TestFindWorkByVNDBIDRejectsInexactRefs(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"object":"list","items":[
			{"object":"work","id":"1","refs":[{"source":"vndb","external_id":"v170"}]},
			{"object":"work","id":"2","refs":[{"source":"VNDB","external_id":"v17"}]},
			{"object":"work","id":"3","refs":[{"source":"vndb","external_id":"V17"}]}]}`)
	})
	_, err := client.FindWorkByVNDBID(context.Background(), "v17", false)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFindWorkByVNDBIDMissing(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"object":"list","items":[],"missing":["vndb:v17"]}`)
	})
	if _, err := client.FindWorkByVNDBID(context.Background(), "v17", true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFindWorkByVNDBIDAmbiguous(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"object":"list","items":[
			{"object":"work","id":"1","refs":[{"source":"vndb","external_id":"v17"}]},
			{"object":"work","id":"2","refs":[{"source":"vndb","external_id":"v17"}]}]}`)
	})
	if _, err := client.FindWorkByVNDBID(context.Background(), "v17", false); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("expected ErrAmbiguous, got %v", err)
	}
}

func TestFindWorkByVNDBIDRejectsInvalidIDWithoutRequest(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeJSON(w, http.StatusOK, `{"object":"list","items":[]}`)
	})
	for _, id := range []string{"", "17", "v0", "v01", "v17,vndb:v1", "r17"} {
		if _, err := client.FindWorkByVNDBID(context.Background(), id, false); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("id %q: expected ErrInvalidID, got %v", id, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid ids must not reach the API, got %d calls", calls.Load())
	}
}

func TestFindWorksByVNDBIDsBatchesByHundred(t *testing.T) {
	var batches []int
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		refs := strings.Split(r.URL.Query().Get("refs"), ",")
		batches = append(batches, len(refs))
		writeJSON(w, http.StatusOK, `{"object":"list","items":[]}`)
	})
	ids := make([]string, 0, 150)
	for i := 1; i <= 150; i++ {
		ids = append(ids, "v"+strconv.Itoa(i))
	}
	matches, err := client.FindWorksByVNDBIDs(context.Background(), ids, true)
	if err != nil {
		t.Fatalf("FindWorksByVNDBIDs: %v", err)
	}
	if len(batches) != 2 || batches[0] != 100 || batches[1] != 50 {
		t.Fatalf("unexpected batches %v", batches)
	}
	if len(matches) != 150 {
		t.Fatalf("expected 150 results, got %d", len(matches))
	}
}

func TestChineseIntroFallsBackToIntroEndpointWithPagination(t *testing.T) {
	var introCalls atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/catalog/works":
			writeJSON(w, http.StatusOK, `{"object":"list","items":[{"object":"work","id":"42","refs":[{"source":"vndb","external_id":"v17"}]}]}`)
		case "/v2/catalog/works/42/intros":
			introCalls.Add(1)
			if r.URL.Query().Get("cursor") == "" {
				writeJSON(w, http.StatusOK, `{"object":"list","items":[{"lang":"ja","value":"日本語","is_machine":false,"source":"vndb"}],"next_cursor":"cur_page2"}`)
				return
			}
			writeJSON(w, http.StatusOK, `{"object":"list","items":[{"lang":"zh-Hans","value":"第二页中文","is_machine":false,"source":"nextmoe"}],"next_cursor":"cur_page2"}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	work, err := client.FindWorkByVNDBID(context.Background(), "v17", false)
	if err != nil {
		t.Fatalf("FindWorkByVNDBID: %v", err)
	}
	intro, err := client.ChineseIntroForWork(context.Background(), work)
	if err != nil {
		t.Fatalf("ChineseIntroForWork: %v", err)
	}
	if intro.Content != "第二页中文" {
		t.Fatalf("unexpected content %q", intro.Content)
	}
	if introCalls.Load() != 2 {
		t.Fatalf("repeated cursor must stop pagination, got %d calls", introCalls.Load())
	}
}

func TestChineseIntroNoUsableIntro(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"object":"list","items":[{"object":"work","id":"42",
			"refs":[{"source":"vndb","external_id":"v17"}],
			"intros":[{"lang":"zh-Hant","value":"繁體","is_machine":false,"source":"a"},
			          {"lang":"zh-Hans","value":"机翻","is_machine":true,"source":"b"},
			          {"lang":"zh-Hans","value":"   ","is_machine":false,"source":"c"}]}]}`)
	})
	if _, err := client.GetChineseIntro(context.Background(), "v17"); !errors.Is(err, ErrNoIntro) {
		t.Fatalf("expected ErrNoIntro, got %v", err)
	}
}

func TestSelectChineseIntroPriority(t *testing.T) {
	intros := []Intro{
		{Lang: "zh", Value: "plain zh", Source: "a"},
		{Lang: "zh-CN", Value: "regional", Source: "a"},
		{Lang: "zh-Hans", Value: "b-text", Source: "b"},
		{Lang: "zh-Hans", Value: "a-text", Source: "a"},
		{Lang: "zh-TW", Value: "traditional", Source: "a"},
	}
	selected, ok := SelectChineseIntro(intros)
	if !ok || selected.Value != "a-text" {
		t.Fatalf("unexpected selection %+v", selected)
	}
	selected, ok = SelectChineseIntro(intros[:2])
	if !ok || selected.Value != "regional" {
		t.Fatalf("expected regional fallback, got %+v", selected)
	}
	if _, ok := SelectChineseIntro(intros[:1]); ok {
		t.Fatalf("script-less zh must be rejected")
	}
}

func TestUnauthorizedIsNotRetriedAndHidesKey(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeProblem(w, http.StatusUnauthorized, "MISSING_CREDENTIAL")
	})
	_, err := client.FindWorkByVNDBID(context.Background(), "v17", false)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("401 must not be retried, got %d calls", calls.Load())
	}
	if strings.Contains(err.Error(), testKey) {
		t.Fatalf("error leaks api key: %v", err)
	}
}

func TestRateLimitedRetriesThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			writeProblem(w, http.StatusTooManyRequests, "RATE_LIMITED")
			return
		}
		writeJSON(w, http.StatusOK, `{"object":"list","items":[{"object":"work","id":"42","refs":[{"source":"vndb","external_id":"v17"}]}]}`)
	})
	work, err := client.FindWorkByVNDBID(context.Background(), "v17", false)
	if err != nil || work.ID != "42" {
		t.Fatalf("expected retry success, got %+v %v", work, err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", calls.Load())
	}
}

func TestQuotaExceededTripsBreaker(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "3600")
		writeProblem(w, http.StatusTooManyRequests, "QUOTA_EXCEEDED")
	})
	for range 3 {
		if _, err := client.FindWorkByVNDBID(context.Background(), "v17", false); !errors.Is(err, ErrQuotaExceeded) {
			t.Fatalf("expected ErrQuotaExceeded, got %v", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("quota exhaustion must stop further calls, got %d", calls.Load())
	}
}

func TestServerErrorRetriesWithLimit(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeProblem(w, http.StatusServiceUnavailable, "UNAVAILABLE")
	})
	_, err := client.FindWorkByVNDBID(context.Background(), "v17", false)
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
	if calls.Load() != int32(defaultMaxRetries+1) {
		t.Fatalf("expected %d calls, got %d", defaultMaxRetries+1, calls.Load())
	}
}

func TestTimeoutIsReportedWithoutURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	t.Cleanup(server.Close)
	httpClient := server.Client()
	httpClient.Timeout = 50 * time.Millisecond
	client := NewClient(server.URL, testKey, Options{HTTPClient: httpClient, RequestsPerMinute: 600000, MaxRetries: 1})
	client.sleep = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
	_, err := client.FindWorkByVNDBID(context.Background(), "v17", false)
	if err == nil || !errors.Is(err, errTransport) {
		t.Fatalf("expected transport error, got %v", err)
	}
	if strings.Contains(err.Error(), "refs=") || strings.Contains(err.Error(), testKey) {
		t.Fatalf("error leaks request details: %v", err)
	}
}

func TestMalformedJSON(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"items":[{"id":`)
	})
	if _, err := client.FindWorkByVNDBID(context.Background(), "v17", false); !errors.Is(err, ErrBadResponse) {
		t.Fatalf("expected ErrBadResponse, got %v", err)
	}
}

func TestMergedWorkFollowsCurrentID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/catalog/works/42/intros":
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"status":404,"code":"ENTITY_MERGED","current_id":"77"}`))
		case "/v2/catalog/works/77/intros":
			writeJSON(w, http.StatusOK, `{"object":"list","items":[{"lang":"zh-Hans","value":"合并后","is_machine":false,"source":"x"}]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	intros, err := client.ListIntros(context.Background(), "42")
	if err != nil || len(intros) != 1 || intros[0].Value != "合并后" {
		t.Fatalf("unexpected merge result %+v %v", intros, err)
	}
}

func TestNotConfigured(t *testing.T) {
	client := NewClient("", "", Options{})
	if _, err := client.GetChineseIntro(context.Background(), "v17"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
	var nilClient *Client
	if _, err := nilClient.GetChineseIntro(context.Background(), "v17"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured for nil client, got %v", err)
	}
}
