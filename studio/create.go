package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The new site screens: the questions of "blup new", as steps in the window.

// Vault is an Obsidian vault found on the computer.
type Vault struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Vaults lists the Obsidian vaults on this computer.
func (s *Studio) Vaults() []Vault {
	out := []Vault{}
	for _, v := range obsidian.Vaults() {
		out = append(out, Vault{v.Name, v.Path})
	}
	return out
}

// Places are the suggested folders of a new site.
type Places struct {
	Notes string `json:"notes"`
	Dir   string `json:"dir"`
}

// SuggestPlaces suggests where a site named title keeps its notes and files.
func (s *Studio) SuggestPlaces(title string) Places {
	return Places{app.DefaultNotesDir(title), app.DefaultSiteDir(title)}
}

// VaultFolders lists the top folders of a vault (folders of the notes).
func (s *Studio) VaultFolders(vault string) []string {
	out := []string{}
	for _, f := range obsidian.Folders(vault, 1) {
		out = append(out, f.Path)
	}
	return out
}

// CheckSiteDir tells why a folder can't hold a new site ("" when it can).
func (s *Studio) CheckSiteDir(dir string) string { return app.CheckSiteDir(dir) }

// NeedsLinkAdjust reports whether a folder is an Obsidian vault that saves
// [[wikilinks]] (which the web doesn't understand).
func (s *Studio) NeedsLinkAdjust(dir string) bool { return !app.VaultLinksOK(dir) }

// PickFolder asks for a folder ("" when the user cancels).
func (s *Studio) PickFolder() (string, error) {
	return application.Get().Dialog.OpenFile().SetTitle("Blup Studio").
		CanChooseDirectories(true).CanChooseFiles(false).CanCreateDirectories(true).PromptForSingleSelection()
}

// CreateSite makes the new site and opens it.
func (s *Studio) CreateSite(o app.NewSiteOptions) (SiteInfo, error) {
	g, err := config.LoadGlobal()
	if err != nil {
		return SiteInfo{}, err
	}
	site, err := app.CreateSite(&g, o)
	if err != nil {
		return SiteInfo{}, err
	}
	return info(site), nil
}

// HomeDir is the user's home folder, to show paths as "~/...".
func (s *Studio) HomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

// Found is what OpenExisting found in the folder the user picked.
type Found struct {
	Site   *SiteInfo `json:"site"`   // the site, now in Blup's list
	Folder string    `json:"folder"` // the folder picked, when no site matched
}

// OpenExisting asks for a folder (a site's own folder or its notes) and
// finds its Blup site. Without one, Folder lets the user make a site of
// those notes. Nil when the user cancels.
func (s *Studio) OpenExisting() (*Found, error) {
	dir, err := s.PickFolder()
	if err != nil || dir == "" {
		return nil, err
	}
	siteDir := app.FindSite(dir)
	if siteDir == "" {
		return &Found{Folder: dir}, nil
	}
	site, err := config.LoadSite(siteDir)
	if err != nil {
		return nil, err
	}
	if err := app.RememberSite(siteDir); err != nil {
		return nil, err
	}
	i := info(site)
	return &Found{Site: &i}, nil
}

// SiteParts lists what the site in dir is made of, for the delete window.
func (s *Studio) SiteParts(dir string) (app.SiteParts, error) { return app.PartsOf(dir) }

// ForgetSite takes a site out of the list without deleting anything.
func (s *Studio) ForgetSite(dir string) error {
	s.closeIfOpen(dir)
	return app.ForgetSite(dir)
}

// DeleteSite deletes the parts chosen of the site in dir (folders to the
// system's trash; online parts for good). Parts that couldn't be deleted
// come back with a link to do it by hand.
func (s *Studio) DeleteSite(dir string, choice app.DeleteChoice) ([]app.DeleteProblem, error) {
	s.closeIfOpen(dir)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return app.DeleteSite(ctx, dir, choice)
}

// closeIfOpen closes the site in dir when it is the open one (its preview,
// terminals and file watcher hold its folders).
func (s *Studio) closeIfOpen(dir string) {
	s.mu.Lock()
	open := s.site != nil && filepath.Clean(s.site.Dir()) == filepath.Clean(dir)
	s.mu.Unlock()
	if !open {
		return
	}
	s.StopPreview()
	closeTerminals(s)
	watchNotes("")
	s.startLive(nil)
	s.mu.Lock()
	s.site = nil
	s.mu.Unlock()
}

// RecentSite is a site opened lately; Listed tells if it is in the list.
type RecentSite struct {
	SiteInfo
	Listed bool `json:"listed"`
}

const maxRecent = 12

// remember puts dir first in the recent sites.
func (s *Studio) remember(dir string) {
	p := s.Prefs()
	p.Recent = append([]string{dir}, slices.DeleteFunc(p.Recent, func(d string) bool { return d == dir })...)
	if len(p.Recent) > maxRecent {
		p.Recent = p.Recent[:maxRecent]
	}
	_ = savePrefs(p)
}

// RecentSites lists the sites opened lately that still exist, newest
// first, to find one taken out of the list.
func (s *Studio) RecentSites() []RecentSite {
	g, _ := config.LoadGlobal()
	out := []RecentSite{}
	for _, dir := range s.Prefs().Recent {
		site, err := config.LoadSite(dir)
		if err != nil {
			continue
		}
		out = append(out, RecentSite{SiteInfo: info(site), Listed: slices.Contains(g.Sites, dir)})
	}
	return out
}

// RestoreSite puts a site back in the list (after "remove from the list").
func (s *Studio) RestoreSite(dir string) (SiteInfo, error) {
	site, err := config.LoadSite(dir)
	if err != nil {
		return SiteInfo{}, err
	}
	return info(site), app.RememberSite(dir)
}
