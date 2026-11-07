// Package cfworkers publishes a folder of static files as a Cloudflare
// Worker with static assets (<name>.<account>.workers.dev), in pure Go (no
// Node, no wrangler). Cloudflare recommends Workers over Pages for new
// projects: https://developers.cloudflare.com/workers/static-assets/migration-guides/migrate-from-pages/
//
// The protocol is the Direct Upload API
// (https://developers.cloudflare.com/workers/static-assets/direct-upload/),
// done the way wrangler does it (github.com/cloudflare/workers-sdk,
// packages/deploy-helpers/src/deploy/helpers/assets.ts and
// create-worker-upload-form.ts):
//
//  1. POST /accounts/{a}/workers/scripts/{w}/assets-upload-session with the
//     manifest {"/path": {hash, size}}: an upload JWT and the buckets of
//     hashes Cloudflare does not have yet;
//  2. POST /accounts/{a}/workers/assets/upload?base64=true (Bearer upload
//     JWT), one multipart body per bucket; the last answer carries the
//     completion JWT;
//  3. PUT  /accounts/{a}/workers/scripts/{w}, multipart with only the
//     metadata: {assets: {jwt, config}, compatibility_date}. A site of
//     static files needs no Worker code (wrangler's "assets only" upload);
//     _headers and _redirects go inside assets.config;
//  4. POST /accounts/{a}/workers/scripts/{w}/subdomain {enabled: true}
//     turns on <w>.<account subdomain>.workers.dev.
//
// Token permission: "Account > Workers Scripts > Edit".
package cfworkers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the Cloudflare API v4 root.
const DefaultBaseURL = "https://api.cloudflare.com/client/v4"

// TokenTemplateURL pre-fills the dashboard's "Create API token" form with
// "Account > Workers Scripts > Edit" (short key "workers_scripts", see
// https://developers.cloudflare.com/fundamentals/api/how-to/account-owned-token-template/).
const TokenTemplateURL = "https://dash.cloudflare.com/profile/api-tokens?permissionGroupKeys=%5B%7B%22key%22%3A%22workers_scripts%22%2C%22type%22%3A%22edit%22%7D%5D&accountId=%2A&zoneId=all&name=blup"

// Sentinel errors. Every error returned by this package that falls in one of
// these categories wraps the sentinel, so callers can use errors.Is.
var (
	// ErrInvalidToken: the API token is malformed, revoked or expired.
	ErrInvalidToken = errors.New("cloudflare: invalid API token")
	// ErrMissingPermission: the token is valid but lacks
	// "Account > Workers Scripts > Edit" (or access to the account).
	ErrMissingPermission = errors.New("cloudflare: API token is missing the Workers Scripts Edit permission")
	// ErrNameTaken: the account already has a Worker with this name that
	// Blup did not publish (Worker names are per account; publishing would
	// replace it).
	ErrNameTaken = errors.New("cloudflare: a Worker with this name already exists")
	// ErrInvalidName: the name does not follow the Worker name rules.
	ErrInvalidName = errors.New("cloudflare: invalid Worker name")
	// ErrNoSubdomain: the account has no workers.dev subdomain yet
	// (RegisterSubdomain creates one).
	ErrNoSubdomain = errors.New("cloudflare: the account has no workers.dev subdomain")
	// ErrSubdomainTaken: that workers.dev subdomain belongs to someone else.
	ErrSubdomainTaken = errors.New("cloudflare: workers.dev subdomain is taken")
	// ErrQuota: too many files or an account limit (Workers, rate limit).
	ErrQuota = errors.New("cloudflare: quota exceeded")
	// ErrTooManyFiles is a specific ErrQuota (errors.Is(err, ErrQuota) holds).
	ErrTooManyFiles = fmt.Errorf("%w: too many files for one version", ErrQuota)
	// ErrFileTooLarge: one file is bigger than MaxFileSize.
	ErrFileTooLarge = errors.New("cloudflare: file too large")
	// ErrNotFound: the requested resource (Worker, account) does not exist.
	ErrNotFound = errors.New("cloudflare: not found")
)

// APIError is a failed Cloudflare API call; Unwrap gives the sentinel.
type APIError struct {
	Method, Path string
	Status       int
	Codes        []int
	Messages     []string
	kind         error
}

func (e *APIError) Error() string {
	msg := strings.Join(e.Messages, "; ")
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	return fmt.Sprintf("cloudflare API %s %s: HTTP %d: %s (codes %v)", e.Method, e.Path, e.Status, msg, e.Codes)
}

func (e *APIError) Unwrap() error { return e.kind }

func (e *APIError) hasCode(c int) bool {
	for _, x := range e.Codes {
		if x == c {
			return true
		}
	}
	return false
}

// Cloudflare / Workers error codes we react to.
const (
	codeAuth              = 10000 // authentication error (missing permission)
	codeNotFound          = 10007 // Worker or workers.dev subdomain not found
	codeSubdomainTaken    = 10031 // workers.dev subdomain unavailable
	codeSubdomainUnusable = 10032 // subdomain check: it is free to register
)

