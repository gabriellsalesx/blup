package main

import (
	"context"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// CloudSites lists the Blup sites found on the user's GitHub (their notes
// synced to a private repository).
func (s *Studio) CloudSites() ([]app.CloudSite, error) {
	return app.FindCloudSites(context.Background())
}

// BringSite makes a site found on GitHub on this computer and opens it.
func (s *Studio) BringSite(owner, repo string) (SiteInfo, error) {
	site, err := app.BringSite(context.Background(), owner, repo)
	if err != nil {
		return SiteInfo{}, err
	}
	return info(site), nil
}

// GeneralPassword tells whether the password of protected posts for every
// site is set.
func (s *Studio) GeneralPassword() bool { return app.HasGeneralPassword() }

// SetGeneralPassword sets the password of protected posts for every site
// without its own ("" removes it).
func (s *Studio) SetGeneralPassword(password string) error { return app.SetGeneralPassword(password) }

// BlupAccount is the Blup Cloud account signed in here (SignedIn false
// when none).
func (s *Studio) BlupAccount() (app.BlupAccount, error) {
	return app.LoadBlupAccount(context.Background())
}

// BlupLoginCode is what the person checks in the browser.
type BlupLoginCode struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

// BlupLogin starts signing in: it opens Blup Cloud in the browser, where
// the person checks the code and approves. Then call BlupLoginWait.
func (s *Studio) BlupLogin() (BlupLoginCode, error) {
	l, err := app.StartBlupLogin(context.Background())
	if err != nil {
		return BlupLoginCode{}, err
	}
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.login, s.cancel = l, nil
	s.mu.Unlock()
	s.OpenURL(l.URL)
	return BlupLoginCode{Code: l.Code, URL: l.URL}, nil
}

// BlupLoginWait waits for the approval in the browser (BlupLoginCancel stops
// it) and returns the account.
func (s *Studio) BlupLoginWait() (app.BlupAccount, error) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	l := s.login
	s.cancel = cancel
	s.mu.Unlock()
	defer cancel()
	if l == nil {
		return app.BlupAccount{}, context.Canceled
	}
	a, err := app.FinishBlupLogin(ctx, l)
	if err == nil {
		s.restartLive()
	}
	return a, err
}

// restartLive starts the live sync again for the open site (after signing
// in or opening the notes).
func (s *Studio) restartLive() {
	if site, err := s.current(); err == nil {
		s.startLive(site)
	}
}

// BlupLoginCancel gives up waiting for the browser.
func (s *Studio) BlupLoginCancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.login, s.cancel = nil, nil
}

// BlupSignOut leaves Blup Cloud on this computer.
func (s *Studio) BlupSignOut() error {
	s.startLive(nil)
	return app.BlupSignOut(context.Background())
}

// OpenBlupCloud opens the account panel in the browser.
func (s *Studio) OpenBlupCloud(page string) { s.OpenURL(blupcloud.BaseURL() + page) }

// CreateNotesPassword turns sync on and returns the recovery key (shown once).
func (s *Studio) CreateNotesPassword(password string) (string, error) {
	r, err := app.CreateNotesPassword(context.Background(), password)
	if err == nil {
		s.restartLive()
	}
	return r, err
}

// UnlockNotes opens the notes here with the notes password or the recovery key.
func (s *Studio) UnlockNotes(secret string) error {
	if err := app.UnlockNotes(context.Background(), secret); err != nil {
		return err
	}
	s.restartLive()
	return nil
}

// SyncRemote is where the open site's notes are synced: "blup" or "github".
func (s *Studio) SyncRemote() string {
	site, err := s.current()
	if err != nil || site.Sync.Remote == "" {
		return "github"
	}
	return site.Sync.Remote
}

// SetSyncRemote picks where the open site's notes are synced.
func (s *Studio) SetSyncRemote(remote string) error {
	site, err := s.current()
	if err != nil {
		return err
	}
	if err := app.SetSyncRemote(site, remote); err != nil {
		return err
	}
	s.startLive(site)
	return nil
}

// startLive starts the live sync of site (nil stops it); the window hears
// of it through "cloudSync".
func (s *Studio) startLive(site *config.Site) {
	s.mu.Lock()
	s.live.Stop()
	s.live = nil
	if site != nil {
		s.live = app.StartLive(site, func(st app.LiveState) { application.Get().Event.Emit("cloudSync", st) })
	}
	l := s.live
	s.mu.Unlock()
	watchMu.Lock()
	onNotesChanged = l.Changed
	watchMu.Unlock()
}

// CloudHistory tells whether the open site keeps old versions on Blup
// Cloud (synced there, on a plan with history, notes opened here).
func (s *Studio) CloudHistory() bool {
	site, err := s.current()
	if err != nil || site.Sync.Remote != app.RemoteBlup || site.Sync.Vault == "" {
		return false
	}
	a, err := app.LoadBlupAccount(context.Background())
	return err == nil && a.Notes == "ready" && a.HistoryDays > 0
}

// NoteHistory lists the earlier contents of a note, newest first.
func (s *Studio) NoteHistory(path string) ([]blupcloud.NoteVersion, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.NoteHistory(context.Background(), site, path)
}

// VersionText is a note's text in an earlier version (IsText false for
// images and other files).
type VersionText struct {
	Text   string `json:"text"`
	IsText bool   `json:"isText"`
}

// NoteVersionText reads a note as it was in version n.
func (s *Studio) NoteVersionText(path string, n int64) (VersionText, error) {
	site, err := s.current()
	if err != nil {
		return VersionText{}, err
	}
	text, ok, err := app.NoteVersionText(context.Background(), site, path, n)
	return VersionText{Text: text, IsText: ok}, err
}

// RestoreNoteVersion puts a note back as it was in version n (the current
// one stays in the history).
func (s *Studio) RestoreNoteVersion(path string, n int64) error {
	site, err := s.current()
	if err != nil {
		return err
	}
	return app.RestoreNoteVersion(context.Background(), site, path, n)
}

// DeletedNotes lists notes deleted that Blup Cloud still keeps.
func (s *Studio) DeletedNotes() ([]blupcloud.DeletedFile, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.DeletedNotes(context.Background(), site)
}

// BlupSites lists the sites kept on Blup Cloud (to bring them here).
func (s *Studio) BlupSites() ([]app.BlupSite, error) {
	return app.BlupSites(context.Background())
}

// BringBlupSite makes a site kept on Blup Cloud on this computer and opens it.
func (s *Studio) BringBlupSite(vault string) (SiteInfo, error) {
	site, err := app.BringBlupSite(context.Background(), vault)
	if err != nil {
		return SiteInfo{}, err
	}
	return info(site), nil
}
