package app

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
)

// Snapshots of the notes, taken by Blup Studio before an AI agent starts in
// its terminal, so what the agent changed can be seen and undone. They live
// in the site's .blup/snapshots (outside the notes, never published); the
// last few are kept.

const (
	keepSnapshots   = 5
	snapshotMaxFile = 20 << 20  // larger files (videos) are not copied
	snapshotMaxAll  = 300 << 20 // nor anything past this, in all
)

func snapshotsDir(site *config.Site) string { return filepath.Join(site.Dir(), ".blup", "snapshots") }

// notesFiles walks the notes, skipping hidden folders and files.
func notesFiles(root string, fn func(rel string, info fs.FileInfo) error) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != root && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		return fn(filepath.ToSlash(rel), info)
	})
}

// TakeSnapshot copies the notes and returns the snapshot's folder.
func TakeSnapshot(site *config.Site) (string, error) {
	dir := filepath.Join(snapshotsDir(site), time.Now().Format("2006-01-02-150405.000"))
	root := site.ContentDir()
	var total int64
	err := notesFiles(root, func(rel string, info fs.FileInfo) error {
		if info.Size() > snapshotMaxFile || total+info.Size() > snapshotMaxAll {
			return nil
		}
		total += info.Size()
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	})
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	pruneSnapshots(site)
	return dir, nil
}

func pruneSnapshots(site *config.Site) {
	list, _ := os.ReadDir(snapshotsDir(site))
	for i := 0; i < len(list)-keepSnapshots; i++ {
		os.RemoveAll(filepath.Join(snapshotsDir(site), list[i].Name()))
	}
}

// Change is a note or file that differs from a snapshot.
type Change struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // "added", "changed" or "removed"
}

func inSnapshots(site *config.Site, dir string) error {
	rel, err := filepath.Rel(snapshotsDir(site), dir)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || strings.ContainsRune(rel, filepath.Separator) {
		return errors.New("not a Blup snapshot")
	}
	if _, err := os.Stat(dir); err != nil {
		return errors.New("the snapshot is gone")
	}
	return nil
}

// SnapshotChanges lists what changed in the notes since the snapshot.
// Files too big to be in it are left out.
func SnapshotChanges(site *config.Site, dir string) ([]Change, error) {
	if err := inSnapshots(site, dir); err != nil {
		return nil, err
	}
	root := site.ContentDir()
	now := map[string]bool{}
	out := []Change{}
	err := notesFiles(root, func(rel string, info fs.FileInfo) error {
		now[rel] = true
		old, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if errors.Is(err, fs.ErrNotExist) {
			if info.Size() <= snapshotMaxFile {
				out = append(out, Change{rel, "added"})
			}
			return nil
		}
		if err != nil {
			return err
		}
		cur, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		if !bytes.Equal(old, cur) {
			out = append(out, Change{rel, "changed"})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = notesFiles(dir, func(rel string, _ fs.FileInfo) error {
		if !now[rel] {
			out = append(out, Change{rel, "removed"})
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, err
}

// UndoSnapshot puts the notes back as they were in the snapshot: changed
// and removed files come back, and new ones go to the Blup trash.
func UndoSnapshot(site *config.Site, dir string) error {
	changes, err := SnapshotChanges(site, dir)
	if err != nil {
		return err
	}
	root := site.ContentDir()
	for _, c := range changes {
		if c.Kind == "added" {
			if _, err := toTrash(site, c.Path); err != nil {
				return err
			}
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(c.Path)))
		if err != nil {
			return err
		}
		dst := filepath.Join(root, filepath.FromSlash(c.Path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}
