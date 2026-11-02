// Package scaffold writes a site folder: the generated hugo.toml (vault
// mounts included), the embedded theme and the folders for overrides.
package scaffold

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/gabriellsalesx/blup"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// Attachments are the vault files that pages may reference. They are
// mounted as Hugo assets, so only the ones actually used get published.
var Attachments = []string{
	"png", "jpg", "jpeg", "webp", "avif", "gif", "svg", "bmp", "tif", "tiff",
	"pdf", "mp3", "m4a", "ogg", "wav", "flac", "mp4", "webm", "mov", "zip",
}

// Write creates or refreshes everything the CLI owns inside the site folder.
// Files the user may customise (layouts/, assets/, static/) are never touched.
func Write(s *config.Site) error {
	for _, d := range []string{"layouts", "assets", "static", "i18n", "data"} {
		if err := os.MkdirAll(filepath.Join(s.Dir(), d), 0o755); err != nil {
			return err
		}
	}
	if err := writeTheme(filepath.Join(s.Dir(), "themes", "blup")); err != nil {
		return err
	}
	b, err := HugoConfig(s, false)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.Dir(), "hugo.toml"), b, 0o644); err != nil {
		return err
	}
	if b, err = HugoConfig(s, true); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Dir(), PreviewConfig), b, 0o644)
}

// AgentGuides are the files at the top of the notes folder that explain
// Blup to AI agents (Claude Code, Codex, OpenCode, Gemini, Antigravity).
// They are left out of the site.
var AgentGuides = []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md", "INSTRUCTIONS.md"}

// PreviewConfig is the config used by the preview server: the same site
// plus PreviewStatic, where the preview's search index is written.
const PreviewConfig = "hugo.preview.toml"

// PreviewStatic is mounted as static files only during the preview.
const PreviewStatic = ".blup/preview"

