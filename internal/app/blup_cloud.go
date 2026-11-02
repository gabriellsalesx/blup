package app

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"

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
		knownError{blupcloud.ErrQuota, "blup.err_quota"},
		knownError{blupcloud.ErrDevicesLimit, "blup.err_devices"},
		knownError{blupcloud.ErrSyncBusy, "blup.err_sync_busy"},
		knownError{blupcloud.ErrVaultMissing, "blup.err_vault_missing"},
		knownError{blupcloud.ErrTicketExpired, "blup.err_ticket_expired"},
		knownError{blupcloud.ErrDenied, "blup.err_denied"},
		knownError{blupcloud.ErrExpired, "blup.err_expired"},
		knownError{cloudcrypt.ErrWrongPassword, "blup.err_password"},
		knownError{errNotesLocked, "blup.err_locked"},
		knownError{notesync.ErrRolledBack, "blup.err_rolled_back"},
	)
}

// errNotesLocked: the notes password was not typed on this computer yet.
var errNotesLocked = errors.New("app: the notes are locked on this computer")

// RemoteBlup is NoteSync.Remote for Blup Cloud.
const RemoteBlup = "blup"

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
	creds.BlupToken, creds.NotesKey = "", ""
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
	c, creds, err := blupClient()
	if err != nil {
		return "", errors.New(describe(err))
	}
	master, recovery, w, err := cloudcrypt.NewKeys(password)
	if err != nil {
		return "", err
	}
	if err := c.SaveKey(ctx, w); err != nil {
		return "", errors.New(describe(err))
	}
	creds.NotesKey = base64.StdEncoding.EncodeToString(master)
	return recovery, creds.Save()
}

// UnlockNotes opens the notes on this computer with the notes password or
// the recovery key.
func UnlockNotes(ctx context.Context, secret string) error {
	c, creds, err := blupClient()
	if err != nil {
		return errors.New(describe(err))
	}
	w, err := c.Key(ctx)
	if err != nil {
		return errors.New(describe(err))
	}
	master, err := cloudcrypt.Unlock(w, secret)
	if errors.Is(err, cloudcrypt.ErrWrongPassword) {
		if m, rerr := cloudcrypt.Recover(w, secret); rerr == nil {
			master, err = m, nil
		}
	}
	if err != nil {
		return errors.New(describe(err))
	}
	creds.NotesKey = base64.StdEncoding.EncodeToString(master)
	return creds.Save()
}

// SetSyncRemote picks where the site's notes are synced: "blup" or
// "github". Changing it starts over (the next sync compares without a
// base, so nothing is deleted on either side).
func SetSyncRemote(site *config.Site, remote string) error {
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
	if creds.NotesKey == "" {
		return nil, errors.New(describe(errNotesLocked))
	}
	master, err := base64.StdEncoding.DecodeString(creds.NotesKey)
	if err != nil {
		return nil, errors.New(describe(errNotesLocked))
	}
	keys, err := cloudcrypt.Derive(master)
	if err != nil {
		return nil, errors.New(describe(errNotesLocked))
	}
	me, err := c.Me(ctx)
	if err != nil {
		return nil, errors.New(describe(err))
	}
	if site.Sync.Vault != "" && !blupcloud.ValidVaultID(site.Sync.Vault) {
		return nil, blupcloud.ErrBadVault // it names a folder here
	}
	if site.Sync.Vault == "" {
		name, err := keys.SealName(site.Title)
		if err != nil {
			return nil, err
		}
		v, err := c.CreateVault(ctx, name)
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
