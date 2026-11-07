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
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"runtime"
	"slices"
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
	// ErrKeyExists: another computer created the notes password first (409
	// KEY_EXISTS); the one made here is dropped, the notes open with theirs.
	ErrKeyExists = errors.New("blupcloud: the notes password was already created")
	// ErrMoved: another computer saved a version first.
	ErrMoved = errors.New("blupcloud: the notes changed on another computer")
	// ErrQuota: the plan's storage is full.
	ErrQuota = errors.New("blupcloud: the plan's storage is full")
	// ErrDevicesLimit: the plan allows no more computers signed in.
	ErrDevicesLimit = errors.New("blupcloud: the plan allows no more computers")
	// ErrDeployBusy: a publish of the same site is already under way.
	ErrDeployBusy = errors.New("blupcloud: a publish of this site is already under way")
	// ErrSyncBusy: a sync, a preparation or a cleanup of the account is
	// under way; try again in a moment.
	ErrSyncBusy = errors.New("blupcloud: the notes are busy with another sync")
	// ErrVaultMissing: no such vault on this account.
	ErrVaultMissing = errors.New("blupcloud: vault not found")
	// ErrTicketExpired: a prepared version waited too long (or was
	// cancelled); prepare it again.
	ErrTicketExpired = errors.New("blupcloud: the prepared version expired")
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
// A 429 (a limit of Blup Cloud reached) is sent again when that is safe:
// see send.
func (c *Client) do(ctx context.Context, method, path string, body any, contentType string, out any) error {
	var data []byte
	switch b := body.(type) {
	case nil:
	case []byte:
		data = b
	case *bytes.Buffer:
		data = b.Bytes()
	default:
		j, err := json.Marshal(b)
		if err != nil {
			return err
		}
		data, contentType = j, "application/json"
	}
	res, err := c.send(ctx, method, path, data, body != nil, contentType)
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
		case e.Code == "KEY_EXISTS":
			return ErrKeyExists
		case e.Code == "MOVED":
			return ErrMoved
		case e.Code == "QUOTA":
			return ErrQuota
		case e.Code == "DEVICES_LIMIT":
			return ErrDevicesLimit
		case e.Code == "DEPLOY_BUSY":
			return ErrDeployBusy
		case e.Code == "SYNC_BUSY":
			return ErrSyncBusy
		case e.Code == "TICKET_EXPIRED":
			return ErrTicketExpired
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
		data, err := io.ReadAll(limited)
		if err != nil {
			return err
		}
		if len(data) > maxBody {
			return errors.New("blupcloud: answer too large")
		}
		// "null" (or nothing) leaves a struct empty without an error:
		// only an answer read into a pointer (GET .../domain) may be null.
		if t := bytes.TrimSpace(data); len(t) == 0 || bytes.Equal(t, []byte("null")) {
			if v := reflect.ValueOf(out); v.Kind() == reflect.Pointer && v.Elem().Kind() == reflect.Pointer {
				return nil
			}
			return fmt.Errorf("%w: empty answer", ErrBadAnswer)
		}
		return json.Unmarshal(data, out)
	}
}

// Retrying a 429: at most maxAttempts sends in all, waiting what the
// answer's Retry-After says (up to maxRetryWait), or a short growing wait
// with jitter when it says nothing.
const (
	maxAttempts  = 4
	maxRetryWait = 60 * time.Second
)

// retrySleep waits d, or less when ctx ends (tests replace it).
var retrySleep = func(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// send makes the request, again on a 429 when it is safe to repeat: the
// methods that are idempotent (GET, HEAD, PUT, DELETE: a file or block is
// put under its hash) and the POSTs that only ask (".../missing"). Other
// POSTs (create a site or vault, save a version, publish) are never sent
// twice: their 429 goes back as is.
func (c *Client) send(ctx context.Context, method, path string, data []byte, hasBody bool, contentType string) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		var r io.Reader
		if hasBody {
			r = bytes.NewReader(data)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+"/api"+path, r)
		if err != nil {
			return nil, err
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
		req.Header.Set("User-Agent", UserAgent())
		if t, _ := ctx.Value(ticketKey{}).(string); t != "" {
			req.Header.Set("x-blup-upload-ticket", t)
		}
		res, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusTooManyRequests || attempt+1 >= maxAttempts || !retriable(method, path) {
			return res, nil
		}
		wait := retryWait(res.Header.Get("Retry-After"), attempt)
		io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16))
		res.Body.Close()
		if err := retrySleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}

// retriable tells whether a request may be sent again after a 429.
func retriable(method, path string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete:
		return true
	case http.MethodPost:
		p, _, _ := strings.Cut(path, "?")
		return strings.HasSuffix(p, "/missing")
	}
	return false
}

