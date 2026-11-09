package main

import (
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/notesync"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func init() {
	// The files the sync changed, one per line, by their path in the vault.
	application.RegisterEvent[string]("notesChanged")
}

// notesChanged tells the screen which files of the open vault changed on
// disk. Without the app (the tests), there is no screen to tell.
var notesChanged = func(list string) {
	if a := application.Get(); a != nil {
		a.Event.Emit("notesChanged", list)
	}
}

// cloudURL is the Blup Cloud the app talks to. Empty, it is the real one
// (blupcloud.DefaultURL, over HTTPS); a build can give another with
// -ldflags "-X main.cloudURL=...". The debug build talks to a Blup Cloud on
// the developer's computer, reached through adb reverse (cloud_debug.go).
var cloudURL = ""

// Account is what the phone shows about Blup Cloud. Notes is "new" (no
// notes password yet), "locked" (not typed on this phone) or "ready".
type Account struct {
	SignedIn bool   `json:"signedIn"`
	Email    string `json:"email"`
	Plan     string `json:"plan"`
	Live     bool   `json:"live"`
	Notes    string `json:"notes"`
	Vault    string `json:"vault"` // the vault linked here ("" for none)
	Title    string `json:"title"`
	// What the plan gives and what is in use.
	StorageBytes int64 `json:"storageBytes"`
	StorageLimit int64 `json:"storageLimit"`
	Devices      int   `json:"devices"`
	DevicesMax   int   `json:"devicesMax"`
}

// LoginCode is what the person checks in the browser.
type LoginCode struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

// RemoteVault is a vault of the account.
type RemoteVault struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// SyncResult tells what a sync did.
type SyncResult struct {
	Sent      int      `json:"sent"`
	Received  int      `json:"received"`
	Conflicts []string `json:"conflicts"`
	At        int64    `json:"at"`
	// Gone: the vault was deleted from Blup Cloud on another device; this
	// phone let go of it (the notes stay, and no longer sync).
	Gone bool `json:"gone"`
}

// link is the vault of this phone, kept next to the sync state.
type link struct{ ID, Title string }

// CloudService signs in, links this phone's notes to a vault and syncs
// them. It uses the Blup packages directly and leaves the site's settings
// alone: a phone has no site, so only the notes travel.
type CloudService struct {
	mu     sync.Mutex
	login  *blupcloud.Login
	cancel context.CancelFunc
}

func prepare() error {
	if _, err := vaultDir(); err != nil {
		return err
	}
	base := cmp.Or(cloudURL, blupcloud.DefaultURL)
	if !safeURL(base) {
		return fail("unsafeCloud", base)
	}
	return os.Setenv("BLUP_CLOUD_URL", base)
}

// safeURL: the session token and the notes only travel over HTTPS. Plain
// HTTP is only for the phone itself, the debug build's Blup Cloud reached
// through adb reverse (cloud_debug.go); a release given an http:// address
// by mistake refuses it instead of sending the token in the clear.
func safeURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return false
	}
	switch u.Scheme {
	case "https":
		return true
	case "http":
		h := u.Hostname()
		if h == "localhost" {
			return true
		}
		ip := net.ParseIP(h)
		return ip != nil && ip.IsLoopback()
	}
	return false
}

func client() (*blupcloud.Client, config.Credentials, error) {
	if err := prepare(); err != nil {
		return nil, config.Credentials{}, err
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, creds, err
	}
	if creds.BlupToken == "" {
		return nil, creds, blupcloud.ErrSignedOut
	}
	return blupcloud.New(creds.BlupToken), creds, nil
}

func syncDir() (string, error) {
	root, err := currentRoot()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "sync")
	return dir, os.MkdirAll(dir, 0o700)
}

func loadLink() (link, error) {
	root, err := currentRoot()
	if err != nil {
		return link{}, err
	}
	return linkAt(root)
}

// linkAt is the Cloud vault of a vault of this phone.
func linkAt(root string) (link, error) {
	raw, err := os.ReadFile(filepath.Join(root, "sync", "vault"))
	if errors.Is(err, os.ErrNotExist) {
		return link{}, nil
	}
	if err != nil {
		return link{}, err
	}
	id, title, _ := cut(string(raw))
	return link{id, title}, nil
}

