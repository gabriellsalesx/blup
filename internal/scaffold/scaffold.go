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

	// Hugo takes the first pattern that matches a file, so exclusions go
	// before the patterns that include.
	skip := []string{"! .*/**", "! **/.*/**"}
	// The authors folder holds profiles (photo, role, bio), read by the theme
	// as data from the assets mount, never published as pages.
	authors := obsidian.Sections(s.ContentDir())["authors"]
	assetFiles := append([]string{"! **/.*/**"}, globs(Attachments)...)
	if authors != "" {
		skip = append(skip, "! "+authors+"/**")
		assetFiles = append(assetFiles, path.Join(folder, authors)+"/*.md")
	}
	if t := obsidian.TemplatesFolder(s.Vault); t != "" {
		switch {
		case folder == "":
			skip = append(skip, "! "+t+"/**")
		case strings.HasPrefix(t+"/", folder+"/"):
			skip = append(skip, "! "+strings.TrimPrefix(t, folder+"/")+"/**")
		}
	}
	files := append(skip, "*.md", "**/*.md")

	date := []string{"date", "publishDate", "created", ":filename", ":fileModTime"}
	if s.Model == "bento" {
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
		"params": params(s, folder, authors),
		"module": map[string]any{
			"mounts": []mount{
				{Source: s.ContentDir(), Target: "content", Files: files},
				{Source: s.Vault, Target: "assets/vault", Files: assetFiles},
				{Source: "assets", Target: "assets"},
				{Source: "layouts", Target: "layouts"},
				{Source: "static", Target: "static"},
				{Source: "i18n", Target: "i18n"},
				{Source: "data", Target: "data"},
			},
		},
	}
	// blup-locks.json lists the passwords of protected notes for the CLI,
	// which encrypts them and deletes the file (see internal/lock).
	cfg["outputFormats"] = map[string]any{"bluplocks": map[string]any{
		"mediaType": "application/json", "baseName": "blup-locks", "isPlainText": true, "notAlternative": true,
	}}
	cfg["outputs"] = map[string]any{"home": []string{"html", "rss", "bluplocks"}}
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

func globs(exts []string) []string {
	out := make([]string, 0, len(exts)*4)
	for _, e := range exts {
		for _, x := range []string{e, strings.ToUpper(e)} {
			out = append(out, "*."+x, "**/*."+x)
		}
	}
	return out
}

func params(s *config.Site, folder, authors string) map[string]any {
	// Name, description, logo, favicon, menu and social links are properties
	// of the site folder's _index note, read by the theme (site/settings.html).
	return map[string]any{
		"model":       s.Model,
		"description": s.Description,
		"author":      s.Author,
		"blup":        map[string]any{"folder": folder, "authors": authorsPath(folder, authors)},
	}
}

// authorsPath is the authors folder relative to the vault, "" without one.
func authorsPath(folder, authors string) string {
	if authors == "" {
		return ""
	}
	return path.Join(folder, authors)
}
