package app

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/notesync"
)

// Blup Cloud for Blup Studio: signing in through the browser, the notes
// password (it opens the key that encrypts the notes on this computer) and
// the notes of a site synced through the cloud.

func init() {
	registerErrors(
		knownError{blupcloud.ErrSignedOut, "blup.err_signed_out"},
		knownError{blupcloud.ErrNoKey, "blup.err_no_key"},
		knownError{blupcloud.ErrKeyExists, "blup.err_key_exists"},
		knownError{blupcloud.ErrQuota, "blup.err_quota"},
		knownError{blupcloud.ErrDevicesLimit, "blup.err_devices"},
		knownError{blupcloud.ErrSyncBusy, "blup.err_sync_busy"},
		knownError{blupcloud.ErrVaultMissing, "blup.err_vault_missing"},
		knownError{blupcloud.ErrCantDelete, "blup.err_cant_delete"},
		knownError{blupcloud.ErrTicketExpired, "blup.err_ticket_expired"},
		knownError{blupcloud.ErrDenied, "blup.err_denied"},
		knownError{blupcloud.ErrExpired, "blup.err_expired"},
		knownError{cloudcrypt.ErrWrongPassword, "blup.err_password"},
		knownError{errNotesLocked, "blup.err_locked"},
		knownError{ErrNotesPasswordShort, "blup.err_password_short"},
		knownError{notesync.ErrRolledBack, "blup.err_rolled_back"},
		knownError{notesync.ErrForked, "blup.err_forked"},
	)
}

// MinNotesPassword is the shortest notes password accepted, in characters.
// The cloud keeps the wrapped key, so it could try passwords offline: the
// screens ask for this too, and this side refuses it whatever calls it.
const MinNotesPassword = 8

// ErrNotesPasswordShort: the notes password has fewer than MinNotesPassword
// characters.
var ErrNotesPasswordShort = errors.New("app: the notes password is too short")

// errNotesLocked: the notes password was not typed on this computer yet.
var errNotesLocked = errors.New("app: the notes are locked on this computer")

// RemoteBlup is NoteSync.Remote for Blup Cloud.
const RemoteBlup = config.SyncBlup

// BlupAccount is the signed in account as Blup Studio shows it.
type BlupAccount struct {
	SignedIn bool `json:"signedIn"`
	// ID is the account's id on Blup Cloud: not a secret, the person tells
	// it when asking for help.
	ID    string `json:"id"`
	Email string `json:"email"`
	Plan  string `json:"plan"`  // "free", "pro", "max", "ultra"
	Label string `json:"label"` // "Grátis", "Pro"...
	// Live: the plan syncs each note as it is saved and hears of the other
	// computers at once; without it, sync is by button.
	Live bool `json:"live"`
	// HistoryDays: how long old versions are kept (0: no history).
	HistoryDays  int   `json:"historyDays"`
	MaxFileMB    int   `json:"maxFileMB"`
	StorageMB    int   `json:"storageMB"`
	StorageBytes int64 `json:"storageBytes"`
	// StorageLimit is the plan's storage in bytes.
	StorageLimit int64 `json:"storageLimit"`
	// After a plan with less storage, while the use is above it: until
	// GraceUntil (ISO 8601) the previous plan's GraceLimit still counts.
	// Empty and 0 otherwise.
	GraceUntil string `json:"graceUntil"`
	GraceLimit int64  `json:"graceLimit"`
	// HistoryBytes is what the old versions of all the vaults take; nil
	// when the cloud can't tell right now (shown as unavailable).
	HistoryBytes *int64 `json:"historyBytes"`
	// Sites and computers used, and how many the plan allows.
	Sites      int `json:"sites"`
	SitesMax   int `json:"sitesMax"`
	Devices    int `json:"devices"`
	DevicesMax int `json:"devicesMax"`
	// AI: the plan includes AI through Blup Cloud (BlupAIUsage tells how
	// much of it was used).
	AI bool `json:"ai"`
	// Notes: "new" (no notes password yet), "locked" (created on another
	// computer, not typed here) or "ready".
	Notes string `json:"notes"`
}

