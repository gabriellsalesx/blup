package app

import (
	"archive/zip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
)

// BackupKeep is how many daily copies of a site stay in the backup folder.
const BackupKeep = 7

// BackupSite saves a copy of the site's notes (attachments and Obsidian's
// settings too) as "<title> <date>.zip" in dir, once a day: when today's
// copy is there, it does nothing and returns its path. Copies beyond
// BackupKeep, the oldest, are deleted. The zip is written to a temporary
// name first, so a copy is never half written.
func BackupSite(site *config.Site, dir string) (string, error) {
	if dir == "" {
		return "", errors.New("backup: no folder")
	}
	root := site.ContentDir()
	if rel, err := filepath.Rel(root, dir); err == nil && !strings.HasPrefix(rel, "..") {
		return "", errors.New("backup: the backup folder can't be inside the notes")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := safeFileName(site.Title)
	if name == "" {
		name = "Blup"
	}
	dest := filepath.Join(dir, name+" "+time.Now().Format("2006-01-02")+".zip")
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}
	tmp, err := os.CreateTemp(dir, ".blup-backup-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	zw := zip.NewWriter(tmp)
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// Git's own folder is history of its own, not notes.
			if d.Name() == ".git" || d.Name() == ".trash" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name, h.Method = filepath.ToSlash(rel), zip.Deflate
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), dest); err != nil {
		return "", err
	}
	pruneBackups(dir, name)
	return dest, nil
}

// pruneBackups deletes the oldest daily copies of name beyond BackupKeep.
func pruneBackups(dir, name string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var copies []string
	for _, e := range entries {
		n := e.Name()
		if day, ok := strings.CutPrefix(n, name+" "); ok && strings.HasSuffix(n, ".zip") {
			if _, err := time.Parse("2006-01-02", strings.TrimSuffix(day, ".zip")); err == nil {
				copies = append(copies, n)
			}
		}
	}
	slices.Sort(copies) // by date: the names end in it
	for len(copies) > BackupKeep {
		_ = os.Remove(filepath.Join(dir, copies[0]))
		copies = copies[1:]
	}
}
