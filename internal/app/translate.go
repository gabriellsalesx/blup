package app

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/narrate"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/scaffold"
	"github.com/gabriellsalesx/blup/internal/translate"
)

// Translations of notes. A note's translation into a language lives in a
// folder named after it, next to the note, under its own translated name
// (Posts/Bolo.md → Posts/es/Pastel.md): the site publishes it as that
// language's page (internal/scaffold). The original and its translations
// share translationKey (Hugo links pages by it; it is set on the original
// at its first translation), and a translation keeps translation_hash, the
// fingerprint of the original it came from, to tell when it is out of date.
// A translation with the original's own file name (and no key) is one too.

// hashKey is the property with the original's fingerprint; linkKey the
// one shared by a note and its translations.
const (
	hashKey = "translation_hash"
	linkKey = "translationKey"
	// aiKey marks a translation made by AI (the site shows it); the author
	// takes it off by marking the translation reviewed.
	aiKey = "ai_translated"
)

// TextProps are the properties a translation translates besides the title
// and description: what the site shows (the site's description and button
// on the home note, a help article's category, the tags) and the menu's
// labels ("Name | target": the name).
var TextProps = []string{"linkTitle", "site_description", "button_text", "category", "tags", "menu", "role"}

// translateVersion goes into each translation's fingerprint: when what a
// translation covers changes (a property more), the translations made
// before show out of date, and Update the translations redoes them.
const translateVersion = "2"

// NoteLang is a language a note can be read in: its own (Original), or a
// translation (Path, there or not yet, Stale when the original changed
// since).
type NoteLang struct {
	Lang     string `json:"lang"`
	Path     string `json:"path"`
	Original bool   `json:"original"`
	Exists   bool   `json:"exists"`
	Stale    bool   `json:"stale"`
	AI       bool   `json:"ai"` // made by AI, not marked reviewed
}

// NoteOrigin is the original of a note and the note's language: a note in
// a language folder (Posts/en/Note.md) is the translation of the note of
// the parent folder with its translationKey (or its file name); any other
// note is an original in the site's language.
func NoteOrigin(site *config.Site, rel string) (original, lang string) {
	dir, file := path.Split(rel)
	dir = strings.TrimSuffix(dir, "/")
	l := path.Base(dir)
	if dir == "" || l == site.Lang || !slices.Contains(config.SiteLangs, l) {
		return rel, site.Lang
	}
	parent := path.Dir(dir)
	if parent == "." {
		parent = ""
	}
	if p, err := notesPath(site, rel); err == nil {
		if n, err := obsidian.ReadNote(p); err == nil {
			if key := n.String(linkKey); key != "" {
				if o := noteWithKey(site, parent, key); o != "" {
					return o, l
				}
				// The key given at the first translation is the original's path
				// (when the key was taken out of the original).
				if o := key + ".md"; path.Dir(o) == cmp.Or(parent, ".") {
					if op, err := notesPath(site, o); err == nil {
						if _, err := os.Stat(op); err == nil {
							return o, l
						}
					}
				}
			}
		}
	}
	return path.Join(parent, file), l
}

// noteWithKey is the note of folder dir (only its own notes) whose
// translationKey is key ("" when none).
func noteWithKey(site *config.Site, dir, key string) string {
	d, err := notesPath(site, dir)
	if err != nil {
		return ""
	}
	entries, err := os.ReadDir(d)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(path.Ext(e.Name()), ".md") {
			continue
		}
		if n, err := obsidian.ReadNote(filepath.Join(d, e.Name())); err == nil && n.String(linkKey) == key {
			return path.Join(dir, e.Name())
		}
	}
	return ""
}

