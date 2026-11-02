package app

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// Notes made quickly in a vault of notes, already in order: the note of
// the day, and new notes that take the tags their folder's notes have.

// FolderTags are the tags a new note in folder gets: the database's tag
// when the folder holds a database's records, else the tags at least half
// of the folder's notes have (with two notes or more, so one note alone
// doesn't decide).
func FolderTags(site *config.Site, folder string) []string {
	if _, s := DatabaseOf(site, path.Join(folder, "x.md")); s != nil && s.Tag != "" {
		return []string{s.Tag}
	}
	dir, err := notesPath(site, folder)
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	count, notes := map[string]int{}, 0
	var order []string
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".md") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		notes++
		tags, _ := noteMeta(string(b))
		for _, t := range tags {
			if count[t] == 0 {
				order = append(order, t)
			}
			count[t]++
		}
	}
	if notes < 2 {
		return nil
	}
	var out []string
	for _, t := range order {
		if count[t]*2 >= notes {
			out = append(out, t)
		}
	}
	return out
}

// DailyNote opens the note of the day (folder/2006-01-02.md, made with
// tag when it isn't there yet) and adds the time at its end (## 15:04),
// to write under it. It returns the note's path.
func DailyNote(site *config.Site, folder, tag string, now time.Time) (string, error) {
	if folder == "" {
		return "", errors.New(i18nInvalidName())
	}
	rel := path.Join(folder, now.Format("2006-01-02")+".md")
	p, err := notesPath(site, rel)
	if err != nil {
		return "", err
	}
	if err := MkdirInside(site.ContentDir(), filepath.Dir(p)); err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		day := now.Format("2006-01-02")
		b = []byte("---\ncreated: " + day + "\nupdated: " + day + "\n---\n")
		if err := writeNew(site.ContentDir(), p, b); err != nil {
			return "", err
		}
		tags := FolderTags(site, folder)
		if tag != "" && !slices.Contains(tags, tag) {
			tags = append([]string{tag}, tags...)
		}
		if len(tags) > 0 {
			if err := obsidian.SetProperties(p, obsidian.Prop{Key: "tags", Value: tags}); err != nil {
				return "", err
			}
		}
		if b, err = os.ReadFile(p); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	heading := "## " + now.Format("15:04")
	text := strings.TrimRight(string(b), "\n")
	if strings.HasSuffix(text, "\n"+heading) {
		return rel, nil // opened again the same minute: that time is there already
	}
	return rel, WriteInside(site.ContentDir(), p, []byte(text+"\n\n"+heading+"\n\n"))
}

// VaultTags are the tags used in the notes' properties, the most used
// first (the tags field's suggestions).
func VaultTags(site *config.Site) ([]string, error) {
	count := map[string]int{}
	root := site.ContentDir()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		if b, err := os.ReadFile(p); err == nil {
			tags, _ := noteMeta(string(b))
			for _, t := range tags {
				count[t]++
			}
		}
		return nil
	})
	out := make([]string, 0, len(count))
	for t := range count {
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b string) int {
		if count[a] != count[b] {
			return count[b] - count[a]
		}
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	return out, err
}
