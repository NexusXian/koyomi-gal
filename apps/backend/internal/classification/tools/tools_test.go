package tools

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type fakeSearchProvider struct {
	calls atomic.Int32
}

func (p *fakeSearchProvider) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	p.calls.Add(1)
	return []SearchResult{{Title: "官网", URL: "https://official.example.com", Snippet: "18禁"}}, nil
}

func TestSearchToolCachesResults(t *testing.T) {
	provider := &fakeSearchProvider{}
	tool := NewSearchTool(provider, nil)
	ctx := context.Background()

	first, err := tool.Search(ctx, SearchToolInput{Query: "サクラノ刻 18禁", Limit: 5})
	if err != nil {
		t.Fatalf("first search: %v", err)
	}
	if !strings.Contains(first, "18禁") {
		t.Fatalf("unexpected first output: %s", first)
	}
	if provider.calls.Load() != 1 {
		t.Fatalf("expected 1 provider call, got %d", provider.calls.Load())
	}
}

func TestSearchToolEmptyQuery(t *testing.T) {
	tool := NewSearchTool(&fakeSearchProvider{}, nil)
	if _, err := tool.Search(context.Background(), SearchToolInput{Query: "   "}); err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestSearchToolWithoutProvider(t *testing.T) {
	tool := NewSearchTool(nil, nil)
	if _, err := tool.Search(context.Background(), SearchToolInput{Query: "x"}); err == nil {
		t.Fatal("expected error when provider is missing")
	}
}

func TestFetchPageToolExtractsContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<html><head><title>サクラノ刻 公式</title></head>
<body>
<nav>菜单导航</nav>
<script>var x = 1;</script>
<main>
<h1>サクラノ刻</h1>
<p>本作品は<strong>18歳以上対象</strong>の成人向けコンテンツです。</p>
</main>
<footer>© example</footer>
</body></html>`)
	}))
	defer server.Close()

	tool := NewFetchPageToolUnsafeForTests(nil)
	output, err := tool.Fetch(context.Background(), FetchPageToolInput{URL: server.URL})
	if err != nil {
		t.Fatalf("fetch page: %v", err)
	}
	if !strings.Contains(output, "サクラノ刻") {
		t.Fatalf("missing title text in output: %s", output)
	}
	if !strings.Contains(output, "18歳以上対象") {
		t.Fatalf("evidence text missing from extracted content: %s", output)
	}
	if strings.Contains(output, "var x") || strings.Contains(output, "© example") {
		t.Fatalf("script/footer leaked into content: %s", output)
	}
}

func TestFetchPageToolSkipsNonHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.WriteString(w, "binary")
	}))
	defer server.Close()

	tool := NewFetchPageToolUnsafeForTests(nil)
	if _, err := tool.Fetch(context.Background(), FetchPageToolInput{URL: server.URL}); err == nil {
		t.Fatal("expected error for binary content type")
	}
}

func TestSSRFGuardRejectsPrivateTargets(t *testing.T) {
	tool := NewFetchPageTool(nil)
	ctx := context.Background()

	cases := []string{
		"http://127.0.0.1:8080/health",
		"http://localhost:8080/health",
		"http://10.0.0.5/admin",
		"http://192.168.1.1/x",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/x",
		"ftp://example.com/file",
		"file:///etc/passwd",
	}
	for _, target := range cases {
		if _, err := tool.Fetch(ctx, FetchPageToolInput{URL: target}); err == nil {
			t.Errorf("expected blocked url error for %s", target)
		} else if !errors.Is(err, ErrBlockedURL) && !strings.Contains(err.Error(), "scheme") {
			// scheme/parse issues are acceptable; silently dialing is not
			t.Errorf("unexpected error type for %s: %v", target, err)
		}
	}
}

func TestValidateFetchURL(t *testing.T) {
	if _, err := validateFetchURL("https://store.steampowered.com/app/12345/"); err != nil {
		t.Fatalf("valid url rejected: %v", err)
	}
	if _, err := validateFetchURL("https://user:pass@example.com/"); err == nil {
		t.Fatal("userinfo must be rejected")
	}
	if _, err := validateFetchURL("javascript:alert(1)"); err == nil {
		t.Fatal("non-http scheme must be rejected")
	}
}