func classify(e *APIError) error {
	text := strings.ToLower(strings.Join(e.Messages, " "))
	switch {
	case e.hasCode(codeSubdomainTaken):
		return ErrSubdomainTaken
	case e.Status == http.StatusNotFound || e.hasCode(codeNotFound) || e.hasCode(7003):
		return ErrNotFound
	case e.hasCode(1000) || e.hasCode(1001) || e.hasCode(6003) || e.hasCode(6111) ||
		e.hasCode(9106) || e.hasCode(9107) || e.hasCode(9103) ||
		strings.Contains(text, "invalid api token") ||
		e.Status == http.StatusUnauthorized:
		return ErrInvalidToken
	case e.Status == http.StatusForbidden || e.hasCode(codeAuth) || e.hasCode(9109):
		return ErrMissingPermission
	case strings.Contains(text, "too many") && (strings.Contains(text, "file") || strings.Contains(text, "asset")):
		return ErrTooManyFiles
	case e.Status == http.StatusRequestEntityTooLarge || strings.Contains(text, "too large"):
		return ErrFileTooLarge
	case e.Status == http.StatusTooManyRequests || strings.Contains(text, "limit"):
		return ErrQuota
	}
	return nil
}

// Client talks to the Cloudflare API with a user-supplied API token.
type Client struct {
	// Token is the API token ("Account > Workers Scripts > Edit").
	Token string
	// HTTP is the HTTP client; nil means a client with a 5 minute timeout.
	HTTP *http.Client
	// BaseURL overrides DefaultBaseURL (tests).
	BaseURL string

	// sleep is used for backoff; tests replace it to run instantly.
	sleep func(context.Context, time.Duration) error
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 5 * time.Minute}
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return DefaultBaseURL
}

func (c *Client) wait(ctx context.Context, d time.Duration) error {
	if c.sleep != nil {
		return c.sleep(ctx, d)
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

type envelope struct {
	Success bool            `json:"success"`
	Errors  []apiMessage    `json:"errors"`
	Result  json.RawMessage `json:"result"`
	Info    *struct {
		Page       int `json:"page"`
		TotalPages int `json:"total_pages"`
	} `json:"result_info"`
}

type apiMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// do performs a request. bearer overrides the API token (upload JWT).
// body may be nil, an io.Reader (with contentType) or a value to JSON-encode.
// headers are extra request headers, as name/value pairs.
func (c *Client) do(ctx context.Context, method, path, bearer string, body any, contentType string, out any, headers ...[2]string) (*envelope, error) {
	var rdr io.Reader
	switch b := body.(type) {
	case nil:
	case io.Reader:
		rdr = b
	default:
		buf, err := json.Marshal(b)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(buf)
		contentType = "application/json"
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL()+path, rdr)
	if err != nil {
		return nil, err
	}
	if bearer == "" {
		bearer = c.Token
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearer))
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for _, h := range headers {
		req.Header.Set(h[0], h[1])
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudflare API %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("cloudflare API %s %s: reading response: %w", method, path, err)
	}
	var env envelope
	jerr := json.Unmarshal(raw, &env)
	if resp.StatusCode >= 300 || jerr != nil || !env.Success {
		ae := &APIError{Method: method, Path: path, Status: resp.StatusCode}
		for _, m := range env.Errors {
			ae.Codes = append(ae.Codes, m.Code)
			ae.Messages = append(ae.Messages, m.Message)
		}
		if jerr != nil && len(ae.Messages) == 0 {
			snippet := strings.TrimSpace(string(raw))
			if len(snippet) > 200 {
				snippet = snippet[:200]
			}
			ae.Messages = append(ae.Messages, snippet)
		}
		ae.kind = classify(ae)
		return &env, ae
	}
	if out != nil && len(env.Result) > 0 && string(env.Result) != "null" {
		if err := json.Unmarshal(env.Result, out); err != nil {
			return &env, fmt.Errorf("cloudflare API %s %s: decoding result: %w", method, path, err)
		}
	}
	return &env, nil
}

// VerifyToken checks that the token is valid and active. It accepts both
// user tokens (/user/tokens/verify) and account-owned tokens
// (/accounts/{id}/tokens/verify). It does not prove the Workers
// permission; the first call to the account reports ErrMissingPermission.
func (c *Client) VerifyToken(ctx context.Context) error {
	if strings.TrimSpace(c.Token) == "" {
		return fmt.Errorf("empty token: %w", ErrInvalidToken)
	}
	var st struct {
		Status string `json:"status"`
	}
	_, err := c.do(ctx, http.MethodGet, "/user/tokens/verify", "", nil, "", &st)
	if err == nil {
		return checkStatus(st.Status)
	}
	userErr := err
	// Account-owned tokens fail on /user/tokens/verify; try per account.
	accts, aerr := c.Accounts(ctx)
	if aerr != nil || len(accts) == 0 {
		return wrapVerify(userErr)
	}
	for _, a := range accts {
		st.Status = ""
		if _, err := c.do(ctx, http.MethodGet, "/accounts/"+a.ID+"/tokens/verify", "", nil, "", &st); err == nil {
			return checkStatus(st.Status)
		}
	}
	return wrapVerify(userErr)
}

func checkStatus(s string) error {
	if s != "" && s != "active" {
		return fmt.Errorf("token status is %q: %w", s, ErrInvalidToken)
	}
	return nil
}

func wrapVerify(err error) error {
	if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrMissingPermission) {
		return fmt.Errorf("verifying token: %w", err)
	}
	return fmt.Errorf("verifying token: %w: %w", ErrInvalidToken, err)
}

// Account is a Cloudflare account the token can access.
type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Accounts lists the accounts the token can access.
func (c *Client) Accounts(ctx context.Context) ([]Account, error) {
	var all []Account
	for page := 1; page <= 100; page++ {
		var batch []Account
		env, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/accounts?page=%d&per_page=50", page), "", nil, "", &batch)
		if err != nil {
			return nil, fmt.Errorf("listing accounts: %w", err)
		}
		all = append(all, batch...)
		if env.Info == nil || env.Info.TotalPages <= page || len(batch) == 0 {
			break
		}
	}
	return all, nil
}
