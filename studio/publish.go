package main

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/cfworkers"
	"github.com/gabriellsalesx/blup/internal/ui"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Publishing and syncing from the window: accounts, destination, and the
// progress of each run, sent as "progress" events.

// Progress is a line of what a publish or sync is doing.
type Progress struct {
	Kind string `json:"kind"` // step, spin, ok, warn, error, hint
	Text string `json:"text"`
}

// Republish is a publish the Studio does alone, heard by the window as
// "republish": the site's own domain on Blup Cloud turned active or
// stopped being, so its links (canonical, feed, sitemap) move to the
// right address. State: "start", "done" (URL is where it is) or "error".
type Republish struct {
	Dir   string `json:"dir"`
	State string `json:"state"`
	URL   string `json:"url"`
	Error string `json:"error"`
}

func init() {
	application.RegisterEvent[Progress]("progress")
	application.RegisterEvent[Republish]("republish")
}

var busy sync.Mutex

// errBusy: another publish or sync is running.
var errBusy = errors.New("busy")

// running sends the progress lines of fn to the window. One run at a time.
func running(fn func() error) error {
	if !busy.TryLock() {
		return errBusy
	}
	defer busy.Unlock()
	ui.Report = func(kind, text string) { application.Get().Event.Emit("progress", Progress{kind, text}) }
	defer func() { ui.Report = nil }()
	return fn()
}

// Connections tells which accounts are connected.
func (s *Studio) Connections(ctx context.Context) (app.Connections, error) {
	return app.LoadConnections(ctx)
}

// OpenKeyPage opens, in the browser, the form for a new key of "github" or
// "cloudflare", with the right permissions ticked.
func (s *Studio) OpenKeyPage(account string) {
	link := app.GitHubKeyLink()
	if account == "cloudflare" {
		link = app.CloudflareKeyLink()
	}
	s.OpenURL(link)
}

// OpenURL opens an address in the browser.
//
// Only web, e-mail and phone addresses: a link written in a note or a
// board (file:///…, or a scheme that starts a program) never reaches the
// system.
func (s *Studio) OpenURL(link string) {
	if u, err := url.Parse(strings.TrimSpace(link)); err == nil && slices.Contains([]string{"http", "https", "mailto", "tel"}, strings.ToLower(u.Scheme)) {
		_ = application.Get().Browser.OpenURL(u.String())
	}
}

// ConnectGitHub checks and saves a GitHub key; it returns the login.
func (s *Studio) ConnectGitHub(ctx context.Context, token string) (string, error) {
	return app.ConnectGitHub(ctx, token)
}

// ConnectCloudflare checks and saves a Cloudflare key; it returns the
// accounts it can publish to.
func (s *Studio) ConnectCloudflare(ctx context.Context, token string) ([]cfworkers.Account, error) {
	return app.ConnectCloudflare(ctx, token)
}

// Disconnect forgets the key of "github" or "cloudflare".
func (s *Studio) Disconnect(account string) error { return app.Disconnect(account) }

// CloudflareAccounts lists the accounts of the saved Cloudflare key.
func (s *Studio) CloudflareAccounts(ctx context.Context) ([]cfworkers.Account, error) {
	return app.CloudflareAccounts(ctx)
}

// CloudflareSubdomain is the workers.dev subdomain of a Cloudflare account
// ("" when the account has none yet: the Publish screen asks for one).
func (s *Studio) CloudflareSubdomain(ctx context.Context, account string) (string, error) {
	return app.CloudflareSubdomain(ctx, account)
}

// SetCloudflareSubdomain gives a Cloudflare account its workers.dev
// subdomain, once (it is the same for every site of the account).
func (s *Studio) SetCloudflareSubdomain(ctx context.Context, account, name string) (string, error) {
	return app.SetCloudflareSubdomain(ctx, account, name)
}

// CheckSiteSlug says whether an address on Blup Cloud is free (while it is
// typed on the Publish screen).
func (s *Studio) CheckSiteSlug(ctx context.Context, slug string) (blupcloud.SlugCheck, error) {
	site, err := s.current()
	if err != nil {
		return blupcloud.SlugCheck{}, err
	}
	return app.CheckBlupSlug(ctx, site, slug)
}

// Target is where the open site is published.
func (s *Studio) Target() (app.Target, error) {
	site, err := s.current()
	if err != nil {
		return app.Target{}, err
	}
	return app.LoadTarget(site), nil
}

// SetTarget saves where the open site is published.
func (s *Studio) SetTarget(t app.Target) (app.Target, error) {
	site, err := s.current()
	if err != nil {
		return app.Target{}, err
	}
	if err := app.SetTarget(site, t); err != nil {
		return app.Target{}, err
	}
	return app.LoadTarget(site), nil
}

// Publish builds the open site and puts it online; it returns the address.
func (s *Studio) Publish() (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	var live string
	err = running(func() (err error) {
		live, err = app.PublishNow(context.Background(), site)
		return err
	})
	return live, err
}

// SyncStatus tells what syncing the notes would send and receive.
func (s *Studio) SyncStatus(ctx context.Context) (app.SyncInfo, error) {
	site, err := s.current()
	if err != nil {
		return app.SyncInfo{}, err
	}
	return app.SyncStatus(ctx, site)
}

