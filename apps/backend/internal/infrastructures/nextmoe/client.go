// Package nextmoe is a minimal read-only client for the NextMoe open catalog
// API v2. It resolves catalog works by their exact VNDB reference and reads
// their intros. The API key is only ever sent as a bearer header and never
// appears in errors.
package nextmoe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	DefaultBaseURL = "https://api.nextmoe.dev/v2"

	maxBatchSize         = 100
	maxIntroPages        = 10
	maxResponseBytes     = 8 << 20
	maxErrorBytes        = 64 << 10
	defaultPerMinute     = 45
	defaultMaxRetries    = 3
	maxRetryAfter        = 60 * time.Second
	defaultQuotaPause    = 15 * time.Minute
	maxQuotaPause        = 24 * time.Hour
	baseBackoff          = 500 * time.Millisecond
	defaultClientTimeout = 15 * time.Second
)

var (
	ErrNotConfigured = errors.New("nextmoe client is not configured")
	ErrInvalidID     = errors.New("nextmoe: invalid id")
	ErrUnauthorized  = errors.New("nextmoe: unauthorized")
	ErrRateLimited   = errors.New("nextmoe: rate limited")
	ErrQuotaExceeded = errors.New("nextmoe: quota exceeded")
	ErrNotFound      = errors.New("nextmoe: not found")
	ErrAmbiguous     = errors.New("nextmoe: ambiguous vndb match")
	ErrNoIntro       = errors.New("nextmoe: no usable chinese intro")
	ErrServer        = errors.New("nextmoe: server error")
	ErrBadResponse   = errors.New("nextmoe: bad response")
	ErrRequest       = errors.New("nextmoe: request rejected")
)

var (
	vndbIDPattern = regexp.MustCompile(`^v[1-9][0-9]*$`)
	workIDPattern = regexp.MustCompile(`^[0-9]{1,20}$`)
)

// APIError describes a non-2xx response. It unwraps to one of the sentinel
// errors so callers can use errors.Is.
type APIError struct {
	Status     int
	Code       string
	RequestID  string
	CurrentID  string
	RetryAfter time.Duration
	kind       error
}

func (e *APIError) Error() string {
	message := fmt.Sprintf("%v: status %d", e.kind, e.Status)
	if e.Code != "" {
		message += " code " + e.Code
	}
	if e.RequestID != "" {
		message += " request_id " + e.RequestID
	}
	return message
}

func (e *APIError) Unwrap() error { return e.kind }

type Ref struct {
	Source     string `json:"source"`
	ExternalID string `json:"external_id"`
}

type Intro struct {
	Lang      string `json:"lang"`
	Value     string `json:"value"`
	IsMachine bool   `json:"is_machine"`
	Source    string `json:"source"`
}

// Work is the subset of a catalog work this client reads. Intros is nil when
// the response omitted the block, as opposed to an empty list.
type Work struct {
	ID     string   `json:"id"`
	Refs   []Ref    `json:"refs"`
	Intros *[]Intro `json:"intros"`
}

// Match is the lookup outcome for one VNDB ID. Err is ErrNotFound,
// ErrAmbiguous or ErrInvalidID when Work is nil.
type Match struct {
	Work *Work
	Err  error
}

// ChineseIntro is the selected Simplified Chinese intro of a work.
type ChineseIntro struct {
	WorkID  string
	Lang    string
	Content string
	Source  string
}

type workList struct {
	Items      []Work   `json:"items"`
	Missing    []string `json:"missing"`
	NextCursor string   `json:"next_cursor"`
}

type introList struct {
	Items      []Intro `json:"items"`
	NextCursor string  `json:"next_cursor"`
}

type problem struct {
	Code      string `json:"code"`
	RequestID string `json:"request_id"`
	CurrentID string `json:"current_id"`
}

type Options struct {
	HTTPClient        *http.Client
	RequestsPerMinute int
	MaxRetries        int
}

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	limiter    *rate.Limiter
	maxRetries int
	sleep      func(ctx context.Context, d time.Duration) error

	mu          sync.Mutex
	pausedUntil time.Time
	quotaUntil  time.Time
}