func blupClient() (*blupcloud.Client, config.Credentials, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, creds, err
	}
	if creds.BlupToken == "" {
		return nil, creds, blupcloud.ErrSignedOut
	}
	return blupcloud.New(creds.BlupToken), creds, nil
}

// LoadBlupAccount reads the account, its plan and the notes password state.
// Signed out is not an error: SignedIn is false.
func LoadBlupAccount(ctx context.Context) (BlupAccount, error) {
	c, creds, err := blupClient()
	if errors.Is(err, blupcloud.ErrSignedOut) {
		return BlupAccount{}, nil
	}
	if err != nil {
		return BlupAccount{}, err
	}
	me, err := c.Me(ctx)
	if errors.Is(err, blupcloud.ErrSignedOut) {
		// Ended in the panel, or expired: forget it.
		creds.BlupToken = ""
		return BlupAccount{}, creds.Save()
	}
	if err != nil {
		return BlupAccount{}, errors.New(describe(err))
	}
	if err := bindNotesKey(ctx, c, &creds, me.ID); err != nil {
		return BlupAccount{}, errors.New(describe(err))
	}
	a := BlupAccount{
		SignedIn: true, ID: me.ID, Plan: me.Plan.Name, Label: me.Plan.Label,
		Live: me.Plan.Limits.Live, HistoryDays: me.Plan.Limits.HistoryDays, MaxFileMB: me.Plan.Limits.MaxFileMB,
		StorageMB: me.Plan.Limits.StorageMB, StorageBytes: me.Usage.StorageBytes,
		Sites: me.Usage.Sites, SitesMax: me.Plan.Limits.Sites,
		Devices: me.Usage.Devices, DevicesMax: me.Plan.Limits.Devices,
		AI: me.Plan.Limits.AIUsage > 0,
	}
	if me.Email != nil {
		a.Email = *me.Email
	}
	u := me.Usage
	a.StorageLimit, a.HistoryBytes = u.StorageLimitBytes, u.HistoryBytes
	if a.StorageLimit == 0 { // an older cloud
		a.StorageLimit = int64(a.StorageMB) << 20
	}
	if u.StorageGraceUntil != nil && u.StorageGraceLimitBytes != nil {
		a.GraceUntil, a.GraceLimit = *u.StorageGraceUntil, *u.StorageGraceLimitBytes
	}
	switch _, err := c.Key(ctx); {
	case errors.Is(err, blupcloud.ErrNoKey):
		a.Notes = "new"
	case err != nil:
		return BlupAccount{}, errors.New(describe(err))
	case creds.NotesKey == "":
		a.Notes = "locked"
	default:
		a.Notes = "ready"
	}
	return a, nil
}

// StartBlupLogin asks Blup Cloud for a sign in code; the person approves it
// in the browser at Login.URL.
func StartBlupLogin(ctx context.Context) (*blupcloud.Login, error) {
	l, err := blupcloud.StartLogin(ctx)
	if err != nil {
		return nil, errors.New(describe(err))
	}
	return l, nil
}

// FinishBlupLogin waits for the approval and saves the session.
func FinishBlupLogin(ctx context.Context, l *blupcloud.Login) (BlupAccount, error) {
	token, err := l.Wait(ctx)
	if err != nil {
		return BlupAccount{}, errors.New(describe(err))
	}
	// A sign out left half done offline goes first: its empty list must
	// not land after the reminders of this new session.
	RetrySignOut(ctx)
	creds, err := config.LoadCredentials()
	if err != nil {
		return BlupAccount{}, err
	}
	creds.BlupToken = token
	if err := creds.Save(); err != nil {
		return BlupAccount{}, err
	}
	return LoadBlupAccount(ctx)
}