// translationPath is where the translation of original (whose note is n)
// into lang is: the note of the language folder with its translationKey,
// else the one with its file name ("" when there is none yet).
func translationPath(site *config.Site, original string, n obsidian.Note, lang string) string {
	dir, file := path.Split(original)
	folder := path.Join(dir, lang)
	// Its key, or the one given at the first translation (the original's
	// path), when the key was taken out of the original.
	for _, key := range []string{n.String(linkKey), strings.TrimSuffix(original, path.Ext(original))} {
		if key == "" {
			continue
		}
		if t := noteWithKey(site, folder, key); t != "" {
			return t
		}
	}
	same := path.Join(folder, file)
	if p, err := notesPath(site, same); err == nil {
		if _, err := os.Stat(p); err == nil {
			return same
		}
	}
	return ""
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
		nl := NoteLang{Lang: l, Path: translationPath(site, original, n, l)}
		if p, err := notesPath(site, nl.Path); nl.Path != "" && err == nil {
			if t, err := obsidian.ReadNote(p); err == nil {
				nl.Exists = true
				nl.Stale = t.String(hashKey) != "" && t.String(hashKey) != sum
				nl.AI = t.Props[aiKey] == true
			}
		}
		out = append(out, nl)
	}
	return out, nil
}

// sourceHash is the fingerprint of what a translation is made from: the
// title (or the file's name), the description and the text, and the
// feedback board's example requests (only that note has them, so the
// others keep their fingerprint).
func sourceHash(n obsidian.Note, rel string) string {
	h := sha256.New()
	h.Write([]byte(translateVersion))
	for _, s := range append([]string{noteTitle(n, rel), n.String("description"), n.Body}, exampleTexts(n.Props[examplesKey])...) {
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

// translator is the AI in the cloud the person chose to translate
// (provider "openrouter" or "blup"): never picked by itself, and only once
// the person accepted sending the notes there (needConsent).
func translator(ctx context.Context, provider string) (*translate.Client, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	switch provider {
	case AIBlup:
		if creds.BlupToken == "" {
			return nil, errors.New(describe(blupcloud.ErrSignedOut))
		}
		if !blupAIIncluded(ctx) {
			return nil, errors.New(byLangCode(string(i18n.Current()),
				"O seu plano do Blup Cloud não inclui IA. Escolha outro jeito de traduzir nas configurações de IA.",
				"Your Blup Cloud plan has no AI. Pick another way to translate in the AI settings.",
				"Tu plan de Blup Cloud no incluye IA. Elige otra forma de traducir en la configuración de IA."))
		}
		if err := needConsent(AIBlup); err != nil {
			return nil, err
		}
		return &translate.Client{Key: creds.BlupToken, BaseURL: blupcloud.AIBaseURL(), Blup: true}, nil
	case AIOpenRouter:
		if creds.OpenRouterToken == "" {
			return nil, ErrNoAIKey
		}
		if err := needConsent(AIOpenRouter); err != nil {
			return nil, err
		}
		// The models the staff picked for one's own key, in order; none, none.
		models := studioAI(ctx).Translate.Own
		if len(models) == 0 {
			return nil, errors.New(byLangCode(string(i18n.Current()),
				"Nenhum modelo de tradução está disponível agora. Tente mais tarde.",
				"No translation model is available right now. Try again later.",
				"Ningún modelo de traducción está disponible ahora. Inténtalo más tarde."))
		}
		return &translate.Client{Key: creds.OpenRouterToken, BaseURL: cmp.Or(openRouterURL, narrate.DefaultBaseURL), Models: models}, nil
	}
	return nil, ErrNoProvider
}

// errLockedCloud: a password protected post does not go to an AI in the
// cloud (the translation on the computer still works).
func errLockedCloud(site *config.Site) error {
	return errors.New(byLang(site,
		"Posts com senha não vão para a IA na nuvem. Use a tradução no computador.",
		"Password protected posts don't go to the AI in the cloud. Use the translation on the computer.",
		"Los posts con contraseña no van a la IA en la nube. Usa la traducción en la computadora."))
}

// TranslateNote writes (or writes again) the translation of the note at
// rel into lang with the AI in the cloud chosen (provider), and returns
// its path. The site then publishes lang too.
func TranslateNote(ctx context.Context, site *config.Site, rel, lang, provider string, progress func(done, total int)) (string, error) {
	if lang == site.Lang || !slices.Contains(config.SiteLangs, lang) {
		return "", errors.New("translate: unknown language " + lang)
	}
	c, err := translator(ctx, provider)
	if err != nil {
		return "", err
	}
	return translateNoteWith(ctx, c, site, rel, lang, progress)
}

// friendly is err with the message people read (describe), still
// matching what it wraps (errors.Is), so the batch knows a used up
// allowance from a note that failed alone.
func friendly(err error) error { return friendlyErr{describe(err), err} }

type friendlyErr struct {
	msg string
	err error
}

func (e friendlyErr) Error() string { return e.msg }
func (e friendlyErr) Unwrap() error { return e.err }

// writeTranslation keeps the translations being written one at a time.
var writeTranslation sync.Mutex

func translateNoteWith(ctx context.Context, c *translate.Client, site *config.Site, rel, lang string, progress func(done, total int)) (string, error) {
	original, _ := NoteOrigin(site, rel)
	op, err := notesPath(site, original)
	if err != nil {
		return "", err
	}
	n, err := obsidian.ReadNote(op)
	if err != nil {
		return "", err
	}
	if noteLocked(n) {
		return "", errLockedCloud(site)
	}
	// Two requests at once: the text, and every short field (title,
	// description, the properties' texts) together in one.
	type answer struct {
		text string
		err  error
	}
	bodyDone := make(chan answer, 1)
	go func() {
		t, err := c.Text(ctx, n.Body, site.Lang, lang, progress)
		bodyDone <- answer{t, err}
	}()
	desc := n.String("description")
	fields := []string{noteTitle(n, original), desc}
	ptexts := PropTexts(n)
	keys := slices.Sorted(maps.Keys(ptexts))
	for _, k := range keys {
		fields = append(fields, ptexts[k]...)
	}
	got, ferr := c.Fields(ctx, fields, site.Lang, lang)
	b := <-bodyDone
	if ferr != nil {
		return "", friendly(ferr)
	}
	if b.err != nil {
		return "", friendly(b.err)
	}
	title, body := got[0], b.text
	if desc != "" {
		desc = got[1]
	}
	props := map[string]any{}
	at := 2
	for _, k := range keys {
		out := make([]string, len(ptexts[k]))
		for i := range out {
			out[i] = strings.TrimSpace(got[at])
			at++
		}
		props[k] = out
	}
	// Writing is one note at a time (the file's free name in the folder,
	// the site's languages): the requests above run side by side.
	writeTranslation.Lock()
	defer writeTranslation.Unlock()
	return WriteTranslation(site, rel, lang, title, desc, body, props)
}

// PropTexts are the texts to translate in a note's TextProps: each one's
// values (one for a single value; a menu line's name only).
func PropTexts(n obsidian.Note) map[string][]string {
	out := map[string][]string{}
	for _, k := range TextProps {
		var texts []string
		for _, v := range n.List(k) {
			if k == "menu" {
				if !strings.Contains(v, "|") {
					continue // a folder or note by its name: the site shows its translated title
				}
				v = parseMenuLine(v).Name
			}
			if v = strings.TrimSpace(v); v != "" {
				texts = append(texts, v)
			}
		}
		if len(texts) > 0 {
			out[k] = texts
		}
	}
	if texts := exampleTexts(n.Props[examplesKey]); len(texts) > 0 {
		out[examplesKey] = texts
	}
	return out
}

// examplesKey is the feedback board's example requests (Bento): a list of
// requests, each with its title, text and example comments (replies).
const examplesKey = "examples"

// exampleTexts are the texts of the example requests to translate, in
// order: each one's title, its text, then its comments' texts.
func exampleTexts(v any) []string {
	var out []string
	walkExamples(v, func(s string) string {
		out = append(out, s)
		return s
	})
	return out
}

// walkExamples calls fn on each text of the example requests (as
// exampleTexts lists them) and returns a copy with what fn gives back.
func walkExamples(v any, fn func(string) string) any {
	list, ok := v.([]any)
	if !ok {
		return v
	}
	text := func(m map[string]any, k string) {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			m[k] = fn(strings.TrimSpace(s))
		}
	}
	out := make([]any, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			out[i] = item
			continue
		}
		c := maps.Clone(m)
		text(c, "title")
		text(c, "body")
		if replies, ok := c["replies"].([]any); ok {
			rs := make([]any, len(replies))
			for j, r := range replies {
				if rm, ok := r.(map[string]any); ok {
					rc := maps.Clone(rm)
					text(rc, "text")
					rs[j] = rc
				} else {
					rs[j] = r
				}
			}
			c["replies"] = rs
		}
		out[i] = c
	}
	return out
}

