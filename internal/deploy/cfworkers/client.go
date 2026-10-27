// Package cfworkers publishes a folder of static files as an assets-only
// Cloudflare Worker (Workers Static Assets), in pure Go (no Node, no wrangler).
//
// Workers Static Assets is Cloudflare's recommended platform for new static
// sites (the Pages docs say "Start new projects with Workers"). The protocol
// mirrors wrangler (github.com/cloudflare/workers-sdk,
// packages/deploy-helpers/src/deploy/helpers/assets.ts,
// create-worker-upload-form.ts, triggers/subdomain.ts) and
// https://developers.cloudflare.com/workers/static-assets/direct-upload/:
//
//  1. POST /accounts/{a}/workers/scripts/{s}/assets-upload-session
//     {manifest: {"/path": {hash, size}}} -> {jwt, buckets: [[hash...]...]}
//  2. POST /accounts/{a}/workers/assets/upload?base64=true  (Bearer upload JWT)
//     multipart, one part per file (name = filename = hash, body = base64,
//     part Content-Type = served Content-Type). The response carrying a jwt
//     is the completion token. With no buckets, the session jwt already is.
//  3. PUT  /accounts/{a}/workers/scripts/{s}  multipart with only "metadata":
//     {assets: {jwt, config: {html_handling, not_found_handling, _headers,
//     _redirects}}, compatibility_date}  (assets-only Worker, no code).
//  4. POST /accounts/{a}/workers/scripts/{s}/subdomain {enabled: true}
//     to turn on https://<script>.<account-subdomain>.workers.dev/.
//
// Token permission: "Account > Workers Scripts > Edit" covers every call
// above, including reading the account's workers.dev subdomain.
package cfworkers

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
	"time"
)

// DefaultBaseURL is the Cloudflare API v4 root.
const DefaultBaseURL = "https://api.cloudflare.com/client/v4"

// Sentinel errors; use errors.Is on anything this package returns.
var (
	// ErrInvalidToken: the API token is malformed, revoked or expired.
	ErrInvalidToken = errors.New("cloudflare: invalid API token")
	// ErrMissingPermission: the token is valid but lacks
	// "Account > Workers Scripts > Edit" (or access to the account).
	ErrMissingPermission = errors.New("cloudflare: API token is missing the Workers Scripts Edit permission")
	// ErrNoWorkersSubdomain: the account has never set up its workers.dev
	// subdomain. The user must open the Workers dashboard once
	// (WorkersOnboardingURL) and pick one; we do not create it.
	ErrNoWorkersSubdomain = errors.New("cloudflare: account has no workers.dev subdomain yet")
	// ErrInvalidScriptName: the name does not fit a workers.dev hostname label.
	ErrInvalidScriptName = errors.New("cloudflare: invalid worker name")
	// ErrQuota: too many files or an account limit / rate limit was hit.
	ErrQuota = errors.New("cloudflare: quota exceeded")
	// ErrTooManyFiles is a specific ErrQuota (errors.Is(err, ErrQuota) holds).
	ErrTooManyFiles = fmt.Errorf("%w: too many files for one deployment", ErrQuota)
	// ErrFileTooLarge: one file is bigger than MaxFileSize.
	ErrFileTooLarge = errors.New("cloudflare: file too large")
	// ErrNotFound: the resource (account, worker) does not exist.
	ErrNotFound = errors.New("cloudflare: not found")
	// ErrUploadExpired: the upload session expired before every file was
	// sent, twice in a row. Retrying resumes (uploaded files are kept).
	ErrUploadExpired = errors.New("cloudflare: upload took too long")
)

// WorkersOnboardingURL is where a user registers the account's workers.dev
// subdomain (same link wrangler prints).
func WorkersOnboardingURL(accountID string) string {
	return "https://dash.cloudflare.com/" + accountID + "/workers/onboarding"
}

