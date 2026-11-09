package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/notesync"
	"github.com/pelletier/go-toml/v2"
)

// Bringing a site to another computer: the sync keeps the site's settings
// next to the notes (notesync.SiteFile, without this computer's folders and
// the password) and marks the repository with a topic. On a new computer,
// Blup Studio lists the repositories with that topic and brings a site back
// whole: its notes and its settings.

// SiteTopic marks the private notes repositories of Blup sites.
const SiteTopic = "blup-site"

// sharedSite is what of blup.toml travels with the notes. Where the site is
// published (Deploy: provider, accounts, project) and its feedback
// repository are not in it: they stay in this computer's blup.toml, so
// whoever can write to the notes cannot redirect another computer's
// publishing. Files written by older versions carry them; they are
// ignored.
type sharedSite struct {
	Version     int    `toml:"version"`
	Title       string `toml:"title"`
	Description string `toml:"description"`
	Author      string `toml:"author"`
	Lang        string `toml:"lang"`
	Model       string `toml:"model"`
	Kind        string `toml:"kind,omitempty"` // "notes" for a vault of notes
	Home        string `toml:"home,omitempty"` // Bento: the section the home leads to
	LogoLink    string `toml:"logo_link,omitempty"`
	// Translations: the site's other languages. nil (a file from an older
	// version, without the key) keeps this computer's.
	Translations *[]string      `toml:"translations"`
	Cookies      config.Cookies `toml:"cookies,omitempty"`
}

func shareOf(s *config.Site) sharedSite {
	langs := s.OtherLangs()
	if langs == nil {
		langs = []string{}
	}
	return sharedSite{
		Version: config.SiteVersion, Title: s.Title, Description: s.Description, Author: s.Author, Lang: s.Lang,
		Model: s.Model, Kind: s.Kind, Home: s.Home, LogoLink: s.LogoLink, Translations: &langs, Cookies: s.Cookies,
	}
}

// apply puts settings from another computer on s. Deploy and Feedback
// are never taken from there.
func (sh sharedSite) apply(s *config.Site) {
	s.Title, s.Description, s.Author, s.Lang, s.Model, s.Kind = sh.Title, sh.Description, sh.Author, sh.Lang, sh.Model, sh.Kind
	s.Cookies, s.Home = sh.Cookies, sh.Home
	// From another computer: only a valid link (else the home page).
	s.LogoLink = ""
	if config.ValidLogoLink(sh.LogoLink) {
		s.LogoLink = sh.LogoLink
	}
	if sh.Translations != nil {
		// Only known languages (they name folders and mount patterns).
		var langs []string
		for _, l := range *sh.Translations {
			if slices.Contains(config.SiteLangs, l) && l != s.Lang && !slices.Contains(langs, l) {
				langs = append(langs, l)
			}
		}
		s.Translations = langs
	}
}

func parseShared(b []byte) (sharedSite, error) {
	var sh sharedSite
	if err := toml.Unmarshal(b, &sh); err != nil {
		return sh, err
	}
	if sh.Version > config.SiteVersion {
		return sh, config.ErrNewerSite
	}
	if strings.TrimSpace(sh.Title) == "" {
		return sh, errors.New("site.toml: no title")
	}
	return sh, nil
}

func sharedPath(s *config.Site) string {
	return filepath.Join(s.ContentDir(), filepath.FromSlash(notesync.SiteFile))
}

