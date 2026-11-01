package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/deploy/cfpages"
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

func init() { application.RegisterEvent[Progress]("progress") }

var busy sync.Mutex

// running sends the progress lines of fn to the window. One run at a time.
func running(fn func() error) error {
	if !busy.TryLock() {
		return errors.New("busy")
	}
	defer busy.Unlock()
	ui.Report = func(kind, text string) { application.Get().Event.Emit("progress", Progress{kind, text}) }
	defer func() { ui.Report = nil }()
	return fn()
}

// Connections tells which accounts are connected.
func (s *Studio) Connections() (app.Connections, error) {
	return app.LoadConnections(context.Background())
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
func (s *Studio) OpenURL(link string) { _ = application.Get().Browser.OpenURL(link) }

// ConnectGitHub checks and saves a GitHub key; it returns the login.
func (s *Studio) ConnectGitHub(token string) (string, error) {
	return app.ConnectGitHub(context.Background(), token)
}

// ConnectCloudflare checks and saves a Cloudflare key; it returns the
// accounts it can publish to.
func (s *Studio) ConnectCloudflare(token string) ([]cfpages.Account, error) {
	return app.ConnectCloudflare(context.Background(), token)
}

// Disconnect forgets the key of "github" or "cloudflare".
func (s *Studio) Disconnect(account string) error { return app.Disconnect(account) }

// CloudflareAccounts lists the accounts of the saved Cloudflare key.
func (s *Studio) CloudflareAccounts() ([]cfpages.Account, error) {
	return app.CloudflareAccounts(context.Background())
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
func (s *Studio) SyncStatus() (app.SyncInfo, error) {
	site, err := s.current()
	if err != nil {
		return app.SyncInfo{}, err
	}
	return app.SyncStatus(context.Background(), site)
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
	start := err == nil && s.live == nil && site.Sync.Remote == app.RemoteBlup
	s.mu.Unlock()
	if start {
		s.startLive(site)
	}
	return info, err
}

// CurrentDomain is the site's custom domain ("" for the free address).
func (s *Studio) CurrentDomain() string {
	site, err := s.current()
	if err != nil {
		return ""
	}
	return site.Domain
}

// SetDomain saves and connects a custom domain.
func (s *Studio) SetDomain(value string) (app.DomainStatus, error) {
	site, err := s.current()
	if err != nil {
		return app.DomainStatus{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return app.SetDomain(ctx, site, value)
}

// CheckDomain connects the domain again and says where it stands.
func (s *Studio) CheckDomain() (app.DomainStatus, error) {
	site, err := s.current()
	if err != nil {
		return app.DomainStatus{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return app.CheckDomain(ctx, site)
}

// RemoveDomain goes back to the free address.
func (s *Studio) RemoveDomain() error {
	site, err := s.current()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return app.RemoveDomain(ctx, site)
}
