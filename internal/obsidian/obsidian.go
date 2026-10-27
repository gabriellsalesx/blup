// Package obsidian finds the user's vaults and adjusts the few vault settings
// that make notes readable by Hugo (standard Markdown links, relative paths).
package obsidian

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Vault is a vault registered in the Obsidian app.
type Vault struct {
	Name string
	Path string
}

// Vaults lists the vaults the Obsidian app knows about, in every install
// flavour we can find (native, Flatpak, Snap).
func Vaults() []Vault {
	seen := map[string]bool{}
	var out []Vault
	for _, f := range registryFiles() {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var reg struct {
			Vaults map[string]struct {
				Path string `json:"path"`
				TS   int64  `json:"ts"`
			} `json:"vaults"`
		}
		if json.Unmarshal(b, &reg) != nil {
			continue
		}
		type entry struct {
			path string
			ts   int64
		}
		var list []entry
		for _, v := range reg.Vaults {
			list = append(list, entry{v.Path, v.TS})
		}
		sort.Slice(list, func(i, j int) bool { return list[i].ts > list[j].ts })
		for _, e := range list {
			if seen[e.path] || !IsVault(e.path) {
				continue
			}
			seen[e.path] = true
			out = append(out, Vault{Name: filepath.Base(e.path), Path: e.path})
		}
	}
	return out
}

func registryFiles() []string {
	var files []string
	if dir, err := os.UserConfigDir(); err == nil {
		files = append(files, filepath.Join(dir, "obsidian", "obsidian.json"))
	}
	if runtime.GOOS == "linux" {
		if home, err := os.UserHomeDir(); err == nil {
			files = append(files,
				filepath.Join(home, ".var/app/md.obsidian.Obsidian/config/obsidian/obsidian.json"),
				filepath.Join(home, "snap/obsidian/current/.config/obsidian/obsidian.json"),
			)
		}
	}
	return files
}

// IsVault reports whether dir looks like an Obsidian vault.
func IsVault(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, ".obsidian"))
	return err == nil && st.IsDir()
}

// Folder is a folder inside a vault.
type Folder struct {
	Path  string // relative to the vault, with forward slashes
	Notes int    // Markdown files inside it, recursively
}

// Folders lists the vault folders (up to depth levels) that contain notes,
// skipping hidden folders. Paths use forward slashes.
func Folders(vault string, depth int) []Folder {
	counts := map[string]int{}
	_ = filepath.WalkDir(vault, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && p != vault && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(vault, filepath.Dir(p))
		parts := strings.Split(filepath.ToSlash(rel), "/")
		for i := 1; i <= len(parts) && i <= depth; i++ {
			if parts[0] == "." {
				break
			}
			counts[strings.Join(parts[:i], "/")]++
		}
		return nil
	})
	out := make([]Folder, 0, len(counts))
	for p, n := range counts {
		out = append(out, Folder{Path: p, Notes: n})
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path) })
	return out
}

// CountNotes counts the Markdown files under dir, skipping hidden folders.
func CountNotes(dir string) int {
	n := 0
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && p != dir && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".md") {
			n++
		}
		return nil
	})
	return n
}

// LinkSettings is what we change in .obsidian/app.json.
type LinkSettings struct {
	UseMarkdownLinks bool
	NewLinkFormat    string
	AttachmentFolder string
}

// ReadLinkSettings reads the current link settings of a vault.
func ReadLinkSettings(vault string) (LinkSettings, error) {
	m, err := readJSON(filepath.Join(vault, ".obsidian", "app.json"))
	if err != nil {
		return LinkSettings{}, err
	}
	s := LinkSettings{NewLinkFormat: "shortest"}
	if v, ok := m["useMarkdownLinks"].(bool); ok {
		s.UseMarkdownLinks = v
	}
	if v, ok := m["newLinkFormat"].(string); ok {
		s.NewLinkFormat = v
	}
	if v, ok := m["attachmentFolderPath"].(string); ok {
		s.AttachmentFolder = v
	}
	return s, nil
}

// UsesHugoFriendlyLinks reports whether new links are already standard,
// relative Markdown links.
func (s LinkSettings) UsesHugoFriendlyLinks() bool {
	return s.UseMarkdownLinks && s.NewLinkFormat == "relative"
}

// EnableHugoFriendlyLinks makes Obsidian write standard relative Markdown
// links. The previous app.json is kept as app.json.blup-backup. Every
// other setting is preserved. Obsidian must be reopened to pick it up.
func EnableHugoFriendlyLinks(vault string) error {
	path := filepath.Join(vault, ".obsidian", "app.json")
	m, err := readJSON(path)
	if err != nil {
		return err
	}
	if orig, err := os.ReadFile(path); err == nil {
		if err := os.WriteFile(path+".blup-backup", orig, 0o644); err != nil {
			return err
		}
	}
	m["useMarkdownLinks"] = true
	m["newLinkFormat"] = "relative"
	return writeJSON(path, m)
}

// TemplatesFolder returns the folder configured in the core Templates plugin,
// or "" if none.
func TemplatesFolder(vault string) string {
	m, err := readJSON(filepath.Join(vault, ".obsidian", "templates.json"))
	if err != nil {
		return ""
	}
	f, _ := m["folder"].(string)
	return strings.Trim(f, "/")
}

// SetTemplatesFolder points the core Templates plugin at folder and turns
// the plugin on.
func SetTemplatesFolder(vault, folder string) error {
	path := filepath.Join(vault, ".obsidian", "templates.json")
	m, err := readJSON(path)
	if err != nil {
		return err
	}
	m["folder"] = folder
	if err := writeJSON(path, m); err != nil {
		return err
	}
	core := filepath.Join(vault, ".obsidian", "core-plugins.json")
	b, err := os.ReadFile(core)
	if errors.Is(err, os.ErrNotExist) {
		return nil // fresh vault: Obsidian enables core plugins by default
	}
	if err != nil {
		return err
	}
	// Two historical formats: a list of enabled ids, or a map id → bool.
	var list []string
	if json.Unmarshal(b, &list) == nil {
		for _, id := range list {
			if id == "templates" {
				return nil
			}
		}
		list = append(list, "templates")
		return writeJSON(core, list)
	}
	var mp map[string]any
	if err := json.Unmarshal(b, &mp); err != nil {
		return err
	}
	mp["templates"] = true
	return writeJSON(core, mp)
}

// WriteNoteIfMissing creates a note (and its folders) unless it exists.
func WriteNoteIfMissing(path, body string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(body), 0o644)
}

func readJSON(path string) (map[string]any, error) {
	m := map[string]any{}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return m, nil
	}
	return m, json.Unmarshal(b, &m)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// StrictLineBreaks reports the vault's "strict line breaks" setting.
func StrictLineBreaks(vault string) (bool, error) {
	m, err := readJSON(filepath.Join(vault, ".obsidian", "app.json"))
	if err != nil {
		return false, err
	}
	v, _ := m["strictLineBreaks"].(bool)
	return v, nil
}