// BlupSignOut ends this computer's session and forgets the notes key here
// (the notes password opens it again). Before that, the reminders waiting
// in the cloud are cancelled (an empty list), so no e-mail comes after
// leaving. Offline, the session is forgotten here all the same and what
// is left (cancelling, ending the session) is kept to try again at the
// next start or sign in (RetrySignOut); pending is then true.
func BlupSignOut(ctx context.Context) (pending bool, err error) {
	c, creds, err := blupClient()
	if errors.Is(err, blupcloud.ErrSignedOut) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := finishSignOut(ctx, c); err != nil {
		if perr := saveSignOutPending(creds.BlupToken); perr != nil {
			return false, perr
		}
		pending = true
	}
	creds.BlupToken = ""
	creds.ForgetNotesKey()
	return pending, creds.Save()
}

// finishSignOut cancels the session's reminders, then ends the session. A
// session already ended (elsewhere, or expired) is done; any other error
// (offline) means trying again later.
func finishSignOut(ctx context.Context, c *blupcloud.Client) error {
	if _, err := c.SetReminders(ctx, nil); err != nil && !errors.Is(err, blupcloud.ErrSignedOut) {
		return err
	}
	if err := c.SignOut(ctx); err != nil && !errors.Is(err, blupcloud.ErrSignedOut) {
		return err
	}
	return nil
}

// CreateNotesPassword turns sync on for the account: it creates the key
// that encrypts the notes, wraps it with the password and returns the
// recovery key, to be shown once.
func CreateNotesPassword(ctx context.Context, password string) (string, error) {
	if utf8.RuneCountInString(password) < MinNotesPassword {
		return "", errors.New(describe(ErrNotesPasswordShort))
	}
	c, creds, err := blupClient()
	if err != nil {
		return "", errors.New(describe(err))
	}
	// The account first: the key is kept with it, and once the cloud has
	// the wrapped key a failure here would leave it with no owner.
	owner, err := NotesOwner(ctx, c)
	if err != nil {
		return "", errors.New(describe(err))
	}
	master, recovery, w, err := cloudcrypt.NewKeys(password)
	if err != nil {
		return "", err
	}
	defer clear(master)
	if err := c.SaveKey(ctx, w); err != nil {
		return "", errors.New(describe(err))
	}
	creds.SetNotesKey(base64.StdEncoding.EncodeToString(master), owner)
	return recovery, creds.Save()
}

// UnlockNotes opens the notes on this computer with the notes password or
// the recovery key.
func UnlockNotes(ctx context.Context, secret string) error {
	c, creds, err := blupClient()
	if err != nil {
		return errors.New(describe(err))
	}
	owner, err := NotesOwner(ctx, c)
	if err != nil {
		return errors.New(describe(err))
	}
	w, err := c.Key(ctx)
	if err != nil {
		return errors.New(describe(err))
	}
	master, err := OpenNotesKey(w, secret)
	if err != nil {
		return errors.New(describe(err))
	}
	defer clear(master)
	creds.SetNotesKey(base64.StdEncoding.EncodeToString(master), owner)
	return creds.Save()
}

func bindNotesKey(ctx context.Context, c *blupcloud.Client, creds *config.Credentials, account string) error {
	return BindNotesKey(ctx, c, creds, account)
}

// BindNotesKey forgets a notes key of another account than the one signed
// in with c (account, from /me), and adopts a key saved without an account
// (config.Credentials.BindNotesKey), saving when it changed. A key without
// an account is adopted only if it can be this account's: one that opens
// none of the account's vault names, or kept while the account has no
// notes key at all, belongs to someone else and is forgotten (the notes
// are then locked until this account's password is typed). The phone
// shares it.
func BindNotesKey(ctx context.Context, c *blupcloud.Client, creds *config.Credentials, account string) error {
	if account != "" && creds.NotesKey != "" && creds.NotesAccount == "" {
		fits, err := keyFitsAccount(ctx, c, creds.NotesKey)
		if err != nil {
			return err
		}
		if !fits {
			creds.ForgetNotesKey()
			return creds.Save()
		}
	}
	if !creds.BindNotesKey(account) {
		return nil
	}
	return creds.Save()
}