// retryWait is how long to wait before sending again: Retry-After (seconds,
// or a date) up to maxRetryWait, else 0.5 s, 1 s, 2 s... with jitter.
func retryWait(header string, attempt int) time.Duration {
	if h := strings.TrimSpace(header); h != "" {
		if n, err := strconv.Atoi(h); err == nil && n >= 0 {
			return min(time.Duration(n)*time.Second, maxRetryWait)
		}
		if t, err := http.ParseTime(h); err == nil {
			return min(max(time.Until(t), 0), maxRetryWait)
		}
	}
	base := 500 * time.Millisecond << min(attempt, 6)
	return min(base+rand.N(base/2+1), maxRetryWait)
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
			// CustomDomain: the plan's sites can have their own domain
			// (Max and Ultra).
			CustomDomain bool `json:"customDomain"`
		} `json:"limits"`
	} `json:"plan"`
	Usage struct {
		StorageBytes int64 `json:"storageBytes"`
		// StorageLimitBytes is the plan's storage (0 from an older cloud:
		// StorageMB then counts).
		StorageLimitBytes int64 `json:"storageLimitBytes"`
		// After a plan with less storage, while the use is above it: until
		// StorageGraceUntil (ISO 8601) the previous plan's
		// StorageGraceLimitBytes still counts. Nil otherwise.
		StorageGraceUntil      *string `json:"storageGraceUntil"`
		StorageGraceLimitBytes *int64  `json:"storageGraceLimitBytes"`
		// HistoryBytes is what the old versions of all the vaults take;
		// nil when the cloud can't tell safely right now (a sync under
		// way, older records): unknown, never 0.
		HistoryBytes *int64 `json:"historyBytes"`
		Vaults       int    `json:"vaults"`
		Sites        int    `json:"sites"`
		Devices      int    `json:"devices"`
	} `json:"usage"`
}

// HistoryPurge is what deleting a vault's old versions freed, and the
// account's storage use after it.
type HistoryPurge struct {
	FreedBytes   int64 `json:"freedBytes"`
	StorageBytes int64 `json:"storageBytes"`
}

// PurgeHistory deletes the old versions of a vault on the cloud; the
// current one stays. Calling it again frees nothing more.
func (c *Client) PurgeHistory(ctx context.Context, vaultID string) (HistoryPurge, error) {
	if !ValidVaultID(vaultID) {
		return HistoryPurge{}, ErrVaultMissing
	}
	var r struct {
		FreedBytes   *int64 `json:"freedBytes"`
		StorageBytes *int64 `json:"storageBytes"`
	}
	err := c.do(ctx, http.MethodDelete, "/sync/vaults/"+url.PathEscape(vaultID)+"/history", nil, "", &r)
	var e *apiError
	if errors.As(err, &e) && (e.Code == "NOT_FOUND" || e.Status == http.StatusNotFound) {
		return HistoryPurge{}, ErrVaultMissing
	}
	if err != nil {
		return HistoryPurge{}, err
	}
	if r.FreedBytes == nil || r.StorageBytes == nil || *r.FreedBytes < 0 || *r.StorageBytes < 0 {
		return HistoryPurge{}, fmt.Errorf("%w: history purge", ErrBadAnswer)
	}
	return HistoryPurge{FreedBytes: *r.FreedBytes, StorageBytes: *r.StorageBytes}, nil
}

// VaultGone tells whether a vault is surely no longer on the account: it
// is not in the list. Anything else (offline, signed out) is false, so a
// device never lets go of a vault by mistake.
func (c *Client) VaultGone(ctx context.Context, vaultID string) bool {
	vaults, err := c.Vaults(ctx)
	return err == nil && !slices.ContainsFunc(vaults, func(v Vault) bool { return v.ID == vaultID })
}

// RenameVault gives a vault a new name, sealed (keys.SealName).
func (c *Client) RenameVault(ctx context.Context, vaultID string, name []byte) error {
	if !ValidVaultID(vaultID) {
		return ErrVaultMissing
	}
	err := c.do(ctx, http.MethodPatch, "/sync/vaults/"+url.PathEscape(vaultID), map[string]any{"name": name}, "", nil)
	var e *apiError
	if errors.As(err, &e) && (e.Code == "NOT_FOUND" || e.Status == http.StatusNotFound) {
		return ErrVaultMissing
	}
	return err
}

// ErrCantDelete: the cloud can't delete vaults (an older one); nothing
// was deleted.
var ErrCantDelete = errors.New("blupcloud: this cloud can't delete vaults yet")

// DeleteVault deletes a vault on the cloud for good: every version of it
// and the blocks no other vault uses. One already gone is fine; a 404 for
// one still listed means a cloud without the route, so it is an error.
func (c *Client) DeleteVault(ctx context.Context, vaultID string) error {
	if !ValidVaultID(vaultID) {
		return ErrVaultMissing
	}
	err := c.do(ctx, http.MethodDelete, "/sync/vaults/"+url.PathEscape(vaultID), nil, "", nil)
	var e *apiError
	if !errors.As(err, &e) || (e.Code != "NOT_FOUND" && e.Status != http.StatusNotFound) {
		return err
	}
	vaults, err := c.Vaults(ctx)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(vaults, func(v Vault) bool { return v.ID == vaultID }) {
		return ErrCantDelete
	}
	return nil
}

