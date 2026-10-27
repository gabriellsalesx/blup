// Package scaffold writes a site folder: the generated hugo.toml (vault
// mounts included), the embedded theme and the folders for overrides.
package scaffold

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
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
	if err := copyBranding(s); err != nil {
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
	Source string   `toml:"source"`
	Target string   `toml:"target"`
	Files  []string `toml:"files,omitempty"` // globs; a "! " prefix excludes
}

// HugoConfig renders hugo.toml for the site (or its preview variant).
func HugoConfig(s *config.Site, preview bool) ([]byte, error) {
	locale := "en-US"
	if s.Lang == "pt" {
		locale = "pt-BR"
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

	files := []string{"*.md", "**/*.md", "! .*/**", "! **/.*/**"}
	if t := obsidian.TemplatesFolder(s.Vault); t != "" {
		switch {
		case folder == "":
			files = append(files, "! "+t+"/**")
		case strings.HasPrefix(t+"/", folder+"/"):
			files = append(files, "! "+strings.TrimPrefix(t, folder+"/")+"/**")
		}
	}

	date := []string{"date", "publishDate", "created", ":filename", ":fileModTime"}
	if s.Model == "docs" {
		// Docs are ordered by weight and file name; modification times
		// must not reshuffle them.
		date = []string{"date", "publishDate", "created"}
	}

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
		"params": params(s, folder),
		"module": map[string]any{
			"mounts": []mount{
				{Source: s.ContentDir(), Target: "content", Files: files},
				{Source: s.Vault, Target: "assets/vault", Files: append(globs(Attachments), "! **/.*/**")},
				{Source: "assets", Target: "assets"},
				{Source: "layouts", Target: "layouts"},
				{Source: "static", Target: "static"},
				{Source: "i18n", Target: "i18n"},
				{Source: "data", Target: "data"},
			},
		},
	}
	if preview {
		mounts := cfg["module"].(map[string]any)["mounts"].([]mount)
		cfg["module"] = map[string]any{"mounts": append(mounts, mount{Source: PreviewStatic, Target: "static"})}
	}
	if s.Model == "docs" {
		cfg["disableKinds"] = []string{"rss"}
	}
	if s.Author != "" {
		// Notes without an "author" property belong to the site's author.
		cfg["cascade"] = []map[string]any{{
			"params":  map[string]any{},
			"authors": []string{s.Author},
			"target":  map[string]any{"kind": "page"},
		}}
	}
	if menu := menuEntries(s.Menu); len(menu) > 0 {
		cfg["menus"] = map[string]any{"main": menu}
	}
	body, err := toml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	header := fmt.Sprintf("# Generated by blup from %s. Edits here are overwritten:\n# change the site in %s or with the blup command.\n\n", config.SiteFile, config.SiteFile)
	return append([]byte(header), body...), nil
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

// Branding files the user picked are copied into assets/blup/ so the
// theme can resize them; params point at the copies.
const brandingDir = "blup"

func copyBranding(s *config.Site) error {
	dir := filepath.Join(s.Dir(), "assets", brandingDir)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	for name, src := range map[string]string{"logo": s.Logo, "avatar": s.Avatar} {
		if src == "" {
			continue
		}
		b, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		dest := filepath.Join(dir, name+strings.ToLower(filepath.Ext(src)))
		if err := os.WriteFile(dest, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func brandingAsset(name, src string) string {
	if src == "" {
		return ""
	}
	return path.Join(brandingDir, name+strings.ToLower(filepath.Ext(src)))
}

func params(s *config.Site, folder string) map[string]any {
	p := map[string]any{
		"model":       s.Model,
		"description": s.Description,
		"author":      s.Author,
		"authorBio":   s.AuthorBio,
		"logo":        brandingAsset("logo", s.Logo),
		"avatar":      brandingAsset("avatar", s.Avatar),
		"blup":        map[string]any{"folder": folder},
	}
	var social []map[string]any
	for _, so := range s.Social {
		if so.URL == "" {
			continue
		}
		icon := so.Network
		if icon == "x" {
			icon = "x-social"
		}
		url := so.URL
		if so.Network == "mail" && !strings.HasPrefix(url, "mailto:") {
			url = "mailto:" + url
		}
		social = append(social, map[string]any{"name": SocialNames[so.Network], "icon": icon, "url": url})
	}
	if len(social) > 0 {
		p["social"] = social
	}
	return p
}

// SocialNames are the networks the footer has icons for.
var SocialNames = map[string]string{
	"github": "GitHub", "x": "X", "instagram": "Instagram", "linkedin": "LinkedIn",
	"youtube": "YouTube", "facebook": "Facebook", "whatsapp": "WhatsApp", "mail": "E-mail",
}

// SocialOrder is the order networks are asked and shown.
var SocialOrder = []string{"instagram", "x", "linkedin", "youtube", "github", "facebook", "whatsapp", "mail"}

func menuEntries(items []config.MenuItem) []map[string]any {
	var out []map[string]any
	for i, it := range items {
		e := map[string]any{"name": it.Name, "weight": (i + 1) * 10, "identifier": fmt.Sprintf("m%d", i)}
		switch {
		case it.Page != "":
			e["pageRef"] = it.Page
		case it.URL != "":
			e["url"] = it.URL
		default:
			continue
		}
		out = append(out, e)
	}
	return out
}