// keyFitsAccount reports whether the notes key (base64) can be the one of
// the account signed in with c. It is false only when it surely is not:
// not a key, the account has no notes key, or it has vaults and the key
// opens none of their names. With no vault there is nothing to tell by.
func keyFitsAccount(ctx context.Context, c *blupcloud.Client, key string) (bool, error) {
	master, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return false, nil
	}
	keys, err := cloudcrypt.Derive(master)
	clear(master)
	if err != nil {
		return false, nil
	}
	if _, err := c.Key(ctx); errors.Is(err, blupcloud.ErrNoKey) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	vaults, err := c.Vaults(ctx)
	if err != nil {
		return false, err
	}
	for _, v := range vaults {
		if _, _, err := keys.OpenName(v.ID, v.Name); err == nil {
			return true, nil
		}
	}
	return len(vaults) == 0, nil
}

// NotesOwner is the id of the account signed in with c, kept with the
// notes key. It needs the cloud, as creating or opening the key does: a
// key is never kept without its account.
func NotesOwner(ctx context.Context, c *blupcloud.Client) (string, error) {
	me, err := c.Me(ctx)
	if err != nil {
		return "", err
	}
	if me.ID == "" {
		return "", blupcloud.ErrBadAnswer
	}
	return me.ID, nil
}

// accountKeys are the subkeys of the notes key, checked against the
// account signed in with c first (BindNotesKey): a key left here by
// another account never seals anything of this one.
func accountKeys(ctx context.Context, c *blupcloud.Client, creds *config.Credentials) (*cloudcrypt.Keys, error) {
	me, err := c.Me(ctx)
	if err != nil {
		return nil, err
	}
	if err := bindNotesKey(ctx, c, creds, me.ID); err != nil {
		return nil, err
	}
	if creds.NotesKey == "" {
		return nil, errNotesLocked
	}
	master, err := base64.StdEncoding.DecodeString(creds.NotesKey)
	if err != nil {
		return nil, errNotesLocked
	}
	defer clear(master)
	keys, err := cloudcrypt.Derive(master)
	if err != nil {
		return nil, errNotesLocked
	}
	return keys, nil
}

// OpenNotesKey opens the master key with what was typed: the notes
// password exactly as typed, then without the spaces around it (pasted
// with a line break, or typed on a screen that trimmed it), then as the
// recovery key.
func OpenNotesKey(w cloudcrypt.Wrapped, secret string) ([]byte, error) {
	master, err := cloudcrypt.Unlock(w, secret)
	if t := strings.TrimSpace(secret); errors.Is(err, cloudcrypt.ErrWrongPassword) && t != secret && t != "" {
		master, err = cloudcrypt.Unlock(w, t)
	}
	if errors.Is(err, cloudcrypt.ErrWrongPassword) {
		if m, rerr := cloudcrypt.Recover(w, secret); rerr == nil {
			master, err = m, nil
		}
	}
	return master, err
}

// SetSyncRemote picks where the site's notes are synced: RemoteBlup or
// GitHub (any other value). Changing it starts over (the next sync compares
// without a base, so nothing is deleted on either side).
func SetSyncRemote(site *config.Site, remote config.SyncRemote) error {
	if remote != RemoteBlup {
		remote = ""
	}
	if site.Sync.Remote == remote {
		return nil
	}
	site.Sync.Remote = remote
	return site.Save()
}

