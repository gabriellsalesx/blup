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