// translatedProps puts the translated texts (as PropTexts lists them)
// back in n's properties.
func translatedProps(n obsidian.Note, tr map[string]any) []obsidian.Prop {
	var props []obsidian.Prop
	for k, v := range tr {
		texts := toStrings(v)
		if k == examplesKey {
			// The same requests, their texts in the new language.
			if len(texts) == len(exampleTexts(n.Props[k])) {
				i := 0
				props = append(props, obsidian.Prop{Key: k, Value: walkExamples(n.Props[k], func(string) string { i++; return texts[i-1] })})
			}
			continue
		}
		orig := n.List(k)
		if !slices.Contains(TextProps, k) || len(texts) == 0 || len(orig) == 0 {
			continue
		}
		if k == "menu" {
			i := 0
			out := make([]string, len(orig))
			for j, line := range orig {
				out[j] = line
				if m := parseMenuLine(line); strings.Contains(line, "|") && m.Name != "" && i < len(texts) {
					m.Name = texts[i]
					out[j] = menuLine(m)
					i++
				}
			}
			props = append(props, obsidian.Prop{Key: k, Value: out})
			continue
		}
		if _, single := n.Props[k].(string); single {
			props = append(props, obsidian.Prop{Key: k, Value: texts[0]})
		} else {
			props = append(props, obsidian.Prop{Key: k, Value: texts})
		}
	}
	return props
}