// blupRemote is the vault of a site synced through Blup Cloud; it is
// created the first time (named after the site, encrypted).
func blupRemote(ctx context.Context, site *config.Site) (*blupcloud.Remote, error) {
	c, creds, err := blupClient()
	if err != nil {
		return nil, errors.New(describe(err))
	}
	// The key must be this account's: one left by another account is
	// forgotten here, and the notes are locked.
	me, err := c.Me(ctx)
	if err != nil {
		return nil, errors.New(describe(err))
	}
	if err := bindNotesKey(ctx, c, &creds, me.ID); err != nil {
		return nil, errors.New(describe(err))
	}
	if creds.NotesKey == "" {
		return nil, errors.New(describe(errNotesLocked))
	}
	master, err := base64.StdEncoding.DecodeString(creds.NotesKey)
	if err != nil {
		return nil, errors.New(describe(errNotesLocked))
	}
	keys, err := cloudcrypt.Derive(master)
	clear(master)
	if err != nil {
		return nil, errors.New(describe(errNotesLocked))
	}
	if site.Sync.Vault != "" && !blupcloud.ValidVaultID(site.Sync.Vault) {
		return nil, blupcloud.ErrBadVault // it names a folder here
	}
	if site.Sync.Vault == "" {
		v, err := c.CreateNamedVault(ctx, keys, site.Title)
		if err != nil {
			return nil, errors.New(describe(err))
		}
		site.Sync.Vault = v.ID
		if err := site.Save(); err != nil {
			return nil, err
		}
	}
	return &blupcloud.Remote{
		Client: c, Keys: keys, Vault: site.Sync.Vault,
		MaxFile: int64(me.Plan.Limits.MaxFileMB) << 20,
		Cache:   filepath.Join(site.Dir(), ".blup", "cloud-history", site.Sync.Vault),
	}, nil
}

// blupSyncer is the Syncer of a site synced through Blup Cloud.
func blupSyncer(ctx context.Context, site *config.Site) (*notesync.Syncer, error) {
	r, err := blupRemote(ctx, site)
	if err != nil {
		return nil, err
	}
	return &notesync.Syncer{
		Remote:    r,
		Dir:       site.ContentDir(),
		StateFile: filepath.Join(site.Dir(), ".blup", "sync-state.json"),
	}, nil
}

// PurgeHistory deletes the old versions of the site's notes on Blup Cloud
// (the current version stays) and says how much it freed and the
// account's use after it.
func PurgeHistory(ctx context.Context, site *config.Site) (blupcloud.HistoryPurge, error) {
	if site.Sync.Remote != RemoteBlup || site.Sync.Vault == "" {
		return blupcloud.HistoryPurge{}, errors.New(i18n.T("blup.history_not_synced"))
	}
	c, _, err := blupClient()
	if err != nil {
		return blupcloud.HistoryPurge{}, errors.New(describe(err))
	}
	r, err := c.PurgeHistory(ctx, site.Sync.Vault)
	if err != nil {
		return r, errors.New(describe(err))
	}
	return r, nil
}

// ErrSyncStuck: the sync of the site's notes stopped for safety, because
// Blup Cloud shows a version that does not continue the one this computer
// synced (notesync.Stuck). Nothing was deleted, and it stays stopped until
// the person chooses to compare from scratch (ResetSync).
var ErrSyncStuck = errors.New("sync stuck")

// SyncStuckError says it in the person's words; errors.Is finds
// ErrSyncStuck. Stuck travels to the window with the error (Wails puts
// the error's JSON in its cause), so the screen offers "Compare from
// scratch" without reading the message.
type SyncStuckError struct {
	Stuck bool `json:"stuck"`
}

func (SyncStuckError) Error() string        { return i18n.T("blup.sync_stuck") }
func (SyncStuckError) Is(target error) bool { return target == ErrSyncStuck }

// syncFailed is the error of a sync as the person reads it.
func syncFailed(err error) error {
	if notesync.Stuck(err) {
		return SyncStuckError{Stuck: true}
	}
	return errors.New(describe(err))
}

// ResetSync is "Compare from scratch": this computer forgets what it knew
// of the site's last sync (the base of every file, the chain of versions
// and the cached old versions) and syncs as a new computer. No note is
// deleted on either side: what differs is kept twice (conflict copy), what
// only one side has goes to the other. It accepts the version Blup Cloud
// shows now, so the screens only offer it after a stuck sync and after the
// person confirms; nothing calls it on its own.
func ResetSync(ctx context.Context, site *config.Site) (SyncInfo, error) {
	return syncSiteWith(ctx, site, true, true)
}

