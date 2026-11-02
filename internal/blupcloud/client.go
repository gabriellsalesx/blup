// Package blupcloud talks to Blup Cloud: signing Blup Studio in through the
// browser (device authorization) and the sync API. Notes are sealed by
// internal/cloudcrypt before they leave the computer.
package blupcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
)

// DefaultURL is Blup Cloud's address; BLUP_CLOUD_URL overrides it (tests and
// development).
const DefaultURL = "https://cloud.getblup.io"

// ClientID is how Blup Studio identifies itself when signing in.
const ClientID = "blup-studio"

var (
	// ErrSignedOut: no session, or it expired or was ended in the panel.
	ErrSignedOut = errors.New("blupcloud: not signed in to Blup Cloud")
	// ErrNoKey: sync was never turned on for this account (no notes password).
	ErrNoKey = errors.New("blupcloud: the notes password was never created")
	// ErrMoved: another computer saved a version first.
	ErrMoved = errors.New("blupcloud: the notes changed on another computer")
	// ErrQuota: the plan's storage is full.
	ErrQuota = errors.New("blupcloud: the plan's storage is full")
	// ErrDevicesLimit: the plan allows no more computers signed in.
	ErrDevicesLimit = errors.New("blupcloud: the plan allows no more computers")
	// ErrDenied: the sign in was refused in the browser.
	ErrDenied = errors.New("blupcloud: sign in refused in the browser")
	// ErrExpired: the code shown was not approved in time.
	ErrExpired = errors.New("blupcloud: the sign in code expired")
)

// UserAgent names this computer to Blup Cloud, as the account panel lists
// it: "Blup Studio (Linux; my-pc)".
func UserAgent() string {
	host, _ := os.Hostname()
	if host == "" {
		host = "?"
	}
	osName := map[string]string{"linux": "Linux", "darwin": "macOS", "windows": "Windows"}[runtime.GOOS]
	if osName == "" {
		osName = runtime.GOOS
	}
	return fmt.Sprintf("Blup Studio (%s; %s)", osName, strings.Map(func(r rune) rune {
		if r == '(' || r == ')' || r == ';' || r < ' ' {
			return -1
		}
		return r
	}, host))
}

// BaseURL is DefaultURL or BLUP_CLOUD_URL.
func BaseURL() string {
	if u := os.Getenv("BLUP_CLOUD_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return DefaultURL
}

// ErrBadVault: a vault id that is not one Blup Cloud makes (they are
// UUIDs). Ids come from the cloud and name folders here, so anything else
// (a "../" in particular) is refused.
var ErrBadVault = errors.New("blupcloud: invalid vault id")

// ValidVaultID tells whether id looks like a vault id: 1 to 64 letters,
// digits and hyphens.
func ValidVaultID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

// maxBody bounds what is read from an answer: a sealed manifest (32 MiB at
// most on the cloud) or block (4 MiB), with room to spare. A larger answer
// is not Blup Cloud's and would only fill the memory.
const maxBody = 64 << 20

// Client calls Blup Cloud with a session token.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// New returns a client for BaseURL() with token.
func New(token string) *Client {
	return &Client{BaseURL: BaseURL(), Token: token, HTTP: &http.Client{Timeout: 2 * time.Minute}}
}

type apiError struct {
	Status int
	Code   string `json:"code"`
	OAuth  string `json:"error"` // device sign in errors
}

func (e *apiError) Error() string {
	return fmt.Sprintf("blupcloud: %d %s%s", e.Status, e.Code, e.OAuth)
}

// do sends a request; body is JSON (any), raw bytes or a multipart form.
func (c *Client) do(ctx context.Context, method, path string, body any, contentType string, out any) error {
	var r io.Reader
	switch b := body.(type) {
	case nil:
	case []byte:
		r = bytes.NewReader(b)
	case *bytes.Buffer:
		r = b
	default:
		data, err := json.Marshal(b)
		if err != nil {
			return err
		}
		r, contentType = bytes.NewReader(data), "application/json"
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+"/api"+path, r)
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("User-Agent", UserAgent())
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		e := &apiError{Status: res.StatusCode}
		json.NewDecoder(io.LimitReader(res.Body, 1<<16)).Decode(e)
		switch {
		case res.StatusCode == http.StatusUnauthorized:
			return ErrSignedOut
		case e.Code == "NO_KEY":
			return ErrNoKey
		case e.Code == "MOVED":
			return ErrMoved
		case e.Code == "QUOTA":
			return ErrQuota
		case e.Code == "DEVICES_LIMIT":
			return ErrDevicesLimit
		}
		return e
	}
	limited := io.LimitReader(res.Body, maxBody+1)
	switch o := out.(type) {
	case nil:
		return nil
	case *[]byte:
		*o, err = io.ReadAll(limited)
		if err == nil && len(*o) > maxBody {
			*o, err = nil, errors.New("blupcloud: answer too large")
		}
		return err
	default:
		return json.NewDecoder(limited).Decode(out)
	}
}

