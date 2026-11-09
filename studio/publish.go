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
	"github.com/gabriellsalesx/blup/internal/notesync"
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
	return runningWith(fn, nil)
}

// runningWith is running, with each line also given to line.
func runningWith(fn func() error, line func(Progress)) error {
	if !busy.TryLock() {
		return errBusy
	}
	defer busy.Unlock()
	return reporting(fn, line)
}

// queued is runningWith for the background jobs (publish, sync): it waits
// for the one running to end instead of failing, so a sync asked for
// during a publish runs right after it.
func queued(fn func() error, line func(Progress)) error {
	busy.Lock()
	defer busy.Unlock()
	return reporting(fn, line)
}

func reporting(fn func() error, line func(Progress)) error {
	ui.Report = func(kind, text string) {
		p := Progress{kind, text}
		if line != nil {
			line(p)
		}
		application.Get().Event.Emit("progress", p)
	}
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

// PublishJob is the publish running in the background: one at a time,
// going on while the person works on other screens (the side strip shows
// it), with its lines kept for a Publish screen opened meanwhile.
type PublishJob struct {
	Running bool       `json:"running"`
	Lines   []Progress `json:"lines"`
	// Sending the files to Blup Cloud: done of total.
	Done  int `json:"done"`
	Total int `json:"total"`
}

// PublishCount is how many files went up of how many, heard by the window
// as "publishCount" while a site goes to Blup Cloud.
type PublishCount struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

// PublishDone is how a background publish ended, heard by the window as
// "publishDone": where the site is, or the error.
type PublishDone struct {
	URL   string `json:"url"`
	Error string `json:"error"`
}

var publishJob struct {
	sync.Mutex
	st PublishJob
}

func init() {
	application.RegisterEvent[Progress]("publishLine")
	application.RegisterEvent[PublishDone]("publishDone")
	application.RegisterEvent[PublishCount]("publishCount")
}

// PublishState is the background publish now, with its lines so far.
func (s *Studio) PublishState() PublishJob {
	publishJob.Lock()
	defer publishJob.Unlock()
	st := publishJob.st
	st.Lines = slices.Clone(st.Lines)
	return st
}

// StartPublish builds and publishes the open site in the background: each
// line goes out as "publishLine" and the end as "publishDone". Starting
// while one runs only keeps it going.
func (s *Studio) StartPublish() error {
	site, err := s.current()
	if err != nil {
		return err
	}
	publishJob.Lock()
	if publishJob.st.Running {
		publishJob.Unlock()
		return nil
	}
	publishJob.st = PublishJob{Running: true}
	publishJob.Unlock()
	ctx := blupcloud.WithUploadProgress(context.Background(), func(done, total int) {
		publishJob.Lock()
		publishJob.st.Done, publishJob.st.Total = done, total
		publishJob.Unlock()
		application.Get().Event.Emit("publishCount", PublishCount{done, total})
	})
	go func() {
		var live string
		err := queued(func() (err error) {
			live, err = app.PublishNow(ctx, site)
			return err
		}, func(p Progress) {
			publishJob.Lock()
			publishJob.st.Lines = append(publishJob.st.Lines, p)
			publishJob.Unlock()
			application.Get().Event.Emit("publishLine", p)
		})
		end := PublishDone{URL: live}
		if err != nil {
			end.Error = err.Error()
		}
		publishJob.Lock()
		publishJob.st.Running = false
		publishJob.Unlock()
		application.Get().Event.Emit("publishDone", end)
	}()
	return nil
}

// SyncStatus tells what syncing the notes would send and receive.
func (s *Studio) SyncStatus(ctx context.Context) (app.SyncInfo, error) {
	site, err := s.current()
	if err != nil {
		return app.SyncInfo{}, err
	}
	return app.SyncStatus(ctx, site)
}

// SyncJob is the sync of the notes running in the background: one at a
// time, going on while the person works on other screens (the window
// shows it in the side strip). Phase ("scan", "compare", "upload",
// "download", "commit", "retry") and, sending or receiving, done of total.
type SyncJob struct {
	Running bool   `json:"running"`
	Phase   string `json:"phase"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Fresh   bool   `json:"fresh"`
}

// SyncDone is how a background sync ended, heard by the window as
// "syncDone". Stuck: it stopped for safety (app.ErrSyncStuck), and the
// window offers "Compare from scratch".
type SyncDone struct {
	Info  app.SyncInfo `json:"info"`
	Error string       `json:"error"`
	Stuck bool         `json:"stuck"`
	Fresh bool         `json:"fresh"`
}

var syncJob struct {
	sync.Mutex
	st SyncJob
}

func init() {
	application.RegisterEvent[SyncJob]("syncProgress")
	application.RegisterEvent[SyncDone]("syncDone")
}

// SyncState is the background sync now (Running false when there is
// none): a window that opens the screen again picks it up.
func (s *Studio) SyncState() SyncJob {
	syncJob.Lock()
	defer syncJob.Unlock()
	return syncJob.st
}

// StartSync sends and receives the open site's notes in the background,
// telling the window how far it went ("syncProgress") and how it ended
// ("syncDone"). fresh is "Compare from scratch", offered only after a sync
// stopped for safety (app.ErrSyncStuck) and confirmed by the person: this
// computer forgets the last sync and compares file by file with Blup
// Cloud, as a new computer; what differs becomes a conflict copy and no
// note is deleted. Starting while one runs only keeps it going.
func (s *Studio) StartSync(fresh bool) error {
	site, err := s.current()
	if err != nil {
		return err
	}
	syncJob.Lock()
	if syncJob.st.Running {
		syncJob.Unlock()
		return nil
	}
	syncJob.st = SyncJob{Running: true, Fresh: fresh}
	syncJob.Unlock()
	ctx := app.WithSyncProgress(context.Background(), func(p notesync.Progress) {
		st := SyncJob{Running: true, Phase: string(p.Phase), Done: p.Done, Total: p.Total, Fresh: fresh}
		syncJob.Lock()
		syncJob.st = st
		syncJob.Unlock()
		application.Get().Event.Emit("syncProgress", st)
	})
	go func() {
		var info app.SyncInfo
		err := queued(func() (err error) {
			if fresh {
				info, err = app.ResetSync(ctx, site)
			} else {
				info, err = app.SyncNow(ctx, site)
			}
			return err
		}, nil)
		// The first sync through Blup Cloud creates the vault: live sync
		// can start now (on live plans).
		s.mu.Lock()
		start := err == nil && (s.live == nil || s.live.Gone()) && site.Sync.Remote == app.RemoteBlup
		s.mu.Unlock()
		if start {
			s.startLive(site)
		}
		end := SyncDone{Info: info, Fresh: fresh}
		if err != nil {
			end.Error = err.Error()
			end.Stuck = errors.Is(err, app.ErrSyncStuck)
		}
		syncJob.Lock()
		syncJob.st = SyncJob{}
		syncJob.Unlock()
		application.Get().Event.Emit("syncDone", end)
	}()
	return nil
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