// Me returns the signed in account.
func (c *Client) Me(ctx context.Context) (Account, error) {
	var a Account
	if err := c.do(ctx, http.MethodGet, "/me", nil, "", &a); err != nil {
		return Account{}, err
	}
	u := &a.Usage
	// A grace that isn't a date or has no limit is no grace.
	if u.StorageGraceUntil != nil {
		if _, err := time.Parse(time.RFC3339, *u.StorageGraceUntil); err != nil || u.StorageGraceLimitBytes == nil || *u.StorageGraceLimitBytes < 0 {
			u.StorageGraceUntil, u.StorageGraceLimitBytes = nil, nil
		}
	}
	if u.StorageGraceUntil == nil {
		u.StorageGraceLimitBytes = nil
	}
	return a, a.check()
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
	if err := c.do(ctx, http.MethodGet, "/sync/key", nil, "", &r); err != nil {
		return cloudcrypt.Wrapped{}, err
	}
	// A key with nothing sealed in it would read as a wrong password.
	if w := r.Wrapped; len(w.ByPassword) == 0 || len(w.ByRecovery) == 0 || len(w.Params.Salt) == 0 {
		return cloudcrypt.Wrapped{}, fmt.Errorf("%w: key", ErrBadAnswer)
	}
	return r.Wrapped, nil
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
		Vaults *[]Vault `json:"vaults"`
	}
	if err := c.do(ctx, http.MethodGet, "/sync/vaults", nil, "", &r); err != nil {
		return nil, err
	}
	// Without the list, "no vaults" would make a second one.
	if r.Vaults == nil {
		return nil, fmt.Errorf("%w: vaults", ErrBadAnswer)
	}
	out := (*r.Vaults)[:0]
	for _, v := range *r.Vaults {
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
	var v struct {
		Head *int64 `json:"head"`
	}
	err := c.do(ctx, http.MethodGet, "/sync/vaults/"+url.PathEscape(id), nil, "", &v)
	var e *apiError
	if errors.As(err, &e) && e.Code == "NOT_FOUND" {
		return 0, ErrVaultMissing
	}
	if err != nil {
		return 0, err
	}
	if v.Head == nil || *v.Head < 0 {
		return 0, fmt.Errorf("%w: vault head", ErrBadAnswer)
	}
	return *v.Head, nil
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
		r, err := missingAnswer(func(r any) error {
			return c.do(ctx, http.MethodPost, "/sync/blocks/missing", map[string]any{"ids": ids[:n]}, "", r)
		}, ids[:n])
		if err != nil {
			return nil, err
		}
		out = append(out, r...)
		ids = ids[n:]
	}
	return out, nil
}

// missingAnswer reads which of asked the cloud lacks: the list must be
// there (without it nothing would go up) and name only what was asked,
// each once.
func missingAnswer(call func(out any) error, asked []string) ([]string, error) {
	var r struct {
		Missing *[]string `json:"missing"`
	}
	if err := call(&r); err != nil {
		return nil, err
	}
	if r.Missing == nil || len(*r.Missing) > len(asked) {
		return nil, fmt.Errorf("%w: missing", ErrBadAnswer)
	}
	set := make(map[string]bool, len(asked))
	for _, id := range asked {
		set[id] = true
	}
	for _, id := range *r.Missing {
		if !set[id] {
			return nil, fmt.Errorf("%w: missing", ErrBadAnswer)
		}
		delete(set, id) // each one once
	}
	return *r.Missing, nil
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
	if err := c.do(ctx, http.MethodPost, "/sync/vaults/"+url.PathEscape(vault)+"/versions", &buf, w.FormDataContentType(), &r); err != nil {
		return 0, err
	}
	if r.N != parent+1 {
		return 0, fmt.Errorf("%w: saved as version %d after %d", ErrBadAnswer, r.N, parent)
	}
	return r.N, nil
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
		Versions *[]VersionInfo `json:"versions"`
	}
	if err := c.do(ctx, http.MethodGet, "/sync/vaults/"+url.PathEscape(vault)+"/versions"+q, nil, "", &r); err != nil {
		return nil, err
	}
	if r.Versions == nil {
		return nil, fmt.Errorf("%w: no versions", ErrBadAnswer)
	}
	if err := checkVersions(*r.Versions, before); err != nil {
		return nil, err
	}
	return *r.Versions, nil
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
	if err := c.do(ctx, http.MethodGet, "/ai/usage", nil, "", &u); err != nil {
		return AIUsage{}, err
	}
	return u, u.check()
}

// AIBaseURL is Blup Cloud's AI API, which speaks OpenRouter's: a narration
// client pointed here with the session as its key narrates through the plan.
func AIBaseURL() string { return BaseURL() + "/api/ai" }
