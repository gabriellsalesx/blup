package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
	"github.com/gabriellsalesx/blup/internal/update"
)

func init() {
	registerErrors(
		knownError{update.ErrChecksum, "update.err_checksum"},
		knownError{update.ErrNoAsset, "update.err_asset"},
		knownError{update.ErrNotWritable, "update.err_writable"},
		knownError{update.ErrNoBackup, "update.err_backup"},
	)
}

// offerUpdate runs when the menu opens: at most one network check a day,
// never more than two seconds, silent on any failure. When a newer version
// exists it asks; it never updates without asking.
func offerUpdate(ctx context.Context) (updated bool) {
	if Version == "dev" || os.Getenv("BLUP_NO_UPDATE_CHECK") != "" || !ui.Interactive() {
		return false
	}
	cache := update.LoadCache()
	if time.Since(cache.Checked) > 24*time.Hour {
		cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		rel, err := (&update.Client{}).Latest(cctx)
		cancel()
		if err != nil {
			return false
		}
		cache.Checked, cache.Latest, cache.Notes, cache.URL = time.Now(), rel.Version, rel.Notes, rel.URL
		_ = cache.Save()
	}
	if !update.Newer(cache.Latest, Version) || cache.Skipped == cache.Latest {
		return false
	}

	choice := "now"
	if err := ui.Choose(i18n.T("update.available", cache.Latest, Version), releaseNotes(cache.Notes), &choice,
		ui.Option(i18n.T("update.now"), "now"),
		ui.Option(i18n.T("update.later"), "later"),
		ui.Option(i18n.T("update.skip"), "skip"),
	); err != nil {
		return false
	}
	switch choice {
	case "skip":
		cache.Skipped = cache.Latest
		_ = cache.Save()
		return false
	case "later":
		return false
	}
	if err := runUpdate(ctx); err != nil {
		ui.Error(describe(err))
		return false
	}
	return true
}

// releaseNotes keeps the first lines of the release text for the prompt.
func releaseNotes(notes string) string {
	lines := strings.Split(strings.TrimSpace(notes), "\n")
	if len(lines) > 8 {
		lines = append(lines[:8], "…")
	}
	return strings.Join(lines, "\n")
}

// runUpdate downloads and installs the latest release.
func runUpdate(ctx context.Context) error {
	exe, err := executable()
	if err != nil {
		return err
	}
	c := &update.Client{}
	var rel update.Release
	if err := ui.Spin(ctx, i18n.T("update.checking"), func(ctx context.Context) error {
		rel, err = c.Latest(ctx)
		return err
	}); err != nil {
		return err
	}
	if !update.Newer(rel.Version, Version) {
		ui.OK(i18n.T("update.current", Version))
		return nil
	}
	if err := ui.Spin(ctx, i18n.T("update.downloading", rel.Version), func(ctx context.Context) error {
		return c.Apply(ctx, rel, exe)
	}); err != nil {
		return err
	}
	ui.OK(i18n.T("update.done", rel.Version))
	ui.Hint(i18n.T("update.restart"))
	return nil
}

// updateCommand is "blup update" (and "blup update --rollback").
func updateCommand(ctx context.Context, args []string) error {
	if Version == "dev" {
		return errors.New(i18n.T("update.dev"))
	}
	for _, a := range args {
		if a == "--rollback" || a == "--voltar" {
			exe, err := executable()
			if err != nil {
				return err
			}
			if err := update.Rollback(exe); err != nil {
				return err
			}
			ui.OK(i18n.T("update.rolled_back"))
			return nil
		}
	}
	return runUpdate(ctx)
}

func executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}
