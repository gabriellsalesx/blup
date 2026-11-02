package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
)

// Notes and folders from Blup Studio's file explorer: rename, move, copy
// and delete (to the Blup trash, which remembers where each thing was).
// Renaming and moving fix the links that point to what changed, and the
// links inside what moved, like Obsidian does. Paths are relative to the
// notes folder, with "/".

func notesPath(site *config.Site, rel string) (string, error) {
	root := filepath.Clean(site.ContentDir())
	p := filepath.Join(root, filepath.FromSlash(rel))
	if rel == "" || p == root || !strings.HasPrefix(p, root+string(filepath.Separator)) || !Within(root, p) {
		return "", errors.New("outside the notes folder")
	}
	return p, nil
}

// Within tells whether p, followed through its links (symlinks), stays in
// root: a link in the notes to a file elsewhere (~/.ssh) is not read or
// written through. A path not made yet counts by its folder.
func Within(root, p string) bool {
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		// A link to nothing (yet) would write wherever it points.
		if _, lerr := os.Lstat(p); lerr == nil || !errors.Is(err, os.ErrNotExist) {
			return false
		}
		dir, rest := p, ""
		for {
			rest = filepath.Join(filepath.Base(dir), rest)
			dir = filepath.Dir(dir)
			r, err := filepath.EvalSymlinks(dir)
			if err == nil {
				real = filepath.Join(r, rest)
				break
			}
			if !errors.Is(err, os.ErrNotExist) || filepath.Dir(dir) == dir {
				return false
			}
		}
	}
	return real == realRoot || strings.HasPrefix(real, realRoot+string(filepath.Separator))
}

// writeNew creates the file p with data, failing when anything is there
// already, a link included: a link to nothing (planted in a synced or
// downloaded folder) would otherwise make the file wherever it points.
func writeNew(p string, data []byte) error {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(p)
		return err
	}
	return f.Close()
}

// isLink tells whether a walked entry is a symbolic link: the walks that
// rewrite notes leave links alone, so a note that is a link to a file
// elsewhere (~/.bashrc) is never written through.
func isLink(d fs.DirEntry) bool { return d.Type()&fs.ModeSymlink != 0 }

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|#^[]`, r) {
			return -1
		}
		return r
	}, name))
	if name == "" || strings.HasPrefix(name, ".") {
		return "", errors.New(i18nInvalidName())
	}
	return name, nil
}

func i18nInvalidName() string { return "invalid name" }

// fileExt is the extension a file keeps when renamed or copied: a note's
// .md, a vault's canvases and drawings, or any other.
func fileExt(name string) string {
	n := strings.ToLower(name)
	for _, ext := range []string{".draw", ".canvas", ".table", ".md"} {
		if strings.HasSuffix(n, ext) {
			return name[len(name)-len(ext):]
		}
	}
	return path.Ext(name)
}

// RenameItem renames a note (name without its extension) or folder,
// keeping it where it is. It returns the new path.
func RenameItem(site *config.Site, rel, name string) (string, error) {
	ext := fileExt(rel)
	if ext != "" && strings.HasSuffix(strings.ToLower(name), strings.ToLower(ext)) {
		name = name[:len(name)-len(ext)]
	}
	name, err := cleanName(strings.TrimSuffix(name, ".md"))
	if err != nil {
		return "", err
	}
	src, err := notesPath(site, rel)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		name += ext
	}
	newRel := path.Join(path.Dir(rel), name)
	if path.Dir(rel) == "." {
		newRel = name
	}
	return newRel, moveItem(site, rel, newRel, st.IsDir())
}

// MoveItem moves a note or folder into folder ("" for the top). It returns
// the new path.
func MoveItem(site *config.Site, rel, folder string) (string, error) {
	src, err := notesPath(site, rel)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	newRel := path.Join(folder, path.Base(rel))
	if folder == "" {
		newRel = path.Base(rel)
	}
	if st.IsDir() && (folder == rel || strings.HasPrefix(folder+"/", rel+"/")) {
		return "", errors.New("a folder can't go inside itself")
	}
	return newRel, moveItem(site, rel, newRel, st.IsDir())
}

func moveItem(site *config.Site, from, to string, dir bool) error {
	if from == to {
		return nil
	}
	src, _ := notesPath(site, from)
	dst, err := notesPath(site, to)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil && !strings.EqualFold(from, to) {
		return errors.New(path.Base(to) + " already exists there")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	if err := followDatabase(site, from, to, dir); err != nil {
		return err
	}
	return fixLinks(site, from, to, dir)
}

// CopyNote makes a copy of a note (or a canvas or drawing) next to it
// ("Name 2.md") and returns it.
func CopyNote(site *config.Site, rel string) (string, error) {
	src, err := notesPath(site, rel)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	ext := fileExt(rel)
	base := strings.TrimSuffix(path.Base(rel), ext)
	for i := 2; ; i++ {
		newRel := path.Join(path.Dir(rel), fmt.Sprintf("%s %d%s", base, i, ext))
		if path.Dir(rel) == "." {
			newRel = fmt.Sprintf("%s %d%s", base, i, ext)
		}
		p, err := notesPath(site, newRel)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(p); err == nil {
			continue
		}
		return newRel, writeNew(p, b)
	}
}

// DeleteItem moves a note or folder to the Blup trash.
func DeleteItem(site *config.Site, rel string) (string, error) {
	return toTrash(site, rel)
}

// toTrash moves rel into a new dated folder of the trash, with a note of
// where it was (.origin), and returns where it went.
func toTrash(site *config.Site, rel string) (string, error) {
	src, err := notesPath(site, rel)
	if err != nil {
		return "", err
	}
	stamp := time.Now().Format("2006-01-02-150405")
	dir := filepath.Join(trashDir(site), stamp)
	for i := 2; ; i++ {
		if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
			break
		}
		dir = filepath.Join(trashDir(site), fmt.Sprintf("%s.%d", stamp, i))
	}
	dst := filepath.Join(dir, filepath.Base(src))
	if err := moveDir(src, dst); err != nil {
		return "", err
	}
	return dst, os.WriteFile(filepath.Join(dir, ".origin"), []byte(rel), 0o644)
}