// Sync sends and receives the notes (private GitHub repository).
func (s *Studio) Sync() (app.SyncInfo, error) {
	site, err := s.current()
	if err != nil {
		return app.SyncInfo{}, err
	}
	var info app.SyncInfo
	err = running(func() (err error) {
		info, err = app.SyncNow(context.Background(), site)
		return err
	})
	// The first sync through Blup Cloud creates the vault: live sync can
	// start now (on live plans).
	s.mu.Lock()
	start := err == nil && (s.live == nil || s.live.Gone()) && site.Sync.Remote == app.RemoteBlup
	s.mu.Unlock()
	if start {
		s.startLive(site)
	}
	return info, err
}

// ResetSync is "Compare from scratch", offered only after a sync stopped
// for safety (app.ErrSyncStuck) and confirmed by the person: this computer
// forgets the last sync of the open site's notes and compares file by file
// with Blup Cloud, as a new computer; what differs becomes a conflict copy
// and no note is deleted.
func (s *Studio) ResetSync() (app.SyncInfo, error) {
	site, err := s.current()
	if err != nil {
		return app.SyncInfo{}, err
	}
	var info app.SyncInfo
	err = running(func() (err error) {
		info, err = app.ResetSync(context.Background(), site)
		return err
	})
	s.mu.Lock()
	start := err == nil && (s.live == nil || s.live.Gone()) && site.Sync.Remote == app.RemoteBlup
	s.mu.Unlock()
	if start {
		s.startLive(site)
	}
	return info, err
}

// OpenNewGitHubRepo opens GitHub's "new repository" form filled in for the
// site ("site", named name) or its notes ("notes"): Blup's key is limited
// to chosen repositories and cannot create them.
func (s *Studio) OpenNewGitHubRepo(kind, name string) error {
	site, err := s.current()
	if err != nil {
		return err
	}
	link := app.GitHubNewRepoLink(site, kind, name)
	if link == "" {
		return errors.New("invalid repository name")
	}
	s.OpenURL(link)
	return nil
}

// BlupDomain is the own domain of the open site on Blup Cloud (the cloud
// checks the DNS again) and whether the plan includes it.
func (s *Studio) BlupDomain(ctx context.Context) (app.BlupDomain, error) {
	site, err := s.current()
	if err != nil {
		return app.BlupDomain{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	d, err := app.LoadBlupDomain(ctx, site)
	s.republishIfStale(site, d)
	return d, err
}

// SetBlupDomain links a domain to the open site on Blup Cloud, or checks
// the one linked again.
func (s *Studio) SetBlupDomain(hostname string) (app.BlupDomain, error) {
	site, err := s.current()
	if err != nil {
		return app.BlupDomain{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	d, err := app.SetBlupDomain(ctx, site, hostname)
	s.republishIfStale(site, d)
	return d, err
}

// RemoveBlupDomain unlinks the open site's own domain on Blup Cloud.
func (s *Studio) RemoveBlupDomain() (app.BlupDomain, error) {
	site, err := s.current()
	if err != nil {
		return app.BlupDomain{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	d, err := app.RemoveBlupDomain(ctx, site)
	s.republishIfStale(site, d)
	return d, err
}

// republishIfStale publishes the site again in the background when its
// links point to an address it no longer has (see app.BlupDomain).
func (s *Studio) republishIfStale(site *config.Site, d app.BlupDomain) {
	if d.Republish {
		go s.republish(site)
	}
}

// republishWait keeps a failed (or just made) publish from running again
// on every check of the domain.
const republishWait = 10 * time.Minute

// republish publishes site again, quietly: one run at a time (a publish
// already going wins, and a later check tries again) and no more than
// once in republishWait for the same site.
func (s *Studio) republish(site *config.Site) {
	dir := site.Dir()
	s.mu.Lock()
	if s.republished == nil {
		s.republished = map[string]time.Time{}
	}
	if time.Since(s.republished[dir]) < republishWait {
		s.mu.Unlock()
		return
	}
	s.republished[dir] = time.Now()
	s.mu.Unlock()
	emit := func(r Republish) {
		r.Dir = dir
		if a := application.Get(); a != nil {
			a.Event.Emit("republish", r)
		}
	}
	var live string
	err := running(func() (err error) {
		emit(Republish{State: "start"})
		live, err = app.PublishNow(context.Background(), site)
		return err
	})
	switch {
	case errors.Is(err, errBusy):
		s.mu.Lock()
		delete(s.republished, dir) // nothing ran: the next check may
		s.mu.Unlock()
	case err != nil:
		emit(Republish{State: "error", Error: err.Error()})
	default:
		emit(Republish{State: "done", URL: live})
	}
}

// checkBase looks, when a site opens, whether its links follow its own
// domain on Blup Cloud, and publishes again if they don't. Silent when
// signed out or offline.
func (s *Studio) checkBase(site *config.Site) {
	if site.Deploy.Provider != config.ProviderBlup || site.Deploy.Blup.ID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if d, err := app.LoadBlupDomain(ctx, site); err == nil {
		s.republishIfStale(site, d)
	}
}