func toStrings(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, i := range x {
			if s, ok := i.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{x}
	}
	return nil
}

// WriteTranslation saves the translation of rel (or of rel's original)
// into lang, made here or in the window (the local translation): the
// original's properties with its own title and description, the text,
// and the original's fingerprint. It returns the translation's path; the
// site then publishes lang too.
func WriteTranslation(site *config.Site, rel, lang, title, desc, body string, props map[string]any) (string, error) {
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
	if v, supplied := props[examplesKey]; supplied || len(exampleTexts(n.Props[examplesKey])) > 0 {
		// Validate before changing either file. Silently dropping a malformed
		// translation would still mark the original examples as up to date.
		texts, ok := v.([]string)
		if values, isList := v.([]any); isList {
			ok = true
			texts = make([]string, len(values))
			for i, value := range values {
				var isText bool
				texts[i], isText = value.(string)
				ok = ok && isText
			}
		}
		if !ok || len(texts) != len(exampleTexts(n.Props[examplesKey])) {
			return "", errors.New("translate: invalid feedback example texts")
		}
	}
	// The note and its translations share a key, set on the original at
	// its first translation.
	key := n.String(linkKey)
	if key == "" {
		key = strings.TrimSuffix(original, path.Ext(original))
		if err := obsidian.SetProperties(op, obsidian.Prop{Key: linkKey, Value: key}); err != nil {
			return "", err
		}
		n.Props[linkKey] = key
	}
	// Where it is already, else under its translated title.
	target := translationPath(site, original, n, lang)
	// A section's note keeps its name (it is the section in that
	// language), and so does an author's profile (its name is the author).
	if target == "" && (strings.EqualFold(path.Base(original), "_index.md") || isAuthor(site, original)) {
		target = path.Join(path.Dir(original), lang, path.Base(original))
	}
	if target == "" {
		if target, err = translationName(site, path.Join(path.Dir(original), lang), safeFileName(strings.TrimSpace(title)), original); err != nil {
			return "", err
		}
	}
	tp, err := notesPath(site, target)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(tp), 0o755); err != nil {
		return "", err
	}
	// The original's properties, then its own title, description, text and
	// fingerprint; a narration is of the original's language, not this one.
	src, err := os.ReadFile(op) // with the key, now
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(tp, src, 0o644); err != nil {
		return "", err
	}
	if err := obsidian.SetBody(tp, body); err != nil {
		return "", err
	}
	set := []obsidian.Prop{
		{Key: "title", Value: strings.TrimSpace(title)},
		{Key: hashKey, Value: sourceHash(n, original)},
		{Key: aiKey, Value: true},
		{Key: "audio", Value: nil}, {Key: "audio_hash", Value: nil}, {Key: "audio_seconds", Value: nil},
	}
	if strings.TrimSpace(desc) != "" {
		set = append(set, obsidian.Prop{Key: "description", Value: strings.TrimSpace(desc)})
	}
	set = append(set, translatedProps(n, props)...)
	if err := obsidian.SetProperties(tp, set...); err != nil {
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

// siteOriginals are the site's notes that get translations: every note
// in the site's language (the pages, and the authors' profiles, which the
// pages show), not the agent guides or the templates.
func siteOriginals(site *config.Site) ([]string, error) {
	root := site.ContentDir()
	skip := map[string]bool{}
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
	op, err := notesPath(site, rel)
	if err != nil {
		return "missing"
	}
	n, err := obsidian.ReadNote(op)
	if err != nil {
		return "missing"
	}
	tr := translationPath(site, rel, n, lang)
	if tr == "" {
		return "missing"
	}
	tp, err := notesPath(site, tr)
	if err != nil {
		return "missing"
	}
	t, err := obsidian.ReadNote(tp)
	if err != nil {
		return "missing"
	}
	if t.String(hashKey) != "" && t.String(hashKey) != sourceHash(n, rel) {
		return "stale"
	}
	return "done"
}

// translationName is a note path in dir named name (a number added when taken by
// another note), or the original's own name when name is empty.
func translationName(site *config.Site, dir, name, original string) (string, error) {
	if name == "" {
		name = strings.TrimSuffix(path.Base(original), path.Ext(original))
	}
	for i := 1; ; i++ {
		base := name
		if i > 1 {
			base = fmt.Sprintf("%s %d", name, i)
		}
		rel := path.Join(dir, base+".md")
		p, err := notesPath(site, rel)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			return rel, nil
		}
	}
}