// Login is a sign in in progress: show Code and open URL; Wait returns the
// session token once the person approves in the browser.
type Login struct {
	Code       string // user code, e.g. "ABCD-EFGH"
	URL        string // the page that approves it, with the code filled in
	deviceCode string
	interval   time.Duration
	expires    time.Time
	c          *Client
}

// StartLogin asks Blup Cloud for a sign in code.
func StartLogin(ctx context.Context) (*Login, error) {
	c := New("")
	var r struct {
		DeviceCode string `json:"device_code"`
		UserCode   string `json:"user_code"`
		URI        string `json:"verification_uri"`
		URIFull    string `json:"verification_uri_complete"`
		Interval   int    `json:"interval"`
		ExpiresIn  int    `json:"expires_in"`
	}
	if err := c.do(ctx, http.MethodPost, "/auth/device/code", map[string]string{"client_id": ClientID}, "", &r); err != nil {
		return nil, err
	}
	code := r.UserCode
	if len(code) == 8 {
		code = code[:4] + "-" + code[4:]
	}
	u := r.URIFull
	if u == "" {
		u = r.URI + "?user_code=" + url.QueryEscape(r.UserCode)
	}
	return &Login{
		Code: code, URL: u, deviceCode: r.DeviceCode, c: c,
		interval: time.Duration(max(r.Interval, 1)) * time.Second,
		expires:  time.Now().Add(time.Duration(r.ExpiresIn) * time.Second),
	}, nil
}

// Wait polls until the code is approved (the token), refused, or expired.
func (l *Login) Wait(ctx context.Context) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(l.interval):
		}
		var r struct {
			AccessToken string `json:"access_token"`
		}
		err := l.c.do(ctx, http.MethodPost, "/auth/device/token", map[string]string{
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
			"device_code": l.deviceCode,
			"client_id":   ClientID,
		}, "", &r)
		var e *apiError
		switch {
		case err == nil && r.AccessToken != "":
			return r.AccessToken, nil
		case errors.As(err, &e) && e.OAuth == "authorization_pending":
		case errors.As(err, &e) && e.OAuth == "slow_down":
			l.interval += 5 * time.Second
		case errors.As(err, &e) && e.OAuth == "access_denied":
			return "", ErrDenied
		case errors.As(err, &e) && e.OAuth == "expired_token", time.Now().After(l.expires):
			return "", ErrExpired
		case err != nil:
			return "", err
		}
	}
}

// Account is who is signed in and their plan.
type Account struct {
	ID    string  `json:"id"`
	Email *string `json:"email"`
	Name  string  `json:"name"`
	Plan  struct {
		Name   string `json:"name"`
		Label  string `json:"label"`
		Limits struct {
			StorageMB   int  `json:"storageMB"`
			MaxFileMB   int  `json:"maxFileMB"`
			Sites       int  `json:"sites"`
			Devices     int  `json:"devices"`
			Live        bool `json:"live"`
			HistoryDays int  `json:"historyDays"`
			AIUsage     int  `json:"aiUsage"` // AI through Blup Cloud, as a multiple of Pro's (0: none)
		} `json:"limits"`
	} `json:"plan"`
	Usage struct {
		StorageBytes int64 `json:"storageBytes"`
		Vaults       int   `json:"vaults"`
		Sites        int   `json:"sites"`
		Devices      int   `json:"devices"`
	} `json:"usage"`
}

// Me returns the signed in account.
func (c *Client) Me(ctx context.Context) (Account, error) {
	var a Account
	return a, c.do(ctx, http.MethodGet, "/me", nil, "", &a)
}

// SignOut ends this computer's session.
func (c *Client) SignOut(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/auth/sign-out", map[string]any{}, "", nil)
}

// Key returns the wrapped master key (ErrNoKey before the first time).
func (c *Client) Key(ctx context.Context) (cloudcrypt.Wrapped, error) {
	var r struct {
		Wrapped cloudcrypt.Wrapped `json:"wrapped"`
	}
	return r.Wrapped, c.do(ctx, http.MethodGet, "/sync/key", nil, "", &r)
}

// SaveKey stores the wrapped master key (the first time, or a new notes
// password).
func (c *Client) SaveKey(ctx context.Context, w cloudcrypt.Wrapped) error {
	return c.do(ctx, http.MethodPut, "/sync/key", map[string]any{"wrapped": w}, "", nil)
}

// Vault is a synced folder of notes; Name is sealed.
type Vault struct {
	ID   string `json:"id"`
	Name []byte `json:"name"`
	Head int64  `json:"head"`
}

// Vaults lists the account's vaults.
func (c *Client) Vaults(ctx context.Context) ([]Vault, error) {
	var r struct {
		Vaults []Vault `json:"vaults"`
	}
	if err := c.do(ctx, http.MethodGet, "/sync/vaults", nil, "", &r); err != nil {
		return nil, err
	}
	out := r.Vaults[:0]
	for _, v := range r.Vaults {
		if ValidVaultID(v.ID) {
			out = append(out, v)
		}
	}
	return out, nil
}

