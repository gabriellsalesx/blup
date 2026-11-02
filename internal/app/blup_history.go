package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
)

// The history of a site's notes on Blup Cloud (plans with history):
// earlier versions of a note, notes deleted, and putting one back. A note
// put back is written to the folder like any edit, so the next sync saves
// it as a new version and nothing is lost.

// NoteHistory lists the contents a note had, newest first.
func NoteHistory(ctx context.Context, site *config.Site, path string) ([]blupcloud.NoteVersion, error) {
	r, err := blupRemote(ctx, site)
	if err != nil {
		return nil, err
	}
	h, err := r.FileHistory(ctx, path, 200)
	if err != nil {
		return nil, errors.New(describe(err))
	}
	return h, nil
}

// NoteVersionText is a note's text in version n ("" and false for a file
// that is not text).
func NoteVersionText(ctx context.Context, site *config.Site, path string, n int64) (string, bool, error) {
	r, err := blupRemote(ctx, site)
	if err != nil {
		return "", false, err
	}
	b, err := r.ReadAt(ctx, path, n)
	if err != nil {
		return "", false, errors.New(describe(err))
	}
	if !utf8.Valid(b) {
		return "", false, nil
	}
	return string(b), true, nil
}

// RestoreNoteVersion puts path back as it was in version n. What the
// folder has now is synced first, so it stays in the history.
func RestoreNoteVersion(ctx context.Context, site *config.Site, path string, n int64) error {
	r, err := blupRemote(ctx, site)
	if err != nil {
		return err
	}
	b, err := r.ReadAt(ctx, path, n)
	if err != nil {
		return errors.New(describe(err))
	}
	if _, err := syncSite(ctx, site, true); err != nil {
		return err
	}
	// Inside the notes, and not through a link (symlink) to elsewhere.
	if !filepath.IsLocal(filepath.FromSlash(path)) {
		return errors.New("app: invalid path")
	}
	dst, err := notesPath(site, path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

// DeletedNotes lists the notes of earlier versions that are no longer in
// the folder.
func DeletedNotes(ctx context.Context, site *config.Site) ([]blupcloud.DeletedFile, error) {
	r, err := blupRemote(ctx, site)
	if err != nil {
		return nil, err
	}
	all, err := r.Deleted(ctx)
	if err != nil {
		return nil, errors.New(describe(err))
	}
	out := all[:0]
	for _, d := range all {
		// Brought back already (not synced yet), or a dot file.
		if _, err := os.Lstat(filepath.Join(site.ContentDir(), filepath.FromSlash(d.Path))); err == nil {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}
