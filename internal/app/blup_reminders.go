package app

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
)

// The calendar's reminders by e-mail through Blup Cloud (Max and Ultra).
// Signed out, both answer blupcloud.ErrSignedOut.

// ReminderEmails is the month's use of reminder e-mails.
func ReminderEmails(ctx context.Context) (blupcloud.ReminderUse, error) {
	c, _, err := blupClient()
	if err != nil {
		return blupcloud.ReminderUse{}, err
	}
	return c.ReminderEmails(ctx)
}

// PushReminders replaces the reminders waiting in the cloud.
func PushReminders(ctx context.Context, list []blupcloud.Reminder) (blupcloud.ReminderUse, error) {
	c, _, err := blupClient()
	if err != nil {
		return blupcloud.ReminderUse{}, err
	}
	return c.SetReminders(ctx, list)
}

// A sign out that could not reach Blup Cloud (offline) leaves its session
// here, in a file only this user reads, to cancel its reminders and end it
// later: at the next start or sign in.
func signOutPendingFile() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "signout-pending"), nil
}

func saveSignOutPending(token string) error {
	f, err := signOutPendingFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f), 0o700); err != nil {
		return err
	}
	tmp := f + ".tmp"
	if err := os.WriteFile(tmp, []byte(token), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f)
}

// RetrySignOut finishes a sign out left half done offline: it cancels that
// session's reminders and ends it. done is true when it did now (nothing
// pending, or still offline: false).
func RetrySignOut(ctx context.Context) (done bool) {
	f, err := signOutPendingFile()
	if err != nil {
		return false
	}
	b, err := os.ReadFile(f)
	if err != nil {
		return false
	}
	if token := strings.TrimSpace(string(b)); token != "" {
		if finishSignOut(ctx, blupcloud.New(token)) != nil {
			return false
		}
	}
	return os.Remove(f) == nil
}

// reminderKey is a secret of this computer that makes reminder ids opaque.
// A plain hash of "folder|note|day|remind" could be checked against
// guesses ("Agenda/Dentist.md") by whoever reads the ids in the cloud; an
// HMAC under a key that never leaves the computer cannot.
var reminderKey struct {
	once sync.Once
	key  []byte
}

func loadReminderKey() []byte {
	reminderKey.once.Do(func() {
		dir, err := config.Dir()
		if err == nil {
			reminderKey.key = readOrMakeKey(filepath.Join(dir, "reminder-key"))
		}
		if len(reminderKey.key) != 32 {
			// No config folder: a key for this run only (ids change on the
			// next start, which at worst sends a reminder twice).
			reminderKey.key = make([]byte, 32)
			rand.Read(reminderKey.key)
		}
	})
	return reminderKey.key
}

// readOrMakeKey reads a 32-byte key at p, creating it (0600) the first time.
func readOrMakeKey(p string) []byte {
	if b, err := os.ReadFile(p); err == nil && len(b) == 32 {
		return b
	}
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return nil
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		// Made meanwhile (or damaged): use what is there when it is whole.
		if b, err := os.ReadFile(p); err == nil && len(b) == 32 {
			return b
		}
		return nil
	}
	if err != nil {
		return nil
	}
	_, err = f.Write(k)
	if cerr := f.Close(); err != nil || cerr != nil {
		os.Remove(p)
		return nil
	}
	return k
}

// ReminderID is the opaque id a reminder goes to Blup Cloud under: 32 hex
// digits that say nothing about the event.
func ReminderID(key string) string {
	m := hmac.New(sha256.New, loadReminderKey())
	m.Write([]byte(key))
	return hex.EncodeToString(m.Sum(nil)[:16])
}
