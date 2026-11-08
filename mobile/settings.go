package main

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

// SettingsService keeps what is set on this phone only: it never syncs.
type SettingsService struct{}

// settings is the file. The PIN is kept as a slow hash of it. It covers the
// app's door, not the notes: they stay as plain files in the app's folder.
type settings struct {
	Quiet bool `json:"quiet,omitempty"` // no notifications
	// Reports: error reports to Blup ("" = not asked yet, "on" or "off"; errors.go).
	Reports string `json:"reports,omitempty"`
	Salt    []byte `json:"salt,omitempty"`
	Hash    []byte `json:"hash,omitempty"`
	Fails   int    `json:"fails,omitempty"`
	Until   int64  `json:"until,omitempty"` // no attempts before this, in ms
	// Tags made before any note has them.
	Tags []string `json:"tags,omitempty"`
	// Lang is the app's language, picked in Account: pt, en or es; empty,
	// the screen goes by the phone's.
	Lang string `json:"lang,omitempty"`
}

// PinResult says how a PIN went; Wait is the seconds to try again, when too
// many went wrong.
type PinResult struct {
	OK   bool `json:"ok"`
	Wait int  `json:"wait"`
}

const (
	pinRounds = 200_000
	freeTries = 5
	maxWait   = 15 * time.Minute
)

var (
	settingsLock sync.Mutex
	fourDigits   = regexp.MustCompile(`^[0-9]{4}$`)
)

func settingsFile() (string, error) {
	root, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "settings.json"), nil
}

func readSettings() (settings, error) {
	var s settings
	file, err := settingsFile()
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	// A file left broken must not keep the notes from opening.
	if json.Unmarshal(data, &s) != nil {
		return settings{}, nil
	}
	return s, nil
}

func writeSettings(s settings) error {
	file, err := settingsFile()
	if err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return writeSafe(file, data)
}

// Notifications tells whether the reminders reach the phone.
func (c *SettingsService) Notifications() (bool, error) {
	settingsLock.Lock()
	defer settingsLock.Unlock()
	s, err := readSettings()
	return !s.Quiet, err
}

// SetNotifications turns the reminders on or off. Turning them on lets the
// app ask Android's permission once more (see MainActivity).
func (c *SettingsService) SetNotifications(on bool) error {
	settingsLock.Lock()
	s, err := readSettings()
	if err == nil {
		s.Quiet = !on
		err = writeSettings(s)
	}
	settingsLock.Unlock()
	if err != nil {
		return err
	}
	if on {
		root, err := dataDir()
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, "ask-notifications"), nil, 0o600); err != nil {
			return err
		}
	}
	return writeReminders()
}

// Language is the language picked in Account, or "" before any pick.
func (c *SettingsService) Language() (string, error) {
	settingsLock.Lock()
	defer settingsLock.Unlock()
	s, err := readSettings()
	if !validLang(s.Lang) {
		return "", err
	}
	return s.Lang, err
}

// UseLanguage is the language the screen opened in (the picked one or the
// phone's): the messages from here and the reminders follow it.
func (c *SettingsService) UseLanguage(lang string) error {
	if !validLang(lang) {
		return fail("badLang", lang)
	}
	changed := appLang() != lang
	useLang(lang)
	if !changed {
		return nil
	}
	return writeReminders()
}

// SetLanguage keeps the language picked in Account and starts using it.
func (c *SettingsService) SetLanguage(lang string) error {
	if !validLang(lang) {
		return fail("badLang", lang)
	}
	settingsLock.Lock()
	s, err := readSettings()
	if err == nil {
		s.Lang = lang
		err = writeSettings(s)
	}
	settingsLock.Unlock()
	if err != nil {
		return err
	}
	return c.UseLanguage(lang)
}

// Tags lists the tags made by hand.
func (c *SettingsService) Tags() ([]string, error) {
	settingsLock.Lock()
	defer settingsLock.Unlock()
	s, err := readSettings()
	return s.Tags, err
}

// AddTag keeps a new tag, to pick in the notes.
func (c *SettingsService) AddTag(name string) error {
	name = strings.Join(strings.Fields(strings.TrimPrefix(strings.TrimSpace(name), "#")), "-")
	if name == "" {
		return fail("tagName")
	}
	settingsLock.Lock()
	defer settingsLock.Unlock()
	s, err := readSettings()
	if err != nil {
		return err
	}
	if !slices.Contains(s.Tags, name) {
		s.Tags = append(s.Tags, name)
	}
	return writeSettings(s)
}

// Locked tells whether the app asks for a PIN.
func (c *SettingsService) Locked() (bool, error) {
	settingsLock.Lock()
	defer settingsLock.Unlock()
	s, err := readSettings()
	return len(s.Hash) > 0, err
}

func hashPin(pin string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, pin, salt, pinRounds, 32)
}

// SetPin locks the app with a new PIN of four digits.
func (c *SettingsService) SetPin(pin string) error {
	if !fourDigits.MatchString(pin) {
		return fail("pinDigits")
	}
	salt := make([]byte, 16)
	rand.Read(salt)
	hash, err := hashPin(pin, salt)
	if err != nil {
		return err
	}
	settingsLock.Lock()
	defer settingsLock.Unlock()
	s, err := readSettings()
	if err != nil {
		return err
	}
	s.Salt, s.Hash, s.Fails, s.Until = salt, hash, 0, 0
	return writeSettings(s)
}

// check tries the PIN. After freeTries wrong ones each attempt has to wait,
// twice as long every time, and closing the app does not reset it.
func check(s *settings, pin string) (PinResult, error) {
	now := time.Now()
	if left := time.UnixMilli(s.Until).Sub(now); left > 0 {
		return PinResult{Wait: int(left.Seconds()) + 1}, nil
	}
	hash, err := hashPin(pin, s.Salt)
	if err != nil {
		return PinResult{}, err
	}
	if subtle.ConstantTimeCompare(hash, s.Hash) == 1 {
		s.Fails, s.Until = 0, 0
		return PinResult{OK: true}, nil
	}
	s.Fails++
	if s.Fails < freeTries {
		return PinResult{}, nil
	}
	wait := min(30*time.Second<<min(s.Fails-freeTries, 10), maxWait)
	s.Until = now.Add(wait).UnixMilli()
	return PinResult{Wait: int(wait.Seconds())}, nil
}

// CheckPin opens the app.
func (c *SettingsService) CheckPin(pin string) (PinResult, error) {
	settingsLock.Lock()
	defer settingsLock.Unlock()
	s, err := readSettings()
	if err != nil {
		return PinResult{}, err
	}
	res, err := check(&s, pin)
	if err != nil {
		return res, err
	}
	return res, writeSettings(s)
}

// RemovePin takes the lock off, given the PIN.
func (c *SettingsService) RemovePin(pin string) (PinResult, error) {
	settingsLock.Lock()
	defer settingsLock.Unlock()
	s, err := readSettings()
	if err != nil {
		return PinResult{}, err
	}
	res, err := check(&s, pin)
	if err != nil {
		return res, err
	}
	if res.OK {
		s.Salt, s.Hash = nil, nil
	}
	return res, writeSettings(s)
}