func cut(s string) (string, string, bool) {
	for i, r := range s {
		if r == '\n' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

func saveLink(l link) error {
	dir, err := syncDir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "vault"), []byte(l.ID+"\n"+l.Title), 0o600)
}

// Account reads the account. Signed out is not an error.
func (s *CloudService) Account(ctx context.Context) (Account, error) {
	c, creds, err := client()
	if errors.Is(err, blupcloud.ErrSignedOut) {
		live.Want(false)
		return Account{}, nil
	}
	if err != nil {
		return Account{}, err
	}
	me, err := c.Me(ctx)
	if errors.Is(err, blupcloud.ErrSignedOut) {
		live.Want(false)
		creds.BlupToken = ""
		return Account{}, creds.Save()
	}
	if err != nil {
		return Account{}, err
	}
	if err := app.BindNotesKey(ctx, c, &creds, me.ID); err != nil {
		return Account{}, err
	}
	a := Account{SignedIn: true, Plan: me.Plan.Label, Live: me.Plan.Limits.Live,
		StorageBytes: me.Usage.StorageBytes, StorageLimit: me.Usage.StorageLimitBytes,
		Devices: me.Usage.Devices, DevicesMax: me.Plan.Limits.Devices}
	if a.StorageLimit == 0 { // an older cloud
		a.StorageLimit = int64(me.Plan.Limits.StorageMB) << 20
	}
	if me.Usage.StorageGraceUntil != nil && me.Usage.StorageGraceLimitBytes != nil {
		a.StorageLimit = *me.Usage.StorageGraceLimitBytes // the previous plan's, for now
	}
	if me.Email != nil {
		a.Email = *me.Email
	}
	switch _, err := c.Key(ctx); {
	case errors.Is(err, blupcloud.ErrNoKey):
		a.Notes = "new"
	case err != nil:
		return Account{}, err
	case creds.NotesKey == "":
		a.Notes = "locked"
	default:
		a.Notes = "ready"
	}
	l, err := loadLink()
	if err != nil {
		return Account{}, err
	}
	a.Vault, a.Title = l.ID, l.Title
	// The live room opens (on screen) only with something to keep live.
	live.Want(a.Live && a.Vault != "" && a.Notes == "ready")
	return a, nil
}

// Login starts signing in: the browser opens on the approval page.
func (s *CloudService) Login(ctx context.Context) (LoginCode, error) {
	if err := prepare(); err != nil {
		return LoginCode{}, err
	}
	l, err := blupcloud.StartLogin(ctx)
	if err != nil {
		return LoginCode{}, err
	}
	// The address comes from the server and opens outside the app: only a
	// web page, never another scheme (intent:, file:, ...).
	if !safeURL(l.URL) {
		return LoginCode{}, fail("unsafeCloud", l.URL)
	}
	s.mu.Lock()
	s.login = l
	s.mu.Unlock()
	application.Mobile.OpenURL(l.URL)
	return LoginCode{Code: l.Code, URL: l.URL}, nil
}

// LoginWait waits for the approval and keeps the session.
func (s *CloudService) LoginWait(ctx context.Context) (Account, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.mu.Lock()
	l := s.login
	s.cancel = cancel
	s.mu.Unlock()
	if l == nil {
		return Account{}, context.Canceled
	}
	token, err := l.Wait(ctx)
	if err != nil {
		return Account{}, err
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		return Account{}, err
	}
	creds.BlupToken = token
	if err := creds.Save(); err != nil {
		return Account{}, err
	}
	return s.Account(ctx)
}

// LoginCancel gives up waiting for the browser.
func (s *CloudService) LoginCancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.login, s.cancel = nil, nil
}

// CreatePassword turns sync on for the account and returns the recovery
// key, shown once.
func (s *CloudService) CreatePassword(ctx context.Context, password string) (string, error) {
	// The cloud keeps the wrapped key and could try passwords offline: the
	// screen asks for 8 characters and this side refuses less, as the
	// Studio does.
	if utf8.RuneCountInString(password) < app.MinNotesPassword {
		return "", fail("shortPassword")
	}
	c, creds, err := client()
	if err != nil {
		return "", err
	}
	// The account first: the key is kept with it, and once the cloud has
	// the wrapped key a failure here would leave it with no owner.
	owner, err := app.NotesOwner(ctx, c)
	if err != nil {
		return "", err
	}
	master, recovery, w, err := cloudcrypt.NewKeys(password)
	if err != nil {
		return "", err
	}
	defer clear(master)
	if err := c.SaveKey(ctx, w); err != nil {
		return "", err
	}
	creds.SetNotesKey(base64.StdEncoding.EncodeToString(master), owner)
	return recovery, creds.Save()
}

