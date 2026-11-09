package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/gabriellsalesx/blup/internal/config"
)

// The home screen's layout: sites and folders of sites, in the user's
// order. It is the Studio's own (studio.json next to Blup's settings) and
// changes nothing in the sites.

// HomeItem is a site, or a folder of sites, on the home screen.
type HomeItem struct {
	Site   string   `json:"site,omitempty"`   // a site's folder
	Folder string   `json:"folder,omitempty"` // a folder's name
	Sites  []string `json:"sites,omitempty"`  // the sites in the folder
}

func layoutFile() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "studio.json"), nil
}

// Layout is the home screen's order, with every known site in it: new
// sites at the end, sites that are gone left out.
func (s *Studio) Layout() ([]HomeItem, error) {
	g, err := config.LoadGlobal()
	if err != nil {
		return nil, err
	}
	known := g.ExistingSites()
	var saved []HomeItem
	if f, err := layoutFile(); err == nil {
		if b, err := os.ReadFile(f); err == nil {
			_ = json.Unmarshal(b, &saved)
		}
	}
	placed := map[string]bool{}
	out := []HomeItem{}
	for _, it := range saved {
		if it.Folder != "" {
			var sites []string
			for _, d := range it.Sites {
				if slices.Contains(known, d) && !placed[d] {
					placed[d] = true
					sites = append(sites, d)
				}
			}
			if len(sites) > 0 {
				out = append(out, HomeItem{Folder: it.Folder, Sites: sites})
			}
		} else if slices.Contains(known, it.Site) && !placed[it.Site] {
			placed[it.Site] = true
			out = append(out, it)
		}
	}
	for _, d := range known {
		if !placed[d] {
			out = append(out, HomeItem{Site: d})
		}
	}
	return out, nil
}

// SaveLayout saves the home screen's order.
func (s *Studio) SaveLayout(items []HomeItem) error {
	for _, it := range items {
		if it.Folder == "" && it.Site == "" {
			return errors.New("empty item")
		}
	}
	f, err := layoutFile()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(f, b, 0o644)
}

// Prefs are the Studio's own choices (studio-prefs.json).
type Prefs struct {
	Theme         string            `json:"theme"`       // "system", "light" or "dark"
	EditorFont    int               `json:"editorFont"`  // the editor's text size, in px
	EditorWidth   string            `json:"editorWidth"` // "narrow" (a readable line) or "wide"
	Spellcheck    bool              `json:"spellcheck"`
	PreviewOff    bool              `json:"previewOff"`    // open sites without the preview
	TermFont      int               `json:"termFont"`      // the terminal's text size, in px
	SpellLang     string            `json:"spellLang"`     // the spell check's dictionary: "pt", "en" or "es" ("" = the Studio's language)
	Words         []string          `json:"words"`         // words added to the dictionary
	SideWidth     int               `json:"sideWidth"`     // the files panel, in px (0 = the default)
	PreviewWidth  int               `json:"previewWidth"`  // the preview, in px (0 = half of the room)
	Tray          bool              `json:"tray"`          // minimized, the Studio goes to the tray
	ToggleKey     string            `json:"toggleKey"`     // global shortcut that shows and hides the Studio
	Keys          map[string]string `json:"keys"`          // the Studio's shortcuts changed by the user (action -> keys)
	ShortcutTip   bool              `json:"shortcutTip"`   // the tip about the global shortcut was answered
	LastSite      string            `json:"lastSite"`      // the site open when the Studio closed
	LastNotes     map[string]string `json:"lastNotes"`     // site folder -> the note open there last
	Recent        []string          `json:"recent"`        // sites opened lately, newest first (even ones taken out of the list)
	Voices        map[string]string `json:"voices"`        // the narration's voice for each site language ("pt", "en", "es"); missing = the default
	Welcomed      bool              `json:"welcomed"`      // the welcome screen (start, or bring the sites of an account) was seen
	CloudTip      bool              `json:"cloudTip"`      // the invitation to Blup Cloud, after the first site, was answered
	BackupDir     string            `json:"backupDir"`     // a daily copy of the open site's notes goes here ("" = off)
	TranslateAI   string            `json:"translateAI"`   // who translates: "local" (on the computer), "openrouter" or "blup", chosen by the person; "" (or the old "cloud") = local when installed, else ask
	RailNames     bool              `json:"railNames"`     // the side menu shows each screen's name next to its icon
	Notifications bool              `json:"notifications"` // the calendar's reminders (on the computer and, in Max and Ultra, by e-mail); on by default
	ErrorReports  string            `json:"errorReports"`  // error reports to Blup: "" (not asked yet), "on" or "off" (errors.go)
}