func NewClient(baseURL, apiKey string, opts Options) *Client {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultClientTimeout}
	}
	// Redirects could forward the bearer header to another host.
	client := *httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	perMinute := opts.RequestsPerMinute
	if perMinute <= 0 {
		perMinute = defaultPerMinute
	}
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		httpClient: &client,
		baseURL:    baseURL,
		apiKey:     strings.TrimSpace(apiKey),
		limiter:    rate.NewLimiter(rate.Every(time.Minute/time.Duration(perMinute)), 1),
		maxRetries: maxRetries,
		sleep:      sleepContext,
	}
}

// ValidVNDBID reports whether id is a canonical VNDB visual novel ID.
func ValidVNDBID(id string) bool { return vndbIDPattern.MatchString(id) }

// FindWorksByVNDBIDs resolves VNDB IDs to catalog works through the refs
// batch lane. Every requested ID gets an entry; a work only matches when it
// carries a vndb ref whose external_id equals the ID exactly.
func (c *Client) FindWorksByVNDBIDs(
	ctx context.Context,
	vndbIDs []string,
	includeIntros bool,
) (map[string]Match, error) {
	if c == nil || c.apiKey == "" {
		return nil, ErrNotConfigured
	}
	results := make(map[string]Match, len(vndbIDs))
	pending := make([]string, 0, len(vndbIDs))
	for _, id := range vndbIDs {
		if _, seen := results[id]; seen {
			continue
		}
		if !ValidVNDBID(id) {
			results[id] = Match{Err: ErrInvalidID}
			continue
		}
		results[id] = Match{Err: ErrNotFound}
		pending = append(pending, id)
	}
	include := "refs"
	if includeIntros {
		include = "refs,intros"
	}
	for start := 0; start < len(pending); start += maxBatchSize {
		chunk := pending[start:min(start+maxBatchSize, len(pending))]
		refs := make([]string, len(chunk))
		for i, id := range chunk {
			refs[i] = "vndb:" + id
		}
		query := url.Values{}
		query.Set("refs", strings.Join(refs, ","))
		query.Set("include", include)
		query.Set("nsfw", "true")
		var list workList
		if err := c.getJSON(ctx, "/catalog/works", query, &list); err != nil {
			return nil, err
		}
		applyMatches(results, chunk, list.Items)
	}
	return results, nil
}

func applyMatches(results map[string]Match, chunk []string, works []Work) {
	wanted := make(map[string]struct{}, len(chunk))
	for _, id := range chunk {
		wanted[id] = struct{}{}
	}
	found := map[string]*Work{}
	for i := range works {
		work := &works[i]
		if !workIDPattern.MatchString(work.ID) {
			continue
		}
		for _, ref := range work.Refs {
			if ref.Source != "vndb" {
				continue
			}
			if _, ok := wanted[ref.ExternalID]; !ok {
				continue
			}
			previous, exists := found[ref.ExternalID]
			switch {
			case !exists:
				found[ref.ExternalID] = work
				results[ref.ExternalID] = Match{Work: work}
			case previous.ID != work.ID:
				results[ref.ExternalID] = Match{Err: ErrAmbiguous}
			}
		}
	}
}

// FindWorkByVNDBID resolves a single VNDB ID.
func (c *Client) FindWorkByVNDBID(ctx context.Context, vndbID string, includeIntros bool) (*Work, error) {
	matches, err := c.FindWorksByVNDBIDs(ctx, []string{vndbID}, includeIntros)
	if err != nil {
		return nil, err
	}
	match := matches[vndbID]
	if match.Work == nil {
		return nil, match.Err
	}
	return match.Work, nil
}

// ListIntros returns every intro of a work, following cursors and one merge
// redirect (ENTITY_MERGED with current_id).
func (c *Client) ListIntros(ctx context.Context, workID string) ([]Intro, error) {
	if c == nil || c.apiKey == "" {
		return nil, ErrNotConfigured
	}
	intros, err := c.listIntros(ctx, workID)
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Code == "ENTITY_MERGED" &&
		apiErr.CurrentID != "" && apiErr.CurrentID != workID {
		return c.listIntros(ctx, apiErr.CurrentID)
	}
	return intros, err
}