// APIError is a failed Cloudflare API call; Unwrap gives the sentinel.
type APIError struct {
	Method, Path string
	Status       int
	Codes        []int
	Messages     []string
	RetryAfter   time.Duration // from the Retry-After header, if any
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

const (
	codeAuthError         = 10000 // token lacks permission
	codeSubdomainNotFound = 10007 // account has no workers.dev subdomain / script not found
	codeWorkerNotFound    = 10090
	codeServiceNotFound   = 10092
	codeRouteNotFound     = 7003
)

func classify(e *APIError) error {
	text := strings.ToLower(strings.Join(e.Messages, " "))
	switch {
	case e.hasCode(1000) || e.hasCode(1001) || e.hasCode(6003) || e.hasCode(6111) ||
		e.hasCode(9106) || e.hasCode(9107) || e.hasCode(9103) ||
		strings.Contains(text, "invalid api token"):
		return ErrInvalidToken
	case e.Status == http.StatusForbidden || e.hasCode(codeAuthError) || e.hasCode(9109):
		return ErrMissingPermission
	case e.Status == http.StatusUnauthorized:
		return ErrInvalidToken
	case e.Status == http.StatusNotFound || e.hasCode(codeSubdomainNotFound) || e.hasCode(codeWorkerNotFound) ||
		e.hasCode(codeServiceNotFound) || e.hasCode(codeRouteNotFound):
		return ErrNotFound
	case strings.Contains(text, "too many") && (strings.Contains(text, "file") || strings.Contains(text, "asset")) ||
		strings.Contains(text, "file count") || strings.Contains(text, "asset count"):
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
	// CompatibilityDate is sent with the Worker; empty uses
	// DefaultCompatibilityDate.
	CompatibilityDate string

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

type request struct {
	method, path string
	bearer       string // overrides the API token (upload JWT)
	body         any    // nil, io.Reader (with contentType) or JSON value
	contentType  string
	headers      map[string]string // extra headers
}

func (c *Client) do(ctx context.Context, r request, out any) (*envelope, error) {
	var rdr io.Reader
	contentType := r.contentType
	switch b := r.body.(type) {
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
	req, err := http.NewRequestWithContext(ctx, r.method, c.baseURL()+r.path, rdr)
	if err != nil {
		return nil, err
	}
	bearer := r.bearer
	if bearer == "" {
		bearer = c.Token
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearer))
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudflare API %s %s: %w", r.method, r.path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("cloudflare API %s %s: reading response: %w", r.method, r.path, err)
	}
	var env envelope
	jerr := json.Unmarshal(raw, &env)
	if resp.StatusCode >= 300 || jerr != nil || !env.Success {
		ae := &APIError{Method: r.method, Path: r.path, Status: resp.StatusCode}
		if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && s >= 0 {
			ae.RetryAfter = time.Duration(s) * time.Second
		}
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
			return &env, fmt.Errorf("cloudflare API %s %s: decoding result: %w", r.method, r.path, err)
		}
	}
	return &env, nil
}

func (c *Client) get(ctx context.Context, path string, out any) (*envelope, error) {
	return c.do(ctx, request{method: http.MethodGet, path: path}, out)
}

// VerifyToken checks that the token is valid and active. It accepts user
// tokens (/user/tokens/verify) and account-owned tokens
// (/accounts/{id}/tokens/verify). The Workers permission itself is checked
// by EnsureSite (ErrMissingPermission).
func (c *Client) VerifyToken(ctx context.Context) error {
	if strings.TrimSpace(c.Token) == "" {
		return fmt.Errorf("empty token: %w", ErrInvalidToken)
	}
	var st struct {
		Status string `json:"status"`
	}
	_, err := c.get(ctx, "/user/tokens/verify", &st)
	if err == nil {
		return checkStatus(st.Status)
	}
	userErr := err
	accts, aerr := c.Accounts(ctx)
	if aerr != nil || len(accts) == 0 {
		return wrapVerify(userErr)
	}
	for _, a := range accts {
		st.Status = ""
		if _, err := c.get(ctx, "/accounts/"+a.ID+"/tokens/verify", &st); err == nil {
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
		env, err := c.get(ctx, fmt.Sprintf("/accounts?page=%d&per_page=50", page), &batch)
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