// SiteTranslation is what the translation of the whole site did: notes
// translated, and password protected ones left out (they don't go to an AI
// in the cloud).
type SiteTranslation struct {
	Done    int `json:"done"`
	Skipped int `json:"skipped"`
	// Failed are the notes that could not be translated this time (the
	// rest went on); translating again tries only them.
	Failed []string `json:"failed"`
}

// TranslateReady says whether provider can translate now (a key, a plan
// with AI, the consent), before a batch starts on its own.
func TranslateReady(ctx context.Context, provider string) error {
	_, err := translator(ctx, provider)
	return err
}

// siteWorkers is how many notes a batch translates side by side: two
// requests each, under Blup Cloud's 120 AI requests a minute an account.
const siteWorkers = 3

// TranslateSite translates every note of the site into lang that has no
// translation yet, or an out of date one, with the AI in the cloud chosen
// (provider), a few notes at a time. Password protected posts are left
// out and counted. progress gets the notes done of total and a note being
// translated. A note that fails alone is listed in Failed and the rest go
// on; what holds for every note (no key, no allowance) stops the batch
// (the notes done stay).
func TranslateSite(ctx context.Context, site *config.Site, lang, provider string, progress func(done, total int, note string)) (SiteTranslation, error) {
	var res SiteTranslation
	if lang == site.Lang || !slices.Contains(config.SiteLangs, lang) {
		return res, errors.New("translate: unknown language " + lang)
	}
	notes, err := siteOriginals(site)
	if err != nil {
		return res, err
	}
	var todo []string
	for _, rel := range notes {
		if translationOf(site, rel, lang) == "done" {
			continue
		}
		if n, err := ReadSiteNote(site, rel); err == nil && noteLocked(n) {
			res.Skipped++
			continue
		}
		todo = append(todo, rel)
	}
	if len(todo) == 0 {
		return res, addSiteLang(site, lang)
	}
	c, err := translator(ctx, provider)
	if err != nil {
		return res, err
	}
	// A few notes at a time (siteWorkers): each waits mostly on the AI,
	// so side by side the batch takes a fraction of the time, at the same
	// cost. What holds for every note (no key, no allowance, cancelled)
	// stops them all; a note that failed alone is left for later.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var mu sync.Mutex
	var fatal error
	done := 0
	next := make(chan string)
	var wg sync.WaitGroup
	for range min(siteWorkers, len(todo)) {
		wg.Go(func() {
			for rel := range next {
				mu.Lock()
				if progress != nil && fatal == nil {
					progress(done, len(todo), rel)
				}
				mu.Unlock()
				_, err := translateNoteWith(ctx, c, site, rel, lang, nil)
				mu.Lock()
				switch {
				case err == nil:
					res.Done++
				case ctx.Err() != nil || errors.Is(err, translate.ErrInvalidKey) || errors.Is(err, translate.ErrNoCredits) || errors.Is(err, translate.ErrLimit):
					if fatal == nil {
						fatal = err
					}
					cancel()
				default:
					res.Failed = append(res.Failed, rel)
				}
				done++
				mu.Unlock()
			}
		})
	}
feed:
	for _, rel := range todo {
		select {
		case next <- rel:
		case <-ctx.Done():
			break feed
		}
	}
	close(next)
	wg.Wait()
	if fatal != nil {
		return res, fatal
	}
	slices.Sort(res.Failed)
	if progress != nil {
		progress(len(todo), len(todo), "")
	}
	return res, addSiteLang(site, lang)
}