func (c *Client) listIntros(ctx context.Context, workID string) ([]Intro, error) {
	if !workIDPattern.MatchString(workID) {
		return nil, ErrInvalidID
	}
	intros := []Intro{}
	cursor := ""
	seen := map[string]struct{}{}
	for page := 0; page < maxIntroPages; page++ {
		query := url.Values{}
		query.Set("nsfw", "true")
		query.Set("limit", strconv.Itoa(maxBatchSize))
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		var list introList
		if err := c.getJSON(ctx, "/catalog/works/"+workID+"/intros", query, &list); err != nil {
			return nil, err
		}
		intros = append(intros, list.Items...)
		if list.NextCursor == "" || !strings.HasPrefix(list.NextCursor, "cur_") {
			break
		}
		if _, repeated := seen[list.NextCursor]; repeated {
			break
		}
		seen[list.NextCursor] = struct{}{}
		cursor = list.NextCursor
	}
	return intros, nil
}

// ChineseIntroForWork selects the Chinese intro of an already resolved work,
// using embedded intros when present and fetching them otherwise.
func (c *Client) ChineseIntroForWork(ctx context.Context, work *Work) (*ChineseIntro, error) {
	if work == nil {
		return nil, ErrNotFound
	}
	var intros []Intro
	if work.Intros != nil {
		intros = *work.Intros
	} else {
		fetched, err := c.ListIntros(ctx, work.ID)
		if err != nil {
			return nil, err
		}
		intros = fetched
	}
	selected, ok := SelectChineseIntro(intros)
	if !ok {
		return nil, ErrNoIntro
	}
	return &ChineseIntro{
		WorkID:  work.ID,
		Lang:    selected.Lang,
		Content: selected.Value,
		Source:  selected.Source,
	}, nil
}

// GetChineseIntro resolves a VNDB ID and returns its Chinese intro.
func (c *Client) GetChineseIntro(ctx context.Context, vndbID string) (*ChineseIntro, error) {
	work, err := c.FindWorkByVNDBID(ctx, vndbID, true)
	if err != nil {
		return nil, err
	}
	return c.ChineseIntroForWork(ctx, work)
}

// SelectChineseIntro picks the best human-written Simplified Chinese intro.
// zh-Hans ranks first, then regional Simplified tags; Traditional and the
// script-less "zh" are rejected because their script cannot be trusted.
// Ties break on source then value so the choice is stable.
func SelectChineseIntro(intros []Intro) (Intro, bool) {
	best := -1
	bestTier := 0
	for i, intro := range intros {
		if intro.IsMachine || strings.TrimSpace(intro.Value) == "" {
			continue
		}
		tier, ok := simplifiedChineseTier(intro.Lang)
		if !ok {
			continue
		}
		if best < 0 || tier < bestTier ||
			(tier == bestTier && lessIntro(intro, intros[best])) {
			best, bestTier = i, tier
		}
	}
	if best < 0 {
		return Intro{}, false
	}
	return intros[best], true
}

func lessIntro(a, b Intro) bool {
	if a.Source != b.Source {
		return a.Source < b.Source
	}
	return a.Value < b.Value
}

func simplifiedChineseTier(lang string) (int, bool) {
	tag := strings.ToLower(strings.TrimSpace(lang))
	switch {
	case tag == "zh-hans":
		return 0, true
	case strings.HasPrefix(tag, "zh-hans-"), tag == "zh-cn", tag == "zh-sg", tag == "zh-my":
		return 1, true
	default:
		return 0, false
	}
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	for attempt := 0; ; attempt++ {
		if err := c.waitTurn(ctx); err != nil {
			return err
		}
		err := c.doOnce(ctx, path, query, out)
		if err == nil {
			return nil
		}
		delay, retryable := c.retryDelay(err, attempt)
		if !retryable || attempt >= c.maxRetries || ctx.Err() != nil {
			return err
		}
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < delay {
			return err
		}
		if sleepErr := c.sleep(ctx, delay); sleepErr != nil {
			return err
		}
	}
}