func writeTheme(dest string) error {
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	return fs.WalkDir(blup.Theme, "theme", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(strings.TrimPrefix(p, "theme")))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := blup.Theme.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

type mount struct {
	Source string      `toml:"source"`
	Target string      `toml:"target"`
	Files  []string    `toml:"files,omitempty"` // globs; a "! " prefix excludes
	Sites  *mountSites `toml:"sites,omitempty"`
}

// mountSites puts a mount's pages in some languages only (Hugo's sites
// matrix).
type mountSites struct {
	Matrix struct {
		Languages []string `toml:"languages"`
	} `toml:"matrix"`
}

func inLangs(langs ...string) *mountSites {
	m := &mountSites{}
	m.Matrix.Languages = langs
	return m
}

// locales are the locale of each site language.
var locales = map[string]string{"pt": "pt-BR", "en": "en-US", "es": "es"}

// labels are each language's name in itself, for the language switcher.
var labels = map[string]string{"pt": "Português", "en": "English", "es": "Español"}

// HugoConfig renders hugo.toml for the site (or its preview variant).
func HugoConfig(s *config.Site, preview bool) ([]byte, error) {
	locale := locales[s.Lang]
	if locale == "" {
		locale = "en-US"
	}
	folder := path.Clean(filepath.ToSlash(s.Folder))
	if folder == "." {
		folder = ""
	}

	// Obsidian shows a single line break as a break unless "strict line
	// breaks" is on; mirror that so pages look like the notes.
	hardWraps := true
	if strict, err := obsidian.StrictLineBreaks(s.Vault); err == nil && strict {
		hardWraps = false
	}

	// Hugo takes the first pattern that matches a file, so exclusions go
	// before the patterns that include.
	skip := []string{"! .*/**", "! **/.*/**"}
	// The guides for AI agents (Blup Studio's terminal) are never pages.
	for _, f := range AgentGuides {
		skip = append(skip, "! "+f)
	}
	// The authors folder holds profiles (photo, role, bio), read by the theme
	// as data from the assets mount, never published as pages.
	authors := obsidian.Sections(s.ContentDir())["authors"]
	assetFiles := append([]string{"! **/.*/**"}, globs(Attachments)...)
	if authors != "" {
		skip = append(skip, "! "+authors+"/**")
		// Profiles, and their translations (in a language's folder).
		assetFiles = append(assetFiles, path.Join(folder, authors)+"/*.md", path.Join(folder, authors)+"/*/*.md")
	}
	if t := obsidian.TemplatesFolder(s.Vault); t != "" {
		switch {
		case folder == "":
			skip = append(skip, "! "+t+"/**")
		case strings.HasPrefix(t+"/", folder+"/"):
			skip = append(skip, "! "+strings.TrimPrefix(t, folder+"/")+"/**")
		}
	}
	// Notes with "publish: false" or "private: true" stay out of the
	// published site like drafts: Hugo never reads them, so they are in no
	// page, list, feed, sitemap, search or tag. The preview shows them, as
	// it shows drafts.
	hidden := map[string]bool{}
	var quiet []string
	if !preview {
		hidden, quiet = unpublished(s)
	}
	var hide []string
	for _, rel := range sortedKeys(hidden) {
		hide = append(hide, "! "+literalGlob(rel))
	}
	files := append(append(hide, skip...), "*.md", "**/*.md")
	// A translated site: the folders named after its other languages hold
	// the translations, out of the site's own language.
	others := s.OtherLangs()
	if len(others) > 0 {
		var out []string
		for _, l := range others {
			out = append(out, "! "+l+"/**", "! **/"+l+"/**")
		}
		files = append(out, files...)
	}

	date := []string{"date", "publishDate", "created", ":filename", ":fileModTime"}
	if s.Model == "bento" {
		// Docs are ordered by weight and file name; modification times
		// must not reshuffle them.
		date = []string{"date", "publishDate", "created"}
	}

	mounts := []mount{{Source: s.ContentDir(), Target: "content", Files: files}}
	if len(others) > 0 {
		mounts[0].Sites = inLangs(s.Lang)
		mounts = append(mounts, translationMounts(s, others, skip, authors, hidden)...)
	}
	mounts = append(mounts,
		mount{Source: s.Vault, Target: "assets/vault", Files: assetFiles},
		mount{Source: "assets", Target: "assets"},
		mount{Source: "layouts", Target: "layouts"},
		mount{Source: "static", Target: "static"},
		mount{Source: "i18n", Target: "i18n"},
		mount{Source: "data", Target: "data"},
	)

	baseURL := "/"
	if s.Domain != "" {
		baseURL = "https://" + s.Domain + "/"
	}

	cfg := map[string]any{
		"baseURL":                baseURL,
		"title":                  s.Title,
		"locale":                 locale,
		"defaultContentLanguage": s.Lang,
		"theme":                  "blup",
		"removePathAccents":      true,
		"disableAliases":         true, // Obsidian "aliases" are alternative names, not old URLs
		"enableRobotsTXT":        true,
		"taxonomies":             map[string]string{"tag": "tags", "author": "authors"},
		"pagination":             map[string]any{"pagerSize": 10},
		"frontmatter": map[string]any{
			"date":    date,
			"lastmod": []string{"lastmod", "updated", "modified", ":fileModTime"},
		},
		"markup": map[string]any{
			"goldmark": map[string]any{
				"renderer": map[string]any{"hardWraps": hardWraps},
				// Block attributes align text: a {.text-center} line under a
				// paragraph (or after a heading) centers it.
				"parser": map[string]any{"attribute": map[string]any{"block": true, "title": true}},
				"extensions": map[string]any{
					"extras": map[string]any{"mark": map[string]any{"enable": true}},
					"passthrough": map[string]any{
						"enable": true,
						"delimiters": map[string]any{
							"block":  [][]string{{"$$", "$$"}},
							"inline": [][]string{{"$", "$"}},
						},
					},
				},
			},
			"tableOfContents": map[string]any{"startLevel": 2, "endLevel": 3},
		},
		"params": params(s, folder, authors, quiet),
		"module": map[string]any{
			"mounts": mounts,
		},
	}
	if len(others) > 0 {
		langs := map[string]any{}
		for i, l := range append([]string{s.Lang}, others...) {
			langs[l] = map[string]any{"weight": i + 1, "label": labels[l], "locale": locales[l]}
		}
		cfg["languages"] = langs
		delete(cfg, "locale")
	}
	// blup-locks.json lists the passwords of protected notes for the CLI,
	// which encrypts them and deletes the file (see internal/lock).
	formats := map[string]any{"bluplocks": map[string]any{
		"mediaType": "application/json", "baseName": "blup-locks", "isPlainText": true, "notAlternative": true,
	}}
	// manifest.json makes the site an installable app; the CLI adds the
	// icons and the offline copy after the build (internal/pwa).
	formats["blupmanifest"] = map[string]any{
		"mediaType": "application/json", "baseName": "manifest", "isPlainText": true, "notAlternative": true,
	}
	// feed.xml: a blog's podcast (the Podcast section's own outputs list
	// it), for Spotify, Apple Podcasts and other podcast apps.
	formats["podcast"] = map[string]any{
		"mediaType": "application/rss+xml", "baseName": "feed", "rel": "alternate",
	}
	home := []string{"html", "rss", "bluplocks", "blupmanifest"}
	if feedbackOn(s) {
		// Pages that get a feedback issue, read by the CLI (internal/app/feedback.go).
		formats["blupfeedback"] = map[string]any{
			"mediaType": "application/json", "baseName": "blup-feedback", "isPlainText": true, "notAlternative": true,
		}
		home = append(home, "blupfeedback")
	}
	if preview {
		// blup-map.json: the page of each note, so Blup Studio's preview
		// follows the note being edited.
		formats["blupmap"] = map[string]any{
			"mediaType": "application/json", "baseName": "blup-map", "isPlainText": true, "notAlternative": true,
		}
		home = append(home, "blupmap")
	}
	cfg["outputFormats"] = formats
	cfg["outputs"] = map[string]any{"home": home}
	if preview {
		mounts := cfg["module"].(map[string]any)["mounts"].([]mount)
		cfg["module"] = map[string]any{"mounts": append(mounts, mount{Source: PreviewStatic, Target: "static"})}
	}
	if s.Author != "" {
		// Notes without an "author" property belong to the site's author.
		cfg["cascade"] = []map[string]any{{
			"params":  map[string]any{},
			"authors": []string{s.Author},
			"target":  map[string]any{"kind": "page"},
		}}
	}
	body, err := toml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	header := fmt.Sprintf("# Generated by blup from %s. Edits here are overwritten:\n# change the site in %s or with the blup command.\n\n", config.SiteFile, config.SiteFile)
	return append([]byte(header), body...), nil
}

// translationMounts put each folder named after a translation language
// (Posts/en) in place of its parent (Posts) in that language's site: the
// note Posts/en/Note.md is the English Posts/Note.md, a translation of it.
func translationMounts(s *config.Site, langs, skip []string, authors string, hidden map[string]bool) []mount {
	root := s.ContentDir()
	var out []mount
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if p != root && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		// The authors' profiles (translated too) are data, never pages.
		if rel, _ := filepath.Rel(root, p); authors != "" && filepath.ToSlash(rel) == authors {
			return filepath.SkipDir
		}
		if !slices.Contains(langs, d.Name()) {
			return nil
		}
		parent, _ := filepath.Rel(root, filepath.Dir(p))
		target := path.Join("content", filepath.ToSlash(parent))
		var files []string
		rel, _ := filepath.Rel(root, p)
		prefix := filepath.ToSlash(rel) + "/"
		for _, h := range sortedKeys(hidden) {
			if strings.HasPrefix(h, prefix) {
				files = append(files, "! "+literalGlob(strings.TrimPrefix(h, prefix)))
			}
		}
		files = append(append(files, skip...), "*.md", "**/*.md")
		out = append(out, mount{Source: p, Target: target, Files: files, Sites: inLangs(d.Name())})
		// A language folder holds that language's notes, not more languages.
		return filepath.SkipDir
	})
	return out
}

