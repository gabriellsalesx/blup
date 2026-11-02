package scaffold

import (
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// translationKey links a note to its translations (see internal/app).
const translationKey = "translationKey"

type noteInfo struct {
	rel    string // relative to the site folder, slash-separated
	hidden bool   // publish: false or private: true
	draft  bool
	key    string // translationKey, "" without one
}

// unpublished finds the notes of the site that must stay off it: hidden
// are the notes with "publish: false" or "private: true" (and the
// translations of such a note, which share its fate); quiet lists them
// with the drafts, the notes whose links become plain text. Paths are
// relative to the site folder, slash-separated.
func unpublished(s *config.Site) (hidden map[string]bool, quiet []string) {
	root := s.ContentDir()
	var notes []noteInfo
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ".md") || !d.Type().IsRegular() {
			return nil
		}
		n, err := obsidian.ReadNote(p)
		if err != nil {
			return nil // Hugo reports a broken front matter itself
		}
		rel, _ := filepath.Rel(root, p)
		key, _ := n.Props[translationKey].(string)
		notes = append(notes, noteInfo{
			rel: filepath.ToSlash(rel), hidden: obsidian.Unpublished(n.Props),
			draft: obsidian.Yes(n.Props["draft"]), key: key,
		})
		return nil
	})

	hidden = map[string]bool{}
	// Keys and paths of the hidden originals, by folder.
	hiddenKeys := map[string]bool{}
	for _, n := range notes {
		if n.hidden {
			hidden[n.rel] = true
			dir := path.Dir(n.rel)
			hiddenKeys[dir+"\x00"+strings.TrimSuffix(n.rel, path.Ext(n.rel))] = true
			if n.key != "" {
				hiddenKeys[dir+"\x00"+n.key] = true
			}
		}
	}
	// A translation (Posts/en/Note.md) of a hidden note (Posts/Note.md,
	// or the note with the same translationKey) is hidden too.
	for _, n := range notes {
		dir := path.Dir(n.rel)
		lang := path.Base(dir)
		if n.hidden || !slices.Contains(config.SiteLangs, lang) || lang == s.Lang {
			continue
		}
		parent := path.Dir(dir)
		same := path.Join(parent, path.Base(n.rel))
		if hidden[same] || n.key != "" && hiddenKeys[parent+"\x00"+n.key] {
			hidden[n.rel] = true
		}
	}
	for _, n := range notes {
		if hidden[n.rel] || n.draft {
			quiet = append(quiet, n.rel)
		}
	}
	slices.Sort(quiet)
	return hidden, quiet
}

// literalGlob is a mount pattern that matches the file at rel. Characters
// with a meaning in globs become "?" (any one character): an escape would
// not survive on Windows, where Hugo turns "\" into "/". At worst a note
// whose name differs only in those characters is left out too.
func literalGlob(rel string) string {
	var b strings.Builder
	runes := []rune(rel)
	for i, r := range runes {
		switch {
		case strings.ContainsRune(`*?[]{}\!,`, r),
			r == ' ' && (i == 0 || i == len(runes)-1):
			b.WriteRune('?')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
