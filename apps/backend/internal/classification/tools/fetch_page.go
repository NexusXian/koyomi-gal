package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

const (
	fetchTimeout      = 8 * time.Second
	maxFetchBodyBytes = 2 << 20 // 2 MB
	maxExtractedRunes = 30000
	maxURLQueryLength = 512
)

type FetchPageTool struct {
	client  *http.Client
	guarded bool // when false, DNS/IP policy checks are skipped (tests only)
	cache   *Cache
}

type FetchPageToolInput struct {
	URL string `json:"url"`
}

type PageContent struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func NewFetchPageTool(cache *Cache) *FetchPageTool {
	return &FetchPageTool{client: newGuardedHTTPClient(0), guarded: true, cache: cache}
}

// NewFetchPageToolUnsafeForTests builds a fetcher without SSRF guards so tests
// can exercise extraction against loopback servers. Never use in production.
func NewFetchPageToolUnsafeForTests(cache *Cache) *FetchPageTool {
	return &FetchPageTool{
		client:  &http.Client{Timeout: fetchTimeout},
		guarded: false,
		cache:   cache,
	}
}

func (t *FetchPageTool) Fetch(ctx context.Context, input FetchPageToolInput) (string, error) {
	target, err := validateFetchURL(input.URL)
	if err != nil {
		return "", err
	}
	canonical := canonicalURL(target)
	if canonical == "" {
		return "", fmt.Errorf("invalid url")
	}
	hash := sha256.Sum256([]byte(canonical))
	cacheKey := pageCacheKeyPrefix + hex.EncodeToString(hash[:])
	var cached PageContent
	if t.cache.getJSON(ctx, cacheKey, &cached) {
		return marshalPageContent(cached), nil
	}

	if t.guarded {
		if err := resolveAndCheck(ctx, target.Hostname()); err != nil {
			return "", err
		}
	}

	fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, target.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create fetch request: %w", err)
	}
	req.Header.Set("User-Agent", "Koyomi-GalClassificationBot/1.0 (+https://koyomi.example.com; age-rating research)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.8")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("page returned HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("page returned HTTP %d", resp.StatusCode)
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "text/plain") &&
		!strings.Contains(contentType, "application/xhtml") {
		return "", fmt.Errorf("unsupported content type %q", contentType)
	}

	reader := io.LimitReader(resp.Body, maxFetchBodyBytes+1)
	raw, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	if len(raw) > maxFetchBodyBytes {
		return "", fmt.Errorf("page body exceeds 2 MB")
	}

	page := PageContent{}
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml") {
		page = extractHTMLPage(raw)
	} else {
		page.Content = extractPlainText(raw)
	}
	page.Content = truncateRunes(strings.TrimSpace(page.Content), maxExtractedRunes)
	page.Title = truncateRunes(strings.TrimSpace(page.Title), 512)

	t.cache.setJSON(ctx, cacheKey, page, pageCacheTTL)
	return marshalPageContent(page), nil
}

func canonicalURL(parsed *url.URL) string {
	query := parsed.RawQuery
	if len(query) > maxURLQueryLength {
		query = query[:maxURLQueryLength]
	}
	return parsed.Scheme + "://" + strings.ToLower(parsed.Host) + parsed.EscapedPath() + "?" + query
}

func marshalPageContent(page PageContent) string {
	payload, err := json.Marshal(page)
	if err != nil {
		return `{"title":"","content":""}`
	}
	return string(payload)
}

func extractHTMLPage(raw []byte) PageContent {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(string(raw)))
	if err != nil {
		return PageContent{Title: "", Content: extractPlainText(raw)}
	}
	document.Find("script,style,noscript,iframe,form,button,svg,nav,footer,aside").Remove()
	title := strings.TrimSpace(document.Find("title").First().Text())
	if title == "" {
		title = strings.TrimSpace(document.Find("h1").First().Text())
	}

	main := document.Find("main,article").First()
	if main.Length() == 0 {
		main = document.Find("body")
	}
	content := normalizeWhitespace(main.Text())
	if len([]rune(content)) > 40000 {
		content = truncateRunes(content, 40000)
	}
	return PageContent{Title: title, Content: content}
}

func extractPlainText(raw []byte) string {
	return normalizeWhitespace(string(raw))
}

// normalizeWhitespace preserves line breaks while collapsing runs of spaces,
// so CJK keywords such as "18歳以上対象" stay intact.
func normalizeWhitespace(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	spacePending := false
	linePending := false
	for _, r := range value {
		switch {
		case r == '\n' || r == '\r':
			linePending = true
			spacePending = false
		case unicode.IsSpace(r):
			if !linePending {
				spacePending = true
			}
		default:
			if linePending {
				if builder.Len() > 0 {
					builder.WriteByte('\n')
				}
				linePending = false
				spacePending = false
			}
			if spacePending && builder.Len() > 0 {
				builder.WriteByte(' ')
				spacePending = false
			}
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}