// trashOrigin is where a trashed item was ("" when unknown: it goes back
// to the top of the notes).
func trashOrigin(trashed string) string {
	b, err := os.ReadFile(filepath.Join(filepath.Dir(trashed), ".origin"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// --- links ------------------------------------------------------------------

// mdLink is a Markdown link or image: [text](target "title").
var mdLink = regexp.MustCompile(`(\]\()(<[^>]+>|[^)\s]+)((?:\s+"[^"]*")?\))`)

// wikiLink is [[Name]], [[Name#heading]] or [[Name|alias]].
var wikiLink = regexp.MustCompile(`\[\[([^\]|#]+)((?:#[^\]|]*)?(?:\|[^\]]*)?)\]\]`)

// canvasFile is a file card of a canvas: "file":"Folder/Note.md".
var canvasFile = regexp.MustCompile(`("file"\s*:\s*)("(?:[^"\\]|\\.)*")`)

// fixCanvas points a canvas's file cards to where their files went (the
// rest of the file stays as written).
func fixCanvas(p string, moved func(string) (string, bool)) error {
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	text := string(b)
	out := canvasFile.ReplaceAllStringFunc(text, func(m string) string {
		parts := canvasFile.FindStringSubmatch(m)
		var file string
		if json.Unmarshal([]byte(parts[2]), &file) != nil {
			return m
		}
		to, ok := moved(file)
		if !ok {
			return m
		}
		q, _ := json.Marshal(to)
		return parts[1] + string(q)
	})
	if out == text {
		return nil
	}
	return os.WriteFile(p, []byte(out), 0o644)
}

// fixLinks rewrites, in every note, the links to what moved from "from" to
// "to" (a note, or a folder and all inside it), and the relative links
// inside what moved. A renamed note's [[wikilinks]] follow its new name.
func fixLinks(site *config.Site, from, to string, dir bool) error {
	root := site.ContentDir()
	// moved maps a path before the move to its path after.
	moved := func(p string) (string, bool) {
		switch {
		case p == from:
			return to, true
		case dir && strings.HasPrefix(p, from+"/"):
			return to + strings.TrimPrefix(p, from), true
		}
		return "", false
	}
	// before is where a note (by its path now) was before the move.
	before := func(p string) string {
		switch {
		case p == to:
			return from
		case dir && strings.HasPrefix(p, to+"/"):
			return from + strings.TrimPrefix(p, to)
		}
		return p
	}
	oldName, newName := strings.TrimSuffix(path.Base(from), ".md"), strings.TrimSuffix(path.Base(to), ".md")
	renamed := !dir && !strings.EqualFold(oldName, newName)

	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if isLink(d) {
			return nil
		}
		if strings.EqualFold(filepath.Ext(p), ".canvas") {
			return fixCanvas(p, moved)
		}
		if !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		relNow, _ := filepath.Rel(root, p)
		relNow = filepath.ToSlash(relNow)
		dirWas, dirNow := path.Dir(before(relNow)), path.Dir(relNow)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		text := string(b)
		out := mdLink.ReplaceAllStringFunc(text, func(m string) string {
			parts := mdLink.FindStringSubmatch(m)
			raw := strings.Trim(parts[2], "<>")
			if strings.Contains(raw, "://") || strings.HasPrefix(raw, "mailto:") || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "/") {
				return m
			}
			target, frag, _ := strings.Cut(raw, "#")
			dec, err := url.PathUnescape(target)
			if err != nil {
				return m
			}
			abs := path.Clean(path.Join(dirWas, dec))
			if strings.HasPrefix(abs, "..") {
				return m
			}
			if n, ok := moved(abs); ok {
				abs = n
			} else if dirWas == dirNow {
				return m // neither the link nor this note moved
			}
			rel, err := filepath.Rel(filepath.FromSlash(dirNow), filepath.FromSlash(abs))
			if err != nil {
				return m
			}
			link := strings.ReplaceAll(filepath.ToSlash(rel), " ", "%20")
			if frag != "" {
				link += "#" + frag
			}
			return parts[1] + link + parts[3]
		})
		if renamed {
			out = wikiLink.ReplaceAllStringFunc(out, func(m string) string {
				parts := wikiLink.FindStringSubmatch(m)
				name := strings.TrimSpace(parts[1])
				if strings.EqualFold(strings.TrimSuffix(path.Base(name), ".md"), oldName) {
					return "[[" + strings.TrimSuffix(name, path.Base(name)) + newName + parts[2] + "]]"
				}
				return m
			})
		}
		if out == text {
			return nil
		}
		return os.WriteFile(p, []byte(out), 0o644)
	})
}
