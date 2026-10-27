package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// APIVersion is the REST API version sent in X-GitHub-Api-Version.
const APIVersion = "2026-03-10"

// DefaultBaseURL is the REST API root.
const DefaultBaseURL = "https://api.github.com"

// Sentinel errors. Errors returned by this package wrap one of these when the
// cause is known, so callers can use errors.Is to show a friendly message.
var (
	// ErrBadToken: the token is missing, invalid, expired or revoked (HTTP 401).
	ErrBadToken = errors.New("github: invalid or expired token")
	// ErrPermission: the token lacks a permission or scope (HTTP 403, or 404
	// on a resource that should be visible).
	ErrPermission = errors.New("github: token lacks a required permission")
	// ErrRepoConflict: the repository name belongs to another account, or the
	// name is already taken / cannot be used.
	ErrRepoConflict = errors.New("github: repository name conflict")
	// ErrRateLimited: GitHub kept rate limiting after all retries.
	ErrRateLimited = errors.New("github: rate limited")
	// ErrPagesUnavailable: GitHub Pages cannot be enabled for this repository
	// (e.g. private repository on a free plan).
	ErrPagesUnavailable = errors.New("github: GitHub Pages unavailable for this repository")
	// ErrPagesBuildFailed: the latest Pages build errored.
	ErrPagesBuildFailed = errors.New("github: GitHub Pages build failed")
	// ErrNotFound: the resource does not exist (or is invisible to the token).
	ErrNotFound = errors.New("github: not found")
	// ErrFileTooLarge: a file exceeds GitHub's 100 MiB per-file limit.
	ErrFileTooLarge = errors.New("github: file larger than 100 MiB")
	// ErrNothingToPublish: the directory has no files.
	ErrNothingToPublish = errors.New("github: nothing to publish")
)

// APIError is a non-2xx response from GitHub.
type APIError struct {
	Method     string
	Path       string
	StatusCode int
	Message    string
	DocURL     string
	// Errors holds the "errors" array of validation responses, flattened.
	Errors []string
	kind   error
}

func (e *APIError) Error() string {
	msg := e.Message
	if len(e.Errors) > 0 {
		msg += " (" + strings.Join(e.Errors, "; ") + ")"
	}
	return fmt.Sprintf("github: %s %s: %d %s", e.Method, e.Path, e.StatusCode, msg)
}

// Unwrap returns the sentinel error classifying this failure, if any.
func (e *APIError) Unwrap() error { return e.kind }

// Client talks to the GitHub REST API with a token. Zero values of the
// exported fields fall back to sensible defaults; adjust them after [New].
type Client struct {
	Token      string
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string

	// Concurrency bounds parallel blob uploads (default 4).
	Concurrency int
	// MinWriteInterval spaces content-creating requests (POST/PATCH/PUT) to
	// stay under the ~80 writes/minute secondary limit. Default 750ms;
	// set a negative value to disable.
	MinWriteInterval time.Duration
	// MaxRetries for rate-limit and transient 5xx responses (default 5).
	MaxRetries int
	// MaxRetryWait caps a single wait; longer waits fail with ErrRateLimited
	// (default 15 minutes).
	MaxRetryWait time.Duration
	// InlineTextLimit: UTF-8 text files up to this size are sent inline in
	// the tree request instead of as separate blobs (default 512 KiB;
	// negative disables inlining).
	InlineTextLimit int64
	// TreeChunkEntries and TreeChunkBytes split large tree creations
	// (defaults 3000 entries, 16 MiB of inline content).
	TreeChunkEntries int
	TreeChunkBytes   int64

	// sleep is replaceable in tests.
	sleep func(ctx context.Context, d time.Duration) error
	now   func() time.Time

	writeMu   sync.Mutex
	lastWrite time.Time
}

// New returns a Client authenticated with token.
func New(token string) *Client {
	return &Client{Token: token}
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return DefaultBaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 5 * time.Minute}
}

func (c *Client) concurrency() int {
	if c.Concurrency > 0 {
		return c.Concurrency
	}
	return 4
}

func (c *Client) maxRetries() int {
	if c.MaxRetries > 0 {
		return c.MaxRetries
	}
	return 5
}

func (c *Client) maxRetryWait() time.Duration {
	if c.MaxRetryWait > 0 {
		return c.MaxRetryWait
	}
	return 15 * time.Minute
}

func (c *Client) writeInterval() time.Duration {
	switch {
	case c.MinWriteInterval < 0:
		return 0
	case c.MinWriteInterval == 0:
		return 750 * time.Millisecond
	}
	return c.MinWriteInterval
}