// TranslationTodo lists the site's notes without an up to date
// translation into lang (for the local translation, which works in the
// window note by note).
func TranslationTodo(site *config.Site, lang string) ([]string, error) {
	notes, err := siteOriginals(site)
	if err != nil {
		return nil, err
	}
	todo := []string{}
	for _, rel := range notes {
		if translationOf(site, rel, lang) != "done" {
			todo = append(todo, rel)
		}
	}
	return todo, nil
}

// LocalTranslateDir is where the local translation is installed.
func LocalTranslateDir() (string, error) {
	d, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "blup", "translate"), nil
}

// MarkReviewed marks the translation at rel reviewed by its author (the
// site stops showing it was made by AI), or not.
func MarkReviewed(site *config.Site, rel string, reviewed bool) error {
	p, err := notesPath(site, rel)
	if err != nil {
		return err
	}
	var v any = true
	if reviewed {
		v = nil
	}
	return obsidian.SetProperties(p, obsidian.Prop{Key: aiKey, Value: v})
}

// ReadSiteNote reads the note at rel of the site.
func ReadSiteNote(site *config.Site, rel string) (obsidian.Note, error) {
	p, err := notesPath(site, rel)
	if err != nil {
		return obsidian.Note{}, err
	}
	return obsidian.ReadNote(p)
}

// isAuthor tells whether rel is a profile of the authors folder.
func isAuthor(site *config.Site, rel string) bool {
	a := obsidian.Sections(site.ContentDir())["authors"]
	return a != "" && path.Dir(rel) == a && !strings.EqualFold(path.Base(rel), "_index.md")
}

// RemoveTranslation takes lang off the site: every folder of that
// language (the translations, kept next to their originals) goes to the
// Blup trash, each one restorable, and the site stops publishing lang.
// The originals are never touched. Anything that fails halfway is put
// back, so the notes are never left half removed. It returns where each
// folder went, for RestoreTranslation.
func RemoveTranslation(site *config.Site, lang string) ([]string, error) {
	if lang == site.Lang || !slices.Contains(config.SiteLangs, lang) {
		return nil, errors.New("translate: unknown language " + lang)
	}
	folders, err := langFolders(site, lang)
	if err != nil {
		return nil, err
	}
	var trashed []string
	for _, rel := range folders {
		t, err := toTrash(site, rel)
		if err != nil {
			for _, back := range trashed {
				_ = RestoreSection(site, back)
			}
			return nil, err
		}
		trashed = append(trashed, t)
	}
	if i := slices.Index(site.Translations, lang); i >= 0 {
		site.Translations = slices.Delete(site.Translations, i, i+1)
		if err := site.Save(); err != nil {
			return trashed, err
		}
		if err := scaffold.Write(site); err != nil {
			return trashed, err
		}
	}
	return trashed, nil
}

// RestoreTranslation brings back what RemoveTranslation took off, and
// the site publishes lang again.
func RestoreTranslation(site *config.Site, lang string, trashed []string) error {
	var first error
	for _, t := range trashed {
		if err := RestoreSection(site, t); err != nil && first == nil {
			first = err
		}
	}
	if err := addSiteLang(site, lang); err != nil && first == nil {
		first = err
	}
	return first
}

// langFolders are the notes' folders named lang (where the translations
// into lang live), outermost only, outside hidden folders.
func langFolders(site *config.Site, lang string) ([]string, error) {
	root := site.ContentDir()
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || p == root {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.Name() == lang {
			rel, _ := filepath.Rel(root, p)
			out = append(out, filepath.ToSlash(rel))
			return filepath.SkipDir
		}
		return nil
	})
	return out, err
}