func globs(exts []string) []string {
	out := make([]string, 0, len(exts)*4)
	for _, e := range exts {
		for _, x := range []string{e, strings.ToUpper(e)} {
			out = append(out, "*."+x, "**/*."+x)
		}
	}
	return out
}

func params(s *config.Site, folder, authors string, unpublished []string) map[string]any {
	// Name, description, logo, favicon, menu and social links are properties
	// of the site folder's _index note, read by the theme (site/settings.html).
	p := map[string]any{
		"model":       s.Model,
		"description": s.Description,
		"author":      s.Author,
		"blup":        blupParams(s, folder, authors, unpublished),
	}
	if config.ValidLogoLink(s.LogoLink) {
		// Where the logo leads (mochi/header.html, nori/header.html).
		p["logoLink"] = s.LogoLink
	}
	return p
}

func blupParams(s *config.Site, folder, authors string, unpublished []string) map[string]any {
	// attachments: the kinds of vault files a note may show or link to
	// (vault-resource.html); any other file is never published.
	p := map[string]any{"folder": folder, "authors": authorsPath(folder, authors), "attachments": Attachments}
	if len(unpublished) > 0 {
		// Links to these notes become plain text without a warning
		// (render-link.html). hugo.toml stays on this computer.
		p["unpublished"] = unpublished
	}
	if feedbackOn(s) {
		p["feedback"] = s.Feedback.Owner + "/" + s.Feedback.Repo
	}
	if s.Model == "bento" && s.Home != "" {
		// The section the home leads to (bento/home-target.html).
		p["home"] = s.Home
	}
	if c := cookieParams(s.Cookies); len(c) > 0 {
		p["cookies"] = c
	}
	return p
}

// cookieParams: the tools and the banner, for the theme (consent.html).
func cookieParams(c config.Cookies) map[string]any {
	p := map[string]any{}
	if c.Banner {
		p["banner"] = true
		if c.Privacy != "" {
			p["privacy"] = c.Privacy
		}
	}
	if c.GoogleAnalytics != "" {
		p["google_analytics"] = c.GoogleAnalytics
	}
	if c.TagManager != "" {
		p["google_tag_manager"] = c.TagManager
	}
	if c.MetaPixel != "" {
		p["meta_pixel"] = c.MetaPixel
	}
	if c.Umami != "" {
		p["umami"] = map[string]any{"id": c.Umami, "src": cmp.Or(c.UmamiScript, config.UmamiCloudScript)}
	}
	if c.Plausible != "" {
		p["plausible"] = map[string]any{"domain": c.Plausible, "src": cmp.Or(c.PlausibleScript, config.PlausibleScript)}
	}
	return p
}

// feedbackOn: votes, reactions and comments (both models).
func feedbackOn(s *config.Site) bool { return s.Feedback.On() }

// authorsPath is the authors folder relative to the vault, "" without one.
func authorsPath(folder, authors string) string {
	if authors == "" {
		return ""
	}
	return path.Join(folder, authors)
}
