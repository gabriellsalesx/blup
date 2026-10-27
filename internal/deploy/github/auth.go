package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultOAuthBaseURL hosts the device-flow endpoints.
const DefaultOAuthBaseURL = "https://github.com"

// DefaultDeviceScopes are the OAuth scopes requested by the device flow.
// "repo" is what the Pages REST docs require for OAuth app tokens
// (https://docs.github.com/en/rest/pages/pages); it also covers creating the
// repository and writing git data. "public_repo" alone is not documented as
// sufficient for the Pages endpoints.
var DefaultDeviceScopes = []string{"repo"}

// Device-flow errors.
var (
	ErrDeviceExpired      = errors.New("github: device code expired, start the login again")
	ErrAccessDenied       = errors.New("github: authorization was denied by the user")
	ErrDeviceFlowDisabled = errors.New("github: device flow is not enabled for this OAuth app")
)

// DeviceFlow implements the OAuth 2.0 Device Authorization Grant against
// GitHub (https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#device-flow).
// No client secret is needed.
type DeviceFlow struct {
	ClientID   string
	Scopes     []string // nil means DefaultDeviceScopes
	BaseURL    string   // default https://github.com
	HTTPClient *http.Client

	sleep func(ctx context.Context, d time.Duration) error
}

// DeviceCode is what the user must see: open VerificationURI and type UserCode.
type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"` // seconds
	Interval        int    `json:"interval"`   // seconds between polls
}

// Token is the result of a successful device flow.
type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type oauthResp struct {
	DeviceCode
	Token
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
}

func (f *DeviceFlow) post(ctx context.Context, path string, form url.Values) (*oauthResp, error) {
	base := f.BaseURL
	if base == "" {
		base = DefaultOAuthBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "blup")
	hc := f.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: device flow: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("github: device flow: %w", err)
	}
	var r oauthResp
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("github: device flow: HTTP %d: unexpected response", resp.StatusCode)
	}
	if r.Error == "" && resp.StatusCode >= 300 {
		r.Error = fmt.Sprintf("http_%d", resp.StatusCode)
	}
	return &r, nil
}

// Start requests a device and user code (POST /login/device/code).
func (f *DeviceFlow) Start(ctx context.Context) (*DeviceCode, error) {
	if f.ClientID == "" {
		return nil, errors.New("github: device flow: ClientID is empty")
	}
	scopes := f.Scopes
	if scopes == nil {
		scopes = DefaultDeviceScopes
	}
	r, err := f.post(ctx, "/login/device/code", url.Values{
		"client_id": {f.ClientID},
		"scope":     {strings.Join(scopes, " ")},
	})
	if err != nil {
		return nil, err
	}
	if r.Error != "" {
		return nil, deviceError(r)
	}
	dc := r.DeviceCode
	if dc.Interval <= 0 {
		dc.Interval = 5
	}
	if dc.ExpiresIn <= 0 {
		dc.ExpiresIn = 900
	}
	if dc.VerificationURI == "" {
		dc.VerificationURI = "https://github.com/login/device"
	}
	return &dc, nil
}

// Poll waits until the user authorizes (POST /login/oauth/access_token),
// honouring interval, authorization_pending and slow_down (+5s, or the
// interval GitHub returns). It stops at expiry or when ctx is done.
func (f *DeviceFlow) Poll(ctx context.Context, dc *DeviceCode) (*Token, error) {
	interval := time.Duration(dc.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	sleep := f.sleep
	if sleep == nil {
		sleep = sleepCtx
	}
	expires := time.Duration(dc.ExpiresIn) * time.Second
	if expires <= 0 {
		expires = 15 * time.Minute
	}
	var waited time.Duration
	for {
		if waited >= expires {
			return nil, ErrDeviceExpired
		}
		if err := sleep(ctx, interval); err != nil {
			return nil, err
		}
		waited += interval
		r, err := f.post(ctx, "/login/oauth/access_token", url.Values{
			"client_id":   {f.ClientID},
			"device_code": {dc.DeviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		})
		if err != nil {
			// Transient network error: keep polling until expiry.
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue
		}
		switch r.Error {
		case "":
			if r.AccessToken == "" {
				return nil, errors.New("github: device flow: empty access token")
			}
			t := r.Token
			return &t, nil
		case "authorization_pending":
			continue
		case "slow_down":
			if r.Interval > 0 {
				interval = time.Duration(r.Interval) * time.Second
			} else {
				interval += 5 * time.Second
			}
			continue
		default:
			return nil, deviceError(r)
		}
	}
}

func deviceError(r *oauthResp) error {
	switch r.Error {
	case "expired_token", "token_expired":
		return ErrDeviceExpired
	case "access_denied":
		return ErrAccessDenied
	case "device_flow_disabled":
		return ErrDeviceFlowDisabled
	case "incorrect_client_credentials":
		return fmt.Errorf("github: device flow: unknown client ID: %s", r.ErrorDescription)
	}
	if r.ErrorDescription != "" {
		return fmt.Errorf("github: device flow: %s: %s", r.Error, r.ErrorDescription)
	}
	return fmt.Errorf("github: device flow: %s", r.Error)
}
