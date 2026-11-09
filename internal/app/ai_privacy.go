package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// AI in the cloud and privacy. The notes stay on the computer unless the
// person sends them to an AI in the cloud (OpenRouter with their own key,
// or Blup Cloud on the plan's allowance) to translate or narrate. Before
// the first time for each of the two, the Studio says what leaves the
// computer and where it goes, and the person accepts or not; the answer is
// kept here, on this computer. The Go side refuses to send without it (the
// window asking is not enough). Password protected posts never go.

// The AIs in the cloud a note's text can go to.
const (
	AIOpenRouter = "openrouter"
	AIBlup       = "blup"
)

// AIConsent is what the person accepted: their notes' text may go to
// OpenRouter, to Blup Cloud.
type AIConsent struct {
	OpenRouter bool `json:"openrouter"`
	Blup       bool `json:"blup"`
}

// ConsentError: the note's text would go to Dest, which the person did not
// accept yet. The window asks and calls SetAIConsent, then tries again.
type ConsentError struct{ Dest string }

func (e *ConsentError) Error() string { return "ai consent needed: " + e.Dest }

// ErrNoProvider: nobody was chosen to translate (the Studio never picks an
// AI in the cloud by itself).
var ErrNoProvider = errors.New("no AI provider chosen")

func consentFile() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ai-consent.json"), nil
}

// LoadAIConsent reads what was accepted (nothing, when never asked). A
// damaged file (not an object, a field of the wrong kind, cut short) counts
// as nothing accepted: in doubt, the person is asked again.
func LoadAIConsent() AIConsent {
	f, err := consentFile()
	if err != nil {
		return AIConsent{}
	}
	b, err := os.ReadFile(f)
	if err != nil {
		return AIConsent{}
	}
	var c *AIConsent
	if json.Unmarshal(b, &c) != nil || c == nil {
		return AIConsent{}
	}
	return *c
}

// SetAIConsent keeps the person's answer for dest ("openrouter" or "blup").
func SetAIConsent(dest string, ok bool) error {
	c := LoadAIConsent()
	switch dest {
	case AIOpenRouter:
		c.OpenRouter = ok
	case AIBlup:
		c.Blup = ok
	default:
		return errors.New("unknown AI: " + dest)
	}
	f, err := consentFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f), 0o700); err != nil {
		return err
	}
	b, _ := json.Marshal(c)
	tmp := f + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f)
}

// needConsent is nil when the person accepted sending notes to dest.
func needConsent(dest string) error {
	c := LoadAIConsent()
	if dest == AIOpenRouter && c.OpenRouter || dest == AIBlup && c.Blup {
		return nil
	}
	return &ConsentError{Dest: dest}
}

// noteLocked tells whether a note is password protected, as the theme reads
// "password" (theme/layouts/_partials/locked.html, Hugo's "with"): any
// value but nothing, false, zero or an empty text. A text of spaces counts
// (the theme encrypts with it), as does "false" written as text.
func noteLocked(n obsidian.Note) bool {
	switch v := n.Props["password"].(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case int:
		return v != 0
	case float64:
		return v != 0
	}
	return true
}