// Unlock opens the notes here with the notes password or the recovery key.
func (s *CloudService) Unlock(ctx context.Context, secret string) error {
	c, creds, err := client()
	if err != nil {
		return err
	}
	owner, err := app.NotesOwner(ctx, c)
	if err != nil {
		return err
	}
	w, err := c.Key(ctx)
	if err != nil {
		return err
	}
	master, err := app.OpenNotesKey(w, secret)
	if errors.Is(err, cloudcrypt.ErrWrongPassword) {
		return fail("wrongSecret")
	}
	if err != nil {
		return err
	}
	defer clear(master)
	creds.SetNotesKey(base64.StdEncoding.EncodeToString(master), owner)
	return creds.Save()
}

func keys(creds config.Credentials) (*cloudcrypt.Keys, error) {
	master, err := base64.StdEncoding.DecodeString(creds.NotesKey)
	if err != nil {
		return nil, err
	}
	defer clear(master)
	return cloudcrypt.Derive(master)
}

// accountKeys are the notes keys of the account signed in with c: a key
// left here by another account is forgotten first (app.BindNotesKey), so
// it never seals a name or a note of this one.
func accountKeys(ctx context.Context, c *blupcloud.Client, creds *config.Credentials) (*cloudcrypt.Keys, error) {
	me, err := c.Me(ctx)
	if err != nil {
		return nil, err
	}
	if err := app.BindNotesKey(ctx, c, creds, me.ID); err != nil {
		return nil, err
	}
	if creds.NotesKey == "" {
		return nil, fail("notesLocked")
	}
	return keys(*creds)
}

