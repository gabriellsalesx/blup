package notesync

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"golang.org/x/text/unicode/norm"
)

// localFile is a file found under Dir.
type localFile struct {
	key   string // NFC, slash-separated, relative to Dir
	abs   string
	size  int64
	mtime int64 // Unix nanoseconds
	perm  fs.FileMode
	sha   string
	// cacheable: mtime is old enough to trust size+mtime next time.
	cacheable bool
}

// remoteFile is a blob of the remote tree.
type remoteFile struct {
	key  string // NFC
	path string // exact path in the repository
	sha  string
	mode string
}

// ignoredName reports whether a path segment is excluded from syncing:
// every dot file and dot folder (.obsidian, .trash, .git, .DS_Store, ...).
func ignoredName(name string) bool { return strings.HasPrefix(name, ".") }

func ignoredPath(p string) bool {
	for _, seg := range strings.Split(p, "/") {
		if ignoredName(seg) {
			return true
		}
	}
	return false
}

func nfc(p string) string { return norm.NFC.String(p) }

// hashFile returns the git blob sha of the file at abs.
func hashFile(abs string) (string, int64, error) {
	f, err := os.Open(abs)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", 0, err
	}
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", st.Size())
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	if n != st.Size() {
		return "", 0, fmt.Errorf("notesync: %s changed while reading", abs)
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// scan hashes every syncable file under s.Dir. Files that cannot be synced
// are added to skip (and to the report).
func (s *Syncer) scan(prev *state, rep *Report, skip map[string]bool, progress func(Progress)) (map[string]*localFile, error) {
	root := s.Dir
	stateAbs, _ := filepath.Abs(s.StateFile)
	trashAbs, _ := filepath.Abs(s.trashDir())
	start := time.Now() // real clock: compared with file mtimes
	out := map[string]*localFile{}
	if _, err := os.Stat(root); os.IsNotExist(err) && prev == nil {
		return out, nil // new computer: everything will be downloaded
	}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		abs, _ := filepath.Abs(p)
		if d.IsDir() {
			if ignoredName(d.Name()) || abs == trashAbs {
				return filepath.SkipDir
			}
			return nil
		}
		if ignoredName(d.Name()) || abs == stateAbs || !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		key := nfc(filepath.ToSlash(rel))
		if _, dup := out[key]; dup || skip[key] {
			rep.skip(key, SkipDuplicateName)
			skip[key] = true
			delete(out, key)
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > github.MaxFileSize {
			rep.skip(key, SkipTooLarge)
			skip[key] = true
			return nil
		}
		lf := &localFile{key: key, abs: p, size: info.Size(), mtime: info.ModTime().UnixNano(), perm: info.Mode().Perm()}
		lf.cacheable = info.ModTime().Before(start.Add(-2 * time.Second))
		if prev != nil {
			if c, ok := prev.Files[key]; ok && c.MTime != 0 && c.MTime == lf.mtime && c.Size == lf.size && lf.cacheable {
				lf.sha = c.SHA
			}
		}
		if lf.sha == "" {
			sha, n, err := hashFile(p)
			if err != nil {
				return err
			}
			if n != lf.size {
				lf.size, lf.cacheable = n, false
			}
			lf.sha = sha
		}
		out[key] = lf
		if progress != nil {
			progress(Progress{Phase: PhaseScan, Done: len(out), Path: key})
		}
		return nil
	})
	return out, err
}

// remoteFiles converts a remote tree into syncable files.
func remoteFiles(tree []github.TreeFile, rep *Report, skip map[string]bool) (map[string]*remoteFile, map[string]bool) {
	out := map[string]*remoteFile{}
	shas := map[string]bool{}
	for _, f := range tree {
		shas[f.SHA] = true
		if ignoredPath(f.Path) {
			continue
		}
		key := nfc(f.Path)
		switch {
		case f.Mode != "100644" && f.Mode != "100755":
			rep.skip(key, SkipUnsupported) // symlinks, submodules
			skip[key] = true
			continue
		case !filepath.IsLocal(filepath.FromSlash(key)) || strings.Contains(key, `\`):
			rep.skip(key, SkipInvalidName)
			skip[key] = true
			continue
		}
		if _, dup := out[key]; dup {
			rep.skip(key, SkipDuplicateName)
			skip[key] = true
			delete(out, key)
			continue
		}
		out[key] = &remoteFile{key: key, path: f.Path, sha: f.SHA, mode: f.Mode}
	}
	return out, shas
}