// writeShared puts the site's settings next to the notes (only when they
// changed, so a sync doesn't send it for nothing).
func writeShared(s *config.Site) error {
	b, err := toml.Marshal(shareOf(s))
	if err != nil {
		return err
	}
	p := sharedPath(s)
	if old, err := os.ReadFile(p); err == nil && bytes.Equal(old, b) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

// readShared applies settings that came from another computer. s is
// never changed (another goroutine may be reading it): the settings go on
// a copy, saved and returned, and whoever holds the site (OnSiteSettings)
// swaps it in.
func readShared(s *config.Site) (*config.Site, error) {
	b, err := os.ReadFile(sharedPath(s))
	if err != nil {
		return s, err
	}
	sh, err := parseShared(b)
	if err != nil {
		return s, err
	}
	fresh := *s
	fresh.Translations = slices.Clone(s.Translations)
	sh.apply(&fresh)
	if err := fresh.Save(); err != nil {
		return s, err
	}
	siteSettings.Lock()
	fn := siteSettings.fn
	siteSettings.Unlock()
	if fn != nil {
		fn(&fresh)
	}
	return &fresh, nil
}

// siteSettings is who hears of settings that came from another computer.
var siteSettings struct {
	sync.Mutex
	fn func(*config.Site)
}

// OnSiteSettings makes fn hear each site whose settings came from another
// computer with a sync (the site as it is now, a new value: the old one
// is left as it was). nil stops it.
func OnSiteSettings(fn func(*config.Site)) {
	siteSettings.Lock()
	siteSettings.fn = fn
	siteSettings.Unlock()
}

// CloudSite is a Blup site found on the user's GitHub.
type CloudSite struct {
	Owner  string    `json:"owner"`
	Repo   string    `json:"repo"`
	Title  string    `json:"title"`
	Model  string    `json:"model"`
	Lang   string    `json:"lang"`
	Pushed time.Time `json:"pushed"`
	// Dir is the site's folder when it is already on this computer.
	Dir string `json:"dir"`
}

// FindCloudSites lists the Blup sites in the user's GitHub repositories.
func FindCloudSites(ctx context.Context) ([]CloudSite, error) {
	c, _, err := savedGitHub(ctx)
	if err != nil {
		return nil, err
	}
	repos, err := c.OwnRepos(ctx)
	if err != nil {
		return nil, errors.New(describe(err))
	}
	here := localSyncs()
	var (
		out []CloudSite
		mu  sync.Mutex
		wg  sync.WaitGroup
	)
	sem := make(chan struct{}, 6)
	for _, r := range repos {
		// The topic needs Administration, which keys limited to chosen
		// repositories don't have: a private "-notas" repository is
		// looked at too (only its .blup/site.toml decides).
		if !r.Private || !(hasTopic(r.Topics, SiteTopic) || notesRepoName(r.Name)) {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			b, err := c.FileContent(ctx, r.Owner, r.Name, notesync.SiteFile)
			if err != nil || b == nil {
				return
			}
			sh, err := parseShared(b)
			if err != nil {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			out = append(out, CloudSite{
				Owner: r.Owner, Repo: r.Name, Title: sh.Title, Model: sh.Model, Lang: sh.Lang, Pushed: r.Pushed,
				Dir: here[strings.ToLower(r.Owner+"/"+r.Name)],
			})
		}()
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool { return out[i].Pushed.After(out[j].Pushed) })
	return out, nil
}

func hasTopic(topics []string, t string) bool {
	for _, x := range topics {
		if x == t {
			return true
		}
	}
	return false
}

// localSyncs maps "owner/repo" (lowercase) to the folder of the site on
// this computer synced with it.
func localSyncs() map[string]string {
	out := map[string]string{}
	g, err := config.LoadGlobal()
	if err != nil {
		return out
	}
	for _, dir := range g.ExistingSites() {
		if s, err := config.LoadSite(dir); err == nil && s.Sync.Repo != "" {
			out[strings.ToLower(s.Sync.Owner+"/"+s.Sync.Repo)] = dir
		}
	}
	return out
}

// BringSite makes a site found on GitHub on this computer: new folders for
// its notes and its files, the notes downloaded, its settings applied. The
// site joins the list.
func BringSite(ctx context.Context, owner, repo string) (*config.Site, error) {
	if dir := localSyncs()[strings.ToLower(owner+"/"+repo)]; dir != "" {
		return config.LoadSite(dir)
	}
	c, _, err := savedGitHub(ctx)
	if err != nil {
		return nil, err
	}
	b, err := c.FileContent(ctx, owner, repo, notesync.SiteFile)
	if err != nil {
		return nil, errors.New(describe(err))
	}
	if b == nil {
		return nil, errors.New(i18n.T("bring.not_a_site", owner+"/"+repo))
	}
	sh, err := parseShared(b)
	if err != nil {
		return nil, err
	}
	notes, dir := freeDir(DefaultNotesDir(sh.Title)), freeDir(DefaultSiteDir(sh.Title))
	for _, d := range []string{notes, dir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	site := config.NewSite(dir)
	sh.apply(site)
	site.Vault, site.Folder = notes, ""
	site.Sync = config.NoteSync{Owner: owner, Repo: repo}
	if err := site.Save(); err != nil {
		return nil, err
	}
	// The first sync only receives: the settings come with the notes.
	if _, err := syncSite(ctx, site, false); err != nil {
		return nil, err
	}
	if site, err = readShared(site); err != nil {
		return nil, err
	}
	if err := RememberSite(dir); err != nil {
		return nil, err
	}
	return site, nil
}

// freeDir is dir, or dir with a number when it already holds something.
func freeDir(dir string) string {
	free := func(d string) bool {
		entries, err := os.ReadDir(d)
		return err != nil || len(entries) == 0
	}
	if free(dir) {
		return dir
	}
	sep := "-" // ~/Sites/my-site-2
	if strings.Contains(filepath.Base(dir), " ") {
		sep = " " // ~/Documents/Blup/My site 2
	}
	for n := 2; ; n++ {
		if d := fmt.Sprintf("%s%s%d", dir, sep, n); free(d) {
			return d
		}
	}
}