func prefsFile() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "studio-prefs.json"), nil
}

func defaultPrefs() Prefs {
	return Prefs{Theme: "system", EditorFont: 16, EditorWidth: "narrow", Spellcheck: true, TermFont: 13, ToggleKey: "Alt+Space", Notifications: true}
}

// prefsMu runs one change of the choices at a time: each reads the file,
// changes its part and writes it whole, so two at once would lose one.
var prefsMu sync.Mutex

// Prefs reads the Studio's choices. A damaged file (not JSON, null, a
// field of the wrong type) gives the defaults, never half of it.
func (s *Studio) Prefs() Prefs { return readPrefs() }

func readPrefs() Prefs {
	p := defaultPrefs()
	if f, err := prefsFile(); err == nil {
		if b, err := os.ReadFile(f); err == nil {
			read := defaultPrefs()
			if err := json.Unmarshal(b, &read); err == nil {
				p = read
			}
		}
	}
	return p.normalized()
}

// SavePrefs saves the Studio's choices. The recent sites are the Studio's
// own (remember), and so is the answer about error reports
// (SetErrorReports): the page's copy may be old, so the saved ones are kept.
func (s *Studio) SavePrefs(p Prefs) error {
	return updatePrefs(func(saved *Prefs) {
		recent, reports := saved.Recent, saved.ErrorReports
		*saved = p
		saved.Recent, saved.ErrorReports = recent, reports
	})
}

// updatePrefs changes the saved choices with fn, one change at a time.
func updatePrefs(fn func(*Prefs)) error {
	prefsMu.Lock()
	defer prefsMu.Unlock()
	p := readPrefs()
	fn(&p)
	return savePrefs(p.normalized())
}

// normalized puts each choice back in its range: a value out of it (an
// unknown theme, a text size of -5 or 9999) becomes the default.
func (p Prefs) normalized() Prefs {
	d := defaultPrefs()
	if p.ErrorReports != "on" && p.ErrorReports != "off" {
		p.ErrorReports = ""
	}
	if p.Theme != "system" && p.Theme != "light" && p.Theme != "dark" {
		p.Theme = d.Theme
	}
	if p.EditorWidth != "narrow" && p.EditorWidth != "wide" {
		p.EditorWidth = d.EditorWidth
	}
	if p.EditorFont < 13 || p.EditorFont > 22 {
		p.EditorFont = d.EditorFont
	}
	if p.TermFont < 11 || p.TermFont > 20 {
		p.TermFont = d.TermFont
	}
	if p.SpellLang != "" && p.SpellLang != "pt" && p.SpellLang != "en" && p.SpellLang != "es" {
		p.SpellLang = ""
	}
	// 0 is "the default"; a panel wider than any screen is a damaged value.
	if p.SideWidth < 0 || p.SideWidth > 4000 {
		p.SideWidth = 0
	}
	if p.PreviewWidth < 0 || p.PreviewWidth > 8000 {
		p.PreviewWidth = 0
	}
	return p
}

// savePrefs writes p whole (through updatePrefs, which holds prefsMu).
func savePrefs(p Prefs) error {
	f, err := prefsFile()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(f, b, 0o644)
}

// writeFileAtomic writes b to a file next to path and renames it there:
// a crash or a full disk leaves the old file, never half of the new one.
func writeFileAtomic(path string, b []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