// waitTurn enforces the shared rate limit and any server-imposed pause. An
// exhausted quota fails fast instead of waiting.
func (c *Client) waitTurn(ctx context.Context) error {
	c.mu.Lock()
	quotaUntil, pausedUntil := c.quotaUntil, c.pausedUntil
	c.mu.Unlock()
	now := time.Now()
	if now.Before(quotaUntil) {
		return &APIError{Status: http.StatusTooManyRequests, Code: "QUOTA_EXCEEDED", kind: ErrQuotaExceeded}
	}
	if wait := pausedUntil.Sub(now); wait > 0 {
		if err := c.sleep(ctx, wait); err != nil {
			return err
		}
	}
	return c.limiter.Wait(ctx)
}

func (c *Client) retryDelay(err error, attempt int) (time.Duration, bool) {
	var apiErr *APIError
	switch {
	case errors.Is(err, ErrRateLimited):
		delay := time.Second
		if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
			delay = apiErr.RetryAfter
		}
		return delay + jitter(delay), true
	case errors.Is(err, ErrServer), errors.Is(err, errTransport):
		delay := baseBackoff << attempt
		return delay + jitter(delay), true
	default:
		return 0, false
	}
}

var errTransport = errors.New("nextmoe: transport error")

func (c *Client) doOnce(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL + path
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build nextmoe request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%w: %s", errTransport, transportReason(err))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return c.apiError(resp)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("%w: read body", errTransport)
	}
	if len(body) > maxResponseBytes {
		return fmt.Errorf("%w: response too large", ErrBadResponse)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%w: decode json", ErrBadResponse)
	}
	return nil
}

func (c *Client) apiError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
	var details problem
	_ = json.Unmarshal(body, &details)
	apiErr := &APIError{
		Status:     resp.StatusCode,
		Code:       sanitizeToken(details.Code, 63),
		RequestID:  sanitizeToken(firstNonEmpty(details.RequestID, resp.Header.Get("X-Request-ID")), 30),
		CurrentID:  details.CurrentID,
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
	}
	if !workIDPattern.MatchString(apiErr.CurrentID) {
		apiErr.CurrentID = ""
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		apiErr.kind = ErrUnauthorized
	case resp.StatusCode == http.StatusNotFound:
		apiErr.kind = ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests && apiErr.Code == "QUOTA_EXCEEDED":
		apiErr.kind = ErrQuotaExceeded
		pause := apiErr.RetryAfter
		if pause <= 0 {
			pause = defaultQuotaPause
		}
		c.setPause(&c.quotaUntil, min(pause, maxQuotaPause))
	case resp.StatusCode == http.StatusTooManyRequests:
		apiErr.kind = ErrRateLimited
		if apiErr.RetryAfter <= 0 || apiErr.RetryAfter > maxRetryAfter {
			apiErr.RetryAfter = min(max(apiErr.RetryAfter, time.Second), maxRetryAfter)
		}
		c.setPause(&c.pausedUntil, apiErr.RetryAfter)
	case resp.StatusCode >= 500:
		apiErr.kind = ErrServer
	default:
		apiErr.kind = ErrRequest
	}
	return apiErr
}

func (c *Client) setPause(target *time.Time, d time.Duration) {
	until := time.Now().Add(d)
	c.mu.Lock()
	if until.After(*target) {
		*target = until
	}
	c.mu.Unlock()
}

func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil {
		if d := time.Until(at); d > 0 {
			return d
		}
	}
	return 0
}

// transportReason keeps only the error class: url.Error text would repeat
// the full request URL in logs.
func transportReason(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return "timeout"
		}
		return urlErr.Op + " failed"
	}
	return "request failed"
}

func sanitizeToken(value string, maxLen int) string {
	if len(value) > maxLen {
		value = value[:maxLen]
	}
	for _, r := range value {
		if !(r == '_' || r == '-' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			return ""
		}
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(d)/5 + 1))
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
