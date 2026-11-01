package main

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
)

// The file explorer's right-click menu.

// RenameItem renames a note or folder (links to it follow); it returns the
// new path.
func (s *Studio) RenameItem(rel, name string) (string, error) {
	var out string
	err := s.with(func(site *config.Site) (err error) { out, err = app.RenameItem(site, rel, name); return })
	return out, err
}

// MoveItem moves a note or folder into folder (links follow).
func (s *Studio) MoveItem(rel, folder string) (string, error) {
	var out string
	err := s.with(func(site *config.Site) (err error) { out, err = app.MoveItem(site, rel, folder); return })
	return out, err
}

// CopyNote makes a copy of a note next to it.
func (s *Studio) CopyNote(rel string) (string, error) {
	var out string
	err := s.with(func(site *config.Site) (err error) { out, err = app.CopyNote(site, rel); return })
	return out, err
}

// DeleteItem sends a note or folder to the Blup trash.
// It returns where it went, to undo (RestoreSection).
func (s *Studio) DeleteItem(rel string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	return app.DeleteItem(site, rel)
}

// NoteLink is a Markdown link to the note at rel, written from the note at
// from ("" for the top of the notes): [Name](../Folder/Name.md).
func (s *Studio) NoteLink(rel, from string) string {
	base := path.Dir(from)
	if from == "" {
		base = "."
	}
	r, err := filepath.Rel(filepath.FromSlash(base), filepath.FromSlash(rel))
	if err != nil {
		r = rel
	}
	return "[" + strings.TrimSuffix(path.Base(rel), ".md") + "](" + strings.ReplaceAll(filepath.ToSlash(r), " ", "%20") + ")"
}

// ShowInFolder opens the system's file manager where the note is.
func (s *Studio) ShowInFolder(rel string) error {
	p, err := s.notePath(rel)
	if err != nil {
		return err
	}
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		p = filepath.Dir(p)
	}
	return openExternal(p)
}

// openExternal opens an address or a folder with the system.
func openExternal(target string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", target)
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		c = exec.Command("xdg-open", target)
	}
	return c.Start()
}