// Vaults lists the account's vaults that have notes, by name.
func (s *CloudService) Vaults(ctx context.Context) ([]RemoteVault, error) {
	c, creds, err := client()
	if err != nil {
		return nil, err
	}
	k, err := accountKeys(ctx, c, &creds)
	if err != nil {
		return nil, err
	}
	list, err := c.Vaults(ctx)
	if err != nil {
		return nil, err
	}
	out := []RemoteVault{}
	for _, v := range list {
		if v.Head == 0 {
			continue
		}
		if title, _, err := k.OpenName(v.ID, v.Name); err == nil {
			out = append(out, RemoteVault{ID: v.ID, Title: title})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out, nil
}

// Link ties this phone's notes to a vault of the account, or to a new one
// named title. The first sync has no base, so it deletes nothing.
func (s *CloudService) Link(ctx context.Context, id, title string) error {
	c, creds, err := client()
	if err != nil {
		return err
	}
	// The id names a folder of the cache here (and is removed with it when
	// the vault is gone): only an id the cloud could have made.
	if id != "" && !blupcloud.ValidVaultID(id) {
		return fail("badVault")
	}
	k, err := accountKeys(ctx, c, &creds)
	if id == "" {
		if err != nil {
			return err
		}
		v, err := c.CreateNamedVault(ctx, k, title)
		if err != nil {
			return err
		}
		id = v.ID
	} else if err == nil {
		// A name sealed before it was bound to the vault's id is sealed
		// again now, once (best effort: it opens as it is anyway).
		_ = c.UpgradeVaultName(ctx, k, id)
	}
	return saveLink(link{id, title})
}

// Sync sends and receives the notes. Changed on both sides, the phone's
// version stays and the other one is kept as a copy, as in the Studio.
func (s *CloudService) Sync(ctx context.Context) (SyncResult, error) {
	return s.sync(ctx, false)
}

// SyncFresh is "Compare from scratch", offered only after a sync stopped
// for safety (stuckError) and confirmed by the person in a sheet: the
// phone forgets the last sync of the linked vault (the base of every file,
// the chain of versions and the cached old versions) and compares file by
// file with Blup Cloud, as a new phone. What differs becomes a conflict
// copy; no note is deleted on either side. It accepts the version the
// cloud shows now, so nothing calls it on its own.
func (s *CloudService) SyncFresh(ctx context.Context) (SyncResult, error) {
	return s.sync(ctx, true)
}

// stuckError: the sync stopped for safety, because Blup Cloud shows a
// version that does not continue the one synced here (notesync.Stuck).
// Stuck travels to the screen with the error (Wails puts the error's JSON
// in its cause), so it offers "Compare from scratch".
type stuckError struct {
	Stuck bool `json:"stuck"`
}

func (stuckError) Error() string { return tr("cloudStuck") }

func (s *CloudService) sync(ctx context.Context, fresh bool) (SyncResult, error) {
	syncLock.Lock()
	defer syncLock.Unlock()
	c, creds, err := client()
	if err != nil {
		return SyncResult{}, err
	}
	l, err := loadLink()
	if err != nil {
		return SyncResult{}, err
	}
	// Not linked (another vault opened since the account was read): no
	// vault to sync, and nothing to let go of as if it had been deleted.
	if l.ID == "" {
		return SyncResult{}, fail("notLinked")
	}
	me, err := c.Me(ctx)
	if err != nil {
		return SyncResult{}, err
	}
	// Only this account's key seals its notes: one left by another account
	// is forgotten, and the notes stay locked until its password is typed.
	if err := app.BindNotesKey(ctx, c, &creds, me.ID); err != nil {
		return SyncResult{}, err
	}
	if creds.NotesKey == "" {
		return SyncResult{}, fail("notesLocked")
	}
	k, err := keys(creds)
	if err != nil {
		return SyncResult{}, err
	}
	state, err := syncDir()
	if err != nil {
		return SyncResult{}, err
	}
	notes, err := vaultDir()
	if err != nil {
		return SyncResult{}, err
	}
	sy := &notesync.Syncer{
		Remote: &blupcloud.Remote{
			Client: c, Keys: k, Vault: l.ID,
			MaxFile: int64(me.Plan.Limits.MaxFileMB) << 20,
			Cache:   filepath.Join(state, "cache", l.ID),
		},
		Dir:       notes,
		StateFile: filepath.Join(state, "state.json"),
	}
	if fresh {
		if err := sy.Forget(); err != nil {
			return SyncResult{}, err
		}
	}
	r, err := sy.Sync(ctx, nil)
	if notesync.Stuck(err) {
		return SyncResult{}, stuckError{Stuck: true}
	}
	if errors.Is(err, blupcloud.ErrVaultMissing) && c.VaultGone(ctx, l.ID) {
		return SyncResult{Gone: true}, unlink(state, l.ID)
	}
	if err != nil {
		return SyncResult{}, err
	}
	// An open note, canvas or drawing reads itself again (or says so) when
	// the sync changed its file under it, as in the Studio.
	if changed := slices.Concat(r.Downloaded, r.DeletedLocal); len(changed) > 0 {
		notesChanged(strings.Join(changed, "\n"))
	}
	// Events may have come with the notes: their alarms go on the list.
	// Nothing came, nothing to do: the list is as the phone left it.
	if len(r.Downloaded)+len(r.DeletedLocal) > 0 {
		if err := writeReminders(); err != nil {
			return SyncResult{}, err
		}
	}
	return SyncResult{
		Sent: len(r.Uploaded) + len(r.Deleted), Received: len(r.Downloaded) + len(r.DeletedLocal),
		Conflicts: r.ConflictCopies, At: time.Now().UnixMilli(),
	}, nil
}

// unlink lets go of the open vault's Cloud vault: the link, the sync state
// and the cache go; the notes stay. Only the person links it again.
func unlink(state, id string) error {
	if err := os.Remove(filepath.Join(state, "vault")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(filepath.Join(state, "state.json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.RemoveAll(filepath.Join(state, "cache", id))
}

// SignOut ends the session here and forgets the key. The notes stay on
// the phone.
func (s *CloudService) SignOut(ctx context.Context) error {
	c, creds, err := client()
	if err != nil {
		return err
	}
	_ = c.SignOut(ctx)
	live.Want(false)
	creds.BlupToken = ""
	creds.ForgetNotesKey()
	return creds.Save()
}
