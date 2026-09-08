// Package github provides a minimal client for the GitHub Releases REST API.
// It only supports the read operations needed to resolve official app package
// download URLs (browser_download_url) for published releases.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	ErrReleaseNotFound = errors.New("github release not found")
	ErrUnauthorized    = errors.New("github api unauthorized")
	ErrRateLimited     = errors.New("github api rate limited")
)

const (
	defaultPerPage = 100
	maxPerPage     = 100
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	repo       string
	token      string
}

func NewClient(baseURL, repo, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &Client{httpClient: httpClient, baseURL: baseURL, repo: repo, token: token}
}

type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

type Asset struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	Digest      string    `json:"digest"`
	DownloadURL string    `json:"browser_download_url"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FindAPKAsset returns the first .apk asset of the release.
func (r *Release) FindAPKAsset() *Asset {
	for i := range r.Assets {
		if strings.HasSuffix(strings.ToLower(r.Assets[i].Name), ".apk") {
			return &r.Assets[i]
		}
	}
	return nil
}

// ListReleases returns published (non-draft) releases, newest first. Draft
// releases are excluded because their asset URLs are untagged and require
// authentication, which makes them unusable as public download URLs.
func (c *Client) ListReleases(ctx context.Context, page, perPage int) ([]Release, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > maxPerPage {
		perPage = defaultPerPage
	}
	var releases []Release
	err := c.get(ctx, fmt.Sprintf("/repos/%s/releases?per_page=%s&page=%d",
		c.repo, strconv.Itoa(perPage), page), &releases)
	if err != nil {
		return nil, err
	}
	published := make([]Release, 0, len(releases))
	for _, release := range releases {
		if !release.Draft {
			published = append(published, release)
		}
	}
	return published, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build github request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call github api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusOK:
	case resp.StatusCode == http.StatusNotFound:
		return ErrReleaseNotFound
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: status %d", ErrUnauthorized, resp.StatusCode)
	case resp.StatusCode == http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("read github response: %w", err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode github response: %w", err)
	}
	return nil
}

// ParseAssetSHA256 extracts the hex SHA-256 from a release asset digest
// ("sha256:<hex>"). It returns false when the digest is absent or not SHA-256.
func ParseAssetSHA256(digest string) (string, bool) {
	value, ok := strings.CutPrefix(digest, "sha256:")
	if !ok {
		return "", false
	}
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != 64 {
		return "", false
	}
	for _, r := range value {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return "", false
		}
	}
	return value, true
}