func (c *Client) nowTime() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func (c *Client) doSleep(ctx context.Context, d time.Duration) error {
	if c.sleep != nil {
		return c.sleep(ctx, d)
	}
	return sleepCtx(ctx, d)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// paceWrite blocks so consecutive writes are at least writeInterval apart.
func (c *Client) paceWrite(ctx context.Context) error {
	iv := c.writeInterval()
	if iv <= 0 {
		return nil
	}
	c.writeMu.Lock()
	now := c.nowTime()
	next := c.lastWrite.Add(iv)
	wait := time.Duration(0)
	if next.After(now) {
		wait = next.Sub(now)
		c.lastWrite = next
	} else {
		c.lastWrite = now
	}
	c.writeMu.Unlock()
	return c.doSleep(ctx, wait)
}

// do performs a JSON request with retries. in may be nil; out may be nil.
// It returns the HTTP status code of the final response.
func (c *Client) do(ctx context.Context, method, path string, in, out any) (int, error) {
	var body []byte
	if in != nil {
		var err error
		body, err = json.Marshal(in)
		if err != nil {
			return 0, err
		}
	}
	write := method != http.MethodGet && method != http.MethodHead
	for attempt := 0; ; attempt++ {
		if write {
			if err := c.paceWrite(ctx); err != nil {
				return 0, err
			}
		}
		var rd io.Reader
		if body != nil {
			rd = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL()+path, rd)
		if err != nil {
			return 0, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", APIVersion)
		ua := c.UserAgent
		if ua == "" {
			ua = "blup"
		}
		req.Header.Set("User-Agent", ua)
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.httpClient().Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return 0, ctx.Err()
			}
			// Network error: retry a few times.
			if attempt >= c.maxRetries() {
				return 0, fmt.Errorf("github: %s %s: %w", method, path, err)
			}
			if err := c.doSleep(ctx, backoff(attempt, time.Second)); err != nil {
				return 0, err
			}
			continue
		}
		data, rerr := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		resp.Body.Close()
		if rerr != nil {
			if attempt >= c.maxRetries() {
				return 0, fmt.Errorf("github: %s %s: %w", method, path, rerr)
			}
			if err := c.doSleep(ctx, backoff(attempt, time.Second)); err != nil {
				return 0, err
			}
			continue
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out != nil && len(data) > 0 {
				if err := json.Unmarshal(data, out); err != nil {
					return resp.StatusCode, fmt.Errorf("github: %s %s: decoding response: %w", method, path, err)
				}
			}
			return resp.StatusCode, nil
		}
		apiErr := parseAPIError(method, path, resp, data)
		wait, retry := c.retryDelay(resp, apiErr, attempt)
		if !retry || attempt >= c.maxRetries() || wait > c.maxRetryWait() {
			return resp.StatusCode, apiErr
		}
		if err := c.doSleep(ctx, wait); err != nil {
			return 0, err
		}
	}
}

func backoff(attempt int, base time.Duration) time.Duration {
	d := base << attempt
	if d > 2*time.Minute {
		d = 2 * time.Minute
	}
	return d
}

// retryDelay decides whether and how long to wait before retrying.
func (c *Client) retryDelay(resp *http.Response, e *APIError, attempt int) (time.Duration, bool) {
	switch {
	case resp.StatusCode == http.StatusTooManyRequests,
		resp.StatusCode == http.StatusForbidden && errors.Is(e, ErrRateLimited):
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if s, err := strconv.Atoi(strings.TrimSpace(ra)); err == nil {
				return time.Duration(s) * time.Second, true
			}
		}
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			if reset, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
				d := time.Unix(reset, 0).Sub(c.nowTime()) + time.Second
				if d < time.Second {
					d = time.Second
				}
				return d, true
			}
		}
		// Secondary limit without hints: at least one minute, growing.
		return backoff(attempt, time.Minute), true
	case resp.StatusCode == http.StatusBadGateway,
		resp.StatusCode == http.StatusServiceUnavailable,
		resp.StatusCode == http.StatusGatewayTimeout,
		resp.StatusCode == http.StatusInternalServerError:
		return backoff(attempt, 2*time.Second), true
	}
	return 0, false
}

func parseAPIError(method, path string, resp *http.Response, data []byte) *APIError {
	var raw struct {
		Message          string            `json:"message"`
		DocumentationURL string            `json:"documentation_url"`
		Errors           []json.RawMessage `json:"errors"`
	}
	_ = json.Unmarshal(data, &raw)
	e := &APIError{
		Method:     method,
		Path:       path,
		StatusCode: resp.StatusCode,
		Message:    raw.Message,
		DocURL:     raw.DocumentationURL,
	}
	if e.Message == "" {
		e.Message = http.StatusText(resp.StatusCode)
	}
	for _, r := range raw.Errors {
		var s string
		if json.Unmarshal(r, &s) == nil {
			e.Errors = append(e.Errors, s)
			continue
		}
		var o struct {
			Message, Code, Field string
		}
		if json.Unmarshal(r, &o) == nil {
			switch {
			case o.Message != "":
				e.Errors = append(e.Errors, o.Message)
			case o.Code != "":
				e.Errors = append(e.Errors, strings.TrimSpace(o.Field+" "+o.Code))
			}
		}
	}
	lower := strings.ToLower(e.Message)
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		e.kind = ErrBadToken
	case http.StatusTooManyRequests:
		e.kind = ErrRateLimited
	case http.StatusForbidden:
		if resp.Header.Get("X-RateLimit-Remaining") == "0" ||
			resp.Header.Get("Retry-After") != "" ||
			strings.Contains(lower, "rate limit") {
			e.kind = ErrRateLimited
		} else {
			e.kind = ErrPermission
		}
	case http.StatusNotFound:
		e.kind = ErrNotFound
	}
	return e
}

// statusOf returns the HTTP status of an *APIError in err, or 0.
func statusOf(err error) int {
	var e *APIError
	if errors.As(err, &e) {
		return e.StatusCode
	}
	return 0
}