// CreateVault makes a vault with a sealed name.
func (c *Client) CreateVault(ctx context.Context, name []byte) (Vault, error) {
	v := Vault{Name: name}
	if err := c.do(ctx, http.MethodPost, "/sync/vaults", map[string]any{"name": name}, "", &v); err != nil {
		return v, err
	}
	if !ValidVaultID(v.ID) {
		return Vault{}, ErrBadVault
	}
	return v, nil
}

// VaultHead returns a vault's current version number (0 = empty).
func (c *Client) VaultHead(ctx context.Context, id string) (int64, error) {
	var v Vault
	err := c.do(ctx, http.MethodGet, "/sync/vaults/"+url.PathEscape(id), nil, "", &v)
	return v.Head, err
}

// Manifest returns the sealed manifest of version n.
func (c *Client) Manifest(ctx context.Context, vault string, n int64) ([]byte, error) {
	var b []byte
	return b, c.do(ctx, http.MethodGet, "/sync/vaults/"+url.PathEscape(vault)+"/versions/"+strconv.FormatInt(n, 10), nil, "", &b)
}

// Missing returns which block ids Blup Cloud does not have yet.
func (c *Client) Missing(ctx context.Context, ids []string) ([]string, error) {
	var out []string
	for len(ids) > 0 {
		n := min(len(ids), 1000)
		var r struct {
			Missing []string `json:"missing"`
		}
		if err := c.do(ctx, http.MethodPost, "/sync/blocks/missing", map[string]any{"ids": ids[:n]}, "", &r); err != nil {
			return nil, err
		}
		out = append(out, r.Missing...)
		ids = ids[n:]
	}
	return out, nil
}

// validBlockID: 64 hex digits (an HMAC-SHA256), as cloudcrypt makes them.
func validBlockID(id string) bool {
	if len(id) != 64 {
		return false
	}
	for _, c := range id {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// PutBlock stores a sealed block.
func (c *Client) PutBlock(ctx context.Context, id string, sealed []byte) error {
	if !validBlockID(id) {
		return cloudcrypt.ErrCorrupt
	}
	return c.do(ctx, http.MethodPut, "/sync/blocks/"+id, sealed, "application/octet-stream", nil)
}

// Block returns a sealed block.
func (c *Client) Block(ctx context.Context, id string) ([]byte, error) {
	if !validBlockID(id) {
		return nil, cloudcrypt.ErrCorrupt
	}
	var b []byte
	return b, c.do(ctx, http.MethodGet, "/sync/blocks/"+id, nil, "", &b)
}

// Commit saves version parent+1 (ErrMoved when parent is no longer the
// current one).
func (c *Client) Commit(ctx context.Context, vault string, parent int64, blocks []string, sealedManifest []byte) (int64, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("parent", strconv.FormatInt(parent, 10))
	w.WriteField("blocks", strings.Join(blocks, "\n"))
	f, err := w.CreateFormFile("manifest", "manifest")
	if err != nil {
		return 0, err
	}
	f.Write(sealedManifest)
	if err := w.Close(); err != nil {
		return 0, err
	}
	var r struct {
		N int64 `json:"n"`
	}
	err = c.do(ctx, http.MethodPost, "/sync/vaults/"+url.PathEscape(vault)+"/versions", &buf, w.FormDataContentType(), &r)
	return r.N, err
}

// VersionInfo is a version of a vault still kept.
type VersionInfo struct {
	N         int64     `json:"n"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

// Versions lists the versions kept, newest first, 50 at a time: before 0
// starts at the newest, else at the version before that one.
func (c *Client) Versions(ctx context.Context, vault string, before int64) ([]VersionInfo, error) {
	q := ""
	if before > 0 {
		q = "?before=" + strconv.FormatInt(before, 10)
	}
	var r struct {
		Versions []VersionInfo `json:"versions"`
	}
	return r.Versions, c.do(ctx, http.MethodGet, "/sync/vaults/"+url.PathEscape(vault)+"/versions"+q, nil, "", &r)
}

// AIUsage is the plan's AI allowance this month and what was used, in
// millionths of a US dollar.
type AIUsage struct {
	MonthMicros     int64     `json:"monthMicros"`
	WeekCapMicros   int64     `json:"weekCapMicros"`
	UsedMonthMicros int64     `json:"usedMonthMicros"`
	UsedWeekMicros  int64     `json:"usedWeekMicros"`
	PerCharMicros   int64     `json:"perCharMicros"`
	Percent         int       `json:"percent"`
	ResetsAt        time.Time `json:"resetsAt"`
}

// Left is how much can still be spent now (the month's rest, or the
// week's, whichever is smaller).
func (u AIUsage) Left() int64 {
	return max(0, min(u.MonthMicros-u.UsedMonthMicros, u.WeekCapMicros-u.UsedWeekMicros))
}

// AIUsage reads the account's AI allowance.
func (c *Client) AIUsage(ctx context.Context) (AIUsage, error) {
	var u AIUsage
	return u, c.do(ctx, http.MethodGet, "/ai/usage", nil, "", &u)
}

// AIBaseURL is Blup Cloud's AI API, which speaks OpenRouter's: a narration
// client pointed here with the session as its key narrates through the plan.
func AIBaseURL() string { return BaseURL() + "/api/ai" }
