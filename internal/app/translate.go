package app

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/narrate"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/scaffold"
	"github.com/gabriellsalesx/blup/internal/translate"
)

// Translations of notes. A note's translation into a language lives in a
// folder named after it, next to the note, with the same file name
// (Posts/Note.md → Posts/en/Note.md): the site publishes it as that
// language's page (internal/scaffold). It keeps the original's properties
// (its own title and description translated) and translation_hash, the
// fingerprint of the original it came from, to tell when it is out of date.

// hashKey is the property with the original's fingerprint.
const hashKey = "translation_hash"

// NoteLang is a language a note can be read in: its own (Original), or a
// translation (Path, there or not yet, Stale when the original changed
// since).
type NoteLang struct {
	Lang     string `json:"lang"`
	Path     string `json:"path"`
	Original bool   `json:"original"`
	Exists   bool   `json:"exists"`
	Stale    bool   `json:"stale"`
}

// NoteOrigin is the original of a note and the note's language: a note in
// a language folder (Posts/en/Note.md) is the translation of
// Posts/Note.md; any other note is an original in the site's language.
func NoteOrigin(site *config.Site, rel string) (original, lang string) {
	dir, file := path.Split(rel)
	dir = strings.TrimSuffix(dir, "/")
	if l := path.Base(dir); dir != "" && l != site.Lang && slices.Contains(config.SiteLangs, l) {
		return path.Join(path.Dir(dir), file), l
	}
	return rel, site.Lang
}

// translationPath is where rel's translation into lang lives.
func translationPath(original, lang string) string {
	dir, file := path.Split(original)
	return path.Join(dir, lang, file)
}

// NoteLangs lists the languages of the note at rel: the original first,
// then the other site languages, translated or not.
func NoteLangs(site *config.Site, rel string) ([]NoteLang, error) {
	original, _ := NoteOrigin(site, rel)
	op, err := notesPath(site, original)
	if err != nil {
		return nil, err
	}
	n, err := obsidian.ReadNote(op)
	if err != nil {
		return nil, err
	}
	sum := sourceHash(n, original)
	out := []NoteLang{{Lang: site.Lang, Path: original, Original: true, Exists: true}}
	for _, l := range config.SiteLangs {
		if l == site.Lang {
			continue
		}
		nl := NoteLang{Lang: l, Path: translationPath(original, l)}
		if p, err := notesPath(site, nl.Path); err == nil {
			if t, err := obsidian.ReadNote(p); err == nil {
				if _, err := os.Stat(p); err == nil {
					nl.Exists = true
					nl.Stale = t.String(hashKey) != "" && t.String(hashKey) != sum
				}
			}
		}
		out = append(out, nl)
	}
	return out, nil
}

// sourceHash is the fingerprint of what a translation is made from: the
// title (or the file's name), the description and the text.
func sourceHash(n obsidian.Note, rel string) string {
	h := sha256.New()
	for _, s := range []string{noteTitle(n, rel), n.String("description"), n.Body} {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func noteTitle(n obsidian.Note, rel string) string {
	if t := n.String("title"); t != "" {
		return t
	}
	return strings.TrimSuffix(path.Base(rel), path.Ext(rel))
}

// translator is the AI that translates: Blup Cloud when the plan includes
// AI, else the OpenRouter key saved.
func translator(ctx context.Context) (*translate.Client, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	if creds.BlupToken != "" && blupAIIncluded(ctx) {
		return &translate.Client{Key: creds.BlupToken, BaseURL: blupcloud.AIBaseURL(), Blup: true}, nil
	}
	if creds.OpenRouterToken != "" {
		return &translate.Client{Key: creds.OpenRouterToken, BaseURL: cmp.Or(openRouterURL, narrate.DefaultBaseURL)}, nil
	}
	return nil, ErrNoAIKey
}

// TranslateNote writes (or writes again) the translation of the note at
// rel into lang, and returns its path. The site then publishes lang too.
func TranslateNote(ctx context.Context, site *config.Site, rel, lang string, progress func(done, total int)) (string, error) {
	if lang == site.Lang || !slices.Contains(config.SiteLangs, lang) {
		return "", errors.New("translate: unknown language " + lang)
	}
	original, _ := NoteOrigin(site, rel)
	op, err := notesPath(site, original)
	if err != nil {
		return "", err
	}
	n, err := obsidian.ReadNote(op)
	if err != nil {
		return "", err
	}
	c, err := translator(ctx)
	if err != nil {
		return "", err
	}
	title, err := c.Text(ctx, noteTitle(n, original), site.Lang, lang, nil)
	if err != nil {
		return "", errors.New(describe(err))
	}
	desc := n.String("description")
	if desc != "" {
		if desc, err = c.Text(ctx, desc, site.Lang, lang, nil); err != nil {
			return "", errors.New(describe(err))
		}
	}
	body, err := c.Text(ctx, n.Body, site.Lang, lang, progress)
	if err != nil {
		return "", errors.New(describe(err))
	}

	target := translationPath(original, lang)
	tp, err := notesPath(site, target)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(tp), 0o755); err != nil {
		return "", err
	}
	// The original's properties, then its own title, description, text and
	// fingerprint; a narration is of the original's language, not this one.
	src, err := os.ReadFile(op)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(tp, src, 0o644); err != nil {
		return "", err
	}
	if err := obsidian.SetBody(tp, body); err != nil {
		return "", err
	}
	props := []obsidian.Prop{
		{Key: "title", Value: strings.TrimSpace(title)},
		{Key: hashKey, Value: sourceHash(n, original)},
		{Key: "audio", Value: nil}, {Key: "audio_hash", Value: nil}, {Key: "audio_seconds", Value: nil},
	}
	if desc != "" {
		props = append(props, obsidian.Prop{Key: "description", Value: strings.TrimSpace(desc)})
	}
	if err := obsidian.SetProperties(tp, props...); err != nil {
		return "", err
	}
	return target, addSiteLang(site, lang)
}

// addSiteLang makes the site publish lang (once it has a translation).
func addSiteLang(site *config.Site, lang string) error {
	if site.IsNotes() || slices.Contains(site.Translations, lang) {
		return nil
	}
	site.Translations = append(site.Translations, lang)
	if err := site.Save(); err != nil {
		return err
	}
	return scaffold.Write(site)
}
