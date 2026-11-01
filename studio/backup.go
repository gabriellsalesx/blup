package main

import (
	"os"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The daily copy of the notes (Settings › General): made a little after a
// site opens, in the background, once a day.

// backupLater copies site's notes a minute after it opens, when backups
// are on, so opening stays quick.
func (s *Studio) backupLater(site *config.Site) {
	dir := s.Prefs().BackupDir
	if dir == "" {
		return
	}
	time.AfterFunc(time.Minute, func() {
		if cur, err := s.current(); err == nil && cur == site {
			_, _ = app.BackupSite(site, dir)
		}
	})
}

// BackupNow copies the open site's notes now (once a day: today's copy is
// kept) and returns the zip's path.
func (s *Studio) BackupNow() (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	return app.BackupSite(site, s.Prefs().BackupDir)
}

// PickBackupDir asks for the backup folder ("" when the person cancels);
// the window saves it in the preferences.
func (s *Studio) PickBackupDir() (string, error) {
	home, _ := os.UserHomeDir()
	return application.Get().Dialog.OpenFile().SetTitle("Blup Studio").CanChooseDirectories(true).
		CanChooseFiles(false).CanCreateDirectories(true).SetDirectory(home).PromptForSingleSelection()
}
