package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SearchResult is one web search hit.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// SearchProvider is the search backend abstraction; concrete implementations
// are interchangeable (Tavily today, Brave/Serper/SearXNG later).
type SearchProvider interface {
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
}

// SearchTool gives the agent web search. Search results are cached in Redis
// for 24h so repeated classification of the same game is cheap.
type SearchTool struct {
	provider SearchProvider
	cache    *Cache
}

type SearchToolInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

func NewSearchTool(provider SearchProvider, cache *Cache) *SearchTool {
	return &SearchTool{provider: provider, cache: cache}
}

func (t *SearchTool) Search(ctx context.Context, input SearchToolInput) (string, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return "", errors.New("query is required")
	}
	if len(query) > 300 {
		return "", errors.New("query is too long (max 300 characters)")
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 5 {
		limit = 5
	}

	hash := sha256.Sum256([]byte(query))
	cacheKey := searchCacheKeyPrefix + hex.EncodeToString(hash[:]) + fmt.Sprintf(":%d", limit)
	var cached []SearchResult
	if t.cache.getJSON(ctx, cacheKey, &cached) {
		return marshalSearchResults(cached), nil
	}

	if t.provider == nil {
		return "", errors.New("web search is not configured on this server")
	}
	results, err := t.provider.Search(ctx, query, limit)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "{\"results\":[]}", nil
	}
	t.cache.setJSON(ctx, cacheKey, results, searchCacheTTL)
	return marshalSearchResults(results), nil
}

func marshalSearchResults(results []SearchResult) string {
	type searchHit struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet"`
	}
	hits := make([]searchHit, 0, len(results))
	for _, result := range results {
		hits = append(hits, searchHit{Title: result.Title, URL: result.URL, Snippet: result.Snippet})
	}
	payload, err := json.Marshal(map[string]any{"results": hits})
	if err != nil {
		return `{"results":[]}`
	}
	return string(payload)
}

const tavilyEndpoint = "https://api.tavily.com/search"

// TavilyProvider implements SearchProvider against the Tavily API.
type TavilyProvider struct {
	apiKey string
	client *http.Client
}

func NewTavilyProvider(apiKey string) *TavilyProvider {
	return &TavilyProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 12 * time.Second},
	}
}

func (p *TavilyProvider) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return nil, errors.New("tavily api key is not configured")
	}
	body, err := json.Marshal(map[string]any{
		"api_key":      p.apiKey,
		"query":        query,
		"max_results":  limit,
		"search_depth": "basic",
	})
	if err != nil {
		return nil, fmt.Errorf("encode tavily request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tavilyEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create tavily request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request tavily: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("tavily returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}

	var payload struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode tavily response: %w", err)
	}
	results := make([]SearchResult, 0, len(payload.Results))
	for _, item := range payload.Results {
		results = append(results, SearchResult{
			Title:   strings.TrimSpace(item.Title),
			URL:     strings.TrimSpace(item.URL),
			Snippet: strings.TrimSpace(item.Content),
		})
	}
	return results, nil
}
