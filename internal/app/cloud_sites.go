package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// sharedSite is what of blup.toml travels with the notes.
type sharedSite struct {
	Version     int             `toml:"version"`
	Title       string          `toml:"title"`
	Description string          `toml:"description"`
	Author      string          `toml:"author"`
	Lang        string          `toml:"lang"`
	Model       string          `toml:"model"`
	Domain      string          `toml:"domain,omitempty"`
	Deploy      config.Deploy   `toml:"deploy"`
	Feedback    config.Feedback `toml:"feedback,omitempty"`
	Cookies     config.Cookies  `toml:"cookies,omitempty"`
}

func shareOf(s *config.Site) sharedSite {
	return sharedSite{
		Version: config.SiteVersion, Title: s.Title, Description: s.Description, Author: s.Author, Lang: s.Lang,
		Model: s.Model, Domain: s.Domain, Deploy: s.Deploy, Feedback: s.Feedback, Cookies: s.Cookies,
	}
}

func (sh sharedSite) apply(s *config.Site) {
	s.Title, s.Description, s.Author, s.Lang, s.Model = sh.Title, sh.Description, sh.Author, sh.Lang, sh.Model
	s.Domain, s.Deploy, s.Feedback, s.Cookies = sh.Domain, sh.Deploy, sh.Feedback, sh.Cookies
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

// readShared applies settings that came from another computer.
func readShared(s *config.Site) error {
	b, err := os.ReadFile(sharedPath(s))
	if err != nil {
		return err
	}
	sh, err := parseShared(b)
	if err != nil {
		return err
	}
	sh.apply(s)
	return s.Save()
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
		if !r.Private || !hasTopic(r.Topics, SiteTopic) {
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
	if err := readShared(site); err != nil {
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
