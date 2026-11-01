package app

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
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

// SiteLangState is how much of the site is in a language: notes
// translated and up to date, out of date, and not translated yet.
type SiteLangState struct {
	Lang    string `json:"lang"`
	On      bool   `json:"on"` // published (the site has it)
	Done    int    `json:"done"`
	Stale   int    `json:"stale"`
	Missing int    `json:"missing"`
}

// siteOriginals are the site's notes that get translations: every page
// note in the site's language (not the agent guides, the authors' profiles
// or the templates, which are never pages).
func siteOriginals(site *config.Site) ([]string, error) {
	root := site.ContentDir()
	skip := map[string]bool{}
	if a := obsidian.Sections(root)["authors"]; a != "" {
		skip[a] = true
	}
	if t := obsidian.TemplatesFolder(site.Vault); t != "" {
		if rel, err := filepath.Rel(root, filepath.Join(site.Vault, t)); err == nil {
			skip[filepath.ToSlash(rel)] = true
		}
	}
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if p != root && (strings.HasPrefix(d.Name(), ".") || skip[rel] || (slices.Contains(config.SiteLangs, d.Name()) && d.Name() != site.Lang)) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(path.Ext(rel), ".md") && !slices.Contains(scaffold.AgentGuides, rel) {
			out = append(out, rel)
		}
		return nil
	})
	return out, err
}

// SiteLangs tells, for each language other than the site's, how much of
// the site is translated.
func SiteLangs(site *config.Site) ([]SiteLangState, error) {
	notes, err := siteOriginals(site)
	if err != nil {
		return nil, err
	}
	var out []SiteLangState
	for _, l := range config.SiteLangs {
		if l == site.Lang {
			continue
		}
		st := SiteLangState{Lang: l, On: slices.Contains(site.Translations, l)}
		for _, rel := range notes {
			switch translationOf(site, rel, l) {
			case "done":
				st.Done++
			case "stale":
				st.Stale++
			default:
				st.Missing++
			}
		}
		out = append(out, st)
	}
	return out, nil
}

// translationOf is the state of rel's translation into lang: "done",
// "stale" or "missing".
func translationOf(site *config.Site, rel, lang string) string {
	tp, err := notesPath(site, translationPath(rel, lang))
	if err != nil {
		return "missing"
	}
	t, err := obsidian.ReadNote(tp)
	if _, serr := os.Stat(tp); err != nil || serr != nil {
		return "missing"
	}
	op, _ := notesPath(site, rel)
	n, err := obsidian.ReadNote(op)
	if err == nil && t.String(hashKey) != "" && t.String(hashKey) != sourceHash(n, rel) {
		return "stale"
	}
	return "done"
}

// TranslateSite translates every note of the site into lang that has no
// translation yet, or an out of date one. progress gets the notes done of
// total and the note being translated. It stops at the first error (the
// notes done stay).
func TranslateSite(ctx context.Context, site *config.Site, lang string, progress func(done, total int, note string)) (int, error) {
	notes, err := siteOriginals(site)
	if err != nil {
		return 0, err
	}
	var todo []string
	for _, rel := range notes {
		if translationOf(site, rel, lang) != "done" {
			todo = append(todo, rel)
		}
	}
	for i, rel := range todo {
		if progress != nil {
			progress(i, len(todo), rel)
		}
		if _, err := TranslateNote(ctx, site, rel, lang, nil); err != nil {
			return i, err
		}
	}
	if progress != nil {
		progress(len(todo), len(todo), "")
	}
	return len(todo), addSiteLang(site, lang)
}
