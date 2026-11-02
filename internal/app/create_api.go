package app

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/scaffold"
)

// Creating a site without questions, for Blup Studio's welcome screens:
// the same steps as "blup new" (newSite), with the answers already given.

// NewSiteOptions are the answers of the new site screens.
type NewSiteOptions struct {
	Vault       string   `json:"vault"`    // an Obsidian vault, or any folder of notes
	Folder      string   `json:"folder"`   // the site's folder inside it ("" for all of it); created when missing
	Dir         string   `json:"dir"`      // where blup.toml and the site's files go
	Model       string   `json:"model"`    // mochi or bento
	Sections    []string `json:"sections"` // Bento's parts
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Lang        string   `json:"lang"`        // pt or en
	AdjustLinks bool     `json:"adjustLinks"` // switch an Obsidian vault to links that work on the web
}

// DefaultNotesDir is a new folder for the notes, in Documents.
func DefaultNotesDir(title string) string {
	home, _ := os.UserHomeDir()
	docs := filepath.Join(home, "Documents")
	if _, err := os.Stat(docs); err != nil {
		docs = home
	}
	return filepath.Join(docs, "Blup", firstOf(strings.TrimSpace(title), "site"))
}

// DefaultSiteDir is where a new site's files go (~/Sites/<name>).
func DefaultSiteDir(title string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, i18n.T("new.sites_folder"), Slug(firstOf(title, "site")))
}

// CheckSiteDir tells why dir can't hold a new site ("" when it can).
func CheckSiteDir(dir string) string {
	dir = expandHome(dir)
	if fileExists(filepath.Join(dir, config.SiteFile)) {
		return i18n.T("new.site_dir_exists")
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return i18n.T("new.site_dir_not_empty")
	}
	return ""
}

// VaultLinksOK reports whether a folder is a plain folder or an Obsidian
// vault already saving links that work on the web.
func VaultLinksOK(dir string) bool {
	if !obsidian.IsVault(dir) {
		return true
	}
	s, err := obsidian.ReadLinkSettings(dir)
	return err != nil || s.UsesHugoFriendlyLinks()
}

// CreateSite makes the site: notes (the example site when the folder has
// no notes), settings and generated files; it adds the site to the list.
func CreateSite(g *config.Global, o NewSiteOptions) (*config.Site, error) {
	o.Title = strings.TrimSpace(o.Title)
	if o.Title == "" {
		return nil, errors.New(i18n.T("required"))
	}
	if o.Model != "bento" {
		if o.Model != "nori" {
			o.Model = "mochi"
		}
		o.Sections = nil
	} else if len(o.Sections) == 0 {
		return nil, errors.New(i18n.T("new.sections_none"))
	}
	for _, k := range o.Sections {
		if !slices.Contains(docsKinds, k) {
			return nil, errors.New("unknown section " + k)
		}
	}
	if o.Lang != "pt" && o.Lang != "es" {
		o.Lang = "en"
	}
	o.Vault, o.Dir = expandHome(o.Vault), expandHome(o.Dir)
	o.Folder = strings.Trim(filepath.ToSlash(strings.TrimSpace(o.Folder)), "/")
	if o.Vault == "" || o.Dir == "" || strings.Contains(o.Folder, "..") {
		return nil, errors.New(i18n.T("required"))
	}
	if why := CheckSiteDir(o.Dir); why != "" {
		return nil, errors.New(why)
	}
	notes := filepath.Join(o.Vault, filepath.FromSlash(o.Folder))
	if err := os.MkdirAll(notes, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(o.Dir, 0o755); err != nil {
		return nil, err
	}

	site := config.NewSite(o.Dir)
	site.Title, site.Description, site.Author = o.Title, strings.TrimSpace(o.Description), strings.TrimSpace(o.Author)
	site.Lang, site.Model, site.Vault, site.Folder = o.Lang, o.Model, o.Vault, o.Folder
	if o.AdjustLinks && obsidian.IsVault(o.Vault) {
		if err := obsidian.EnableHugoFriendlyLinks(o.Vault); err != nil {
			return nil, err
		}
	}
	if obsidian.CountNotes(notes) == 0 {
		if err := writeSamples(site, o.Sections); err != nil {
			return nil, err
		}
	} else {
		if err := ensureHome(site); err != nil {
			return nil, err
		}
		for _, kind := range o.Sections {
			if err := writeSection(site, kind); err != nil {
				return nil, err
			}
		}
	}
	if err := writeSiteNotes(site); err != nil {
		return nil, err
	}
	// No Obsidian templates folder here: the Studio makes its notes itself,
	// and the folder only confused the tree (the CLI still writes it).
	if err := scaffold.Write(site); err != nil {
		return nil, err
	}
	if err := site.Save(); err != nil {
		return nil, err
	}
	g.AddSite(o.Dir)
	return site, g.Save()
}