// ErrVaultGone: the site's notes were deleted from Blup Cloud on another
// device; this one let go of them (the notes stay here).
var ErrVaultGone = errors.New("vault gone")

// VaultGoneError says it in the person's words; errors.Is finds ErrVaultGone.
type VaultGoneError struct{}

func (VaultGoneError) Error() string        { return i18n.T("sync.vault_gone") }
func (VaultGoneError) Is(target error) bool { return target == ErrVaultGone }

// unlinkGoneVault lets go of the site's Blup Cloud vault when a sync found
// it missing and the account's list confirms it was deleted: the notes stay,
// syncing stops, and only the person sends them again (the Sync button makes
// a new vault). Otherwise, nil: the error stays what it was.
func unlinkGoneVault(ctx context.Context, site *config.Site, err error) error {
	if site.Sync.Remote != RemoteBlup || site.Sync.Vault == "" || !errors.Is(err, blupcloud.ErrVaultMissing) {
		return nil
	}
	c, _, cerr := blupClient()
	if cerr != nil || !c.VaultGone(ctx, site.Sync.Vault) {
		return nil
	}
	old := site.Sync.Vault
	site.Sync.Vault = ""
	if err := site.Save(); err != nil {
		return err
	}
	// Starting over: the next sync compares without a base, so nothing is
	// deleted on either side.
	os.Remove(filepath.Join(site.Dir(), ".blup", "sync-state.json"))
	os.RemoveAll(filepath.Join(site.Dir(), ".blup", "cloud-history", old))
	return VaultGoneError{}
}

// RenameSite gives a site or a vault of notes a new name. With its notes on
// Blup Cloud, the name changes there too, so the other devices see it; it
// changes here anyway, and cloudOK says whether it did there.
func RenameSite(ctx context.Context, dir, title string) (cloudOK bool, err error) {
	title = strings.Join(strings.Fields(title), " ")
	if title == "" || utf8.RuneCountInString(title) > 120 {
		return false, errors.New(i18nInvalidName())
	}
	site, err := config.LoadSite(dir)
	if err != nil {
		return false, err
	}
	if err := renameNotesFolder(site, title); err != nil {
		return false, err
	}
	site.Title = title
	if err := site.Save(); err != nil {
		return false, err
	}
	if site.Sync.Remote != config.SyncBlup || site.Sync.Vault == "" {
		return true, nil
	}
	return renameBlupVault(ctx, site.Sync.Vault, title) == nil, nil
}

// renameNotesFolder gives the notes folder the new name too, when it is a
// folder Blup made (in Documents/Blup or Documents/Blup/Vaults). A folder
// the person picked keeps its name: other apps may point at it.
func renameNotesFolder(site *config.Site, title string) error {
	if site.Folder != "" || site.Vault == "" {
		return nil
	}
	old := filepath.Clean(site.Vault)
	parent := filepath.Dir(old)
	blup := filepath.Dir(DefaultNotesDir("x"))
	vaults := filepath.Dir(parent) == blup && slices.Contains([]string{"Vaults", "Cofres", "Bóvedas"}, filepath.Base(parent))
	if parent != blup && !vaults {
		return nil
	}
	to := filepath.Join(parent, folderName(title))
	if to == old {
		return nil
	}
	// Another folder with that name (not this one in other letter case).
	if a, err := os.Stat(to); err == nil {
		if b, err := os.Stat(old); err != nil || !os.SameFile(a, b) {
			return errors.New(i18n.T("rename.folder_taken", filepath.Base(to)))
		}
	}
	if err := os.Rename(old, to); err != nil {
		return err
	}
	site.Vault = to
	return nil
}

// renameBlupVault seals title with the notes key and names the vault with it.
func renameBlupVault(ctx context.Context, id, title string) error {
	c, creds, err := blupClient()
	if err != nil {
		return err
	}
	keys, err := accountKeys(ctx, c, &creds)
	if err != nil {
		return err
	}
	return c.RenameSealed(ctx, keys, id, title)
}
