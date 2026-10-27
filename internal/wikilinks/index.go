// Package wikilinks converts Obsidian wikilinks ([[Note]], ![[image.png]], ...)
// into standard relative Markdown links that Hugo can render.
//
// Conventions used by the converter:
//
//   - Destinations are relative to the source note's directory. Every path
//     segment is percent-escaped only where CommonMark or URL parsing would
//     break (spaces become %20, plus % ( ) < > # ? [ ] \ ^ | " ` and control
//     characters); non-ASCII letters such as "ç" or "á" are kept as they are,
//     which is how Obsidian writes Markdown links itself.
//   - Link text is the alias when there is one. Otherwise it is the target as
//     typed ("Note", "Pasta/Nota", "Foto.pdf"); with a heading it follows the
//     way Obsidian displays it: "Note > Heading". [[#Heading]] shows "Heading".
//   - Heading anchors are the IDs Hugo generates with the default
//     autoHeadingIDType = "github": the heading's text is lowercased, each
//     space or hyphen becomes "-", letters (any script), digits and "_" are
//     kept, everything else is dropped; repeated IDs get "-1", "-2", ...
//     When the target note can be read, the link's heading is matched against
//     the note's real headings (ignoring case and punctuation, as Obsidian
//     does) so the anchor is exactly the one Hugo will emit, including custom
//     {#id} attributes and duplicate suffixes.
//   - Block references ([[Note#^id]]) are dropped: the result links to the note.
//   - Image embeds become Markdown images. A size given in the embed
//     (![[img.png|300]] or ![[img.png|300x200]]) is kept in the alt text after
//     a pipe, ![img|300](img.png), which is the Markdown image syntax Obsidian
//     itself understands; the site's image render hook splits it off. A
//     non-numeric alias (![[img.png|A caption]]) becomes the alt text; without
//     one the alt text is the file name without its extension.
//   - Note embeds (transclusions) and embeds of non-image attachments cannot
//     be rendered by Hugo; they become normal links and are reported as
//     warnings (Problem.Warning = true).
//   - Text inside frontmatter, fenced code blocks (``` and ~~~, also inside
//     blockquotes and lists) and inline code spans is never touched. Inline
//     code spans are matched within a single line. Wikilinks preceded by a
//     backslash are left alone. Unresolvable targets are left untouched and
//     reported.
package wikilinks

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Index knows every file of a vault and resolves wikilink targets the way
// Obsidian does. It is safe for concurrent use.
type Index struct {
	root  string
	files []string // vault-relative, slash-separated paths

	notes     map[string][]string // note name without .md -> paths
	notesFold map[string][]string // same, lowercased key
	names     map[string][]string // full file name -> paths
	namesFold map[string][]string // same, lowercased key

	mu       sync.Mutex
	headings map[string][]heading // cache of parsed headings per note path
}

// NewIndex walks vaultRoot and indexes every file, skipping dot directories
// such as .obsidian, .trash and .git.
func NewIndex(vaultRoot string) (*Index, error) {
	var files []string
	err := filepath.WalkDir(vaultRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != vaultRoot && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		rel, err := filepath.Rel(vaultRoot, p)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return newIndex(vaultRoot, files), nil
}

// newIndex builds an Index from vault-relative slash paths.
func newIndex(root string, files []string) *Index {
	idx := &Index{
		root:      root,
		files:     files,
		notes:     map[string][]string{},
		notesFold: map[string][]string{},
		names:     map[string][]string{},
		namesFold: map[string][]string{},
		headings:  map[string][]heading{},
	}
	sort.Strings(idx.files)
	for _, f := range idx.files {
		base := path.Base(f)
		idx.names[base] = append(idx.names[base], f)
		idx.namesFold[strings.ToLower(base)] = append(idx.namesFold[strings.ToLower(base)], f)
		if isNote(f) {
			stem := base[:len(base)-len(".md")]
			idx.notes[stem] = append(idx.notes[stem], f)
			idx.notesFold[strings.ToLower(stem)] = append(idx.notesFold[strings.ToLower(stem)], f)
		}
	}
	return idx
}

// Files returns the vault-relative, slash-separated paths of all indexed files.
func (idx *Index) Files() []string { return append([]string(nil), idx.files...) }

// Resolve returns the vault-relative path of the file a wikilink target
// (without "#heading" or "|alias") points to, as seen from the note at
// notePath. It reports false when nothing matches.
func (idx *Index) Resolve(notePath, target string) (string, bool) {
	notePath = path.Clean(filepath.ToSlash(notePath))
	target = strings.TrimSpace(filepath.ToSlash(target))
	if target == "" {
		return "", false
	}
	srcDir := path.Dir(notePath)

	var cands []string
	switch {
	case strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../"):
		cands = idx.byPath(path.Join(srcDir, target), false)
	case strings.HasPrefix(target, "/"):
		cands = idx.byPath(strings.TrimPrefix(target, "/"), false)
	case strings.Contains(target, "/"):
		cands = idx.byPath(target, true)
	default:
		cands = idx.byName(target)
	}
	if len(cands) == 0 {
		return "", false
	}
	return pick(cands, srcDir), true
}

// byName finds files by bare name: notes by name without extension, any file
// by its full name. Exact case wins over a case-insensitive match.
func (idx *Index) byName(name string) []string {
	if c := idx.notes[name]; len(c) > 0 {
		return c
	}
	if c := idx.names[name]; len(c) > 0 {
		return c
	}
	low := strings.ToLower(name)
	if c := idx.notesFold[low]; len(c) > 0 {
		return c
	}
	return idx.namesFold[low]
}

// byPath finds files whose path (with or without the .md of a note) equals
// p or, when suffix is true, ends with "/"+p.
func (idx *Index) byPath(p string, suffix bool) []string {
	match := func(fold bool) []string {
		want := p
		if fold {
			want = strings.ToLower(p)
		}
		var out []string
		for _, f := range idx.files {
			keys := []string{f}
			if isNote(f) {
				keys = append(keys, f[:len(f)-len(".md")])
			}
			for _, k := range keys {
				if fold {
					k = strings.ToLower(k)
				}
				if k == want || (suffix && strings.HasSuffix(k, "/"+want)) {
					out = append(out, f)
					break
				}
			}
		}
		return out
	}
	if out := match(false); len(out) > 0 {
		return out
	}
	return match(true)
}

// pick chooses among several candidates like Obsidian: a file in the source
// note's folder first, then the one fewest folder steps away from it, then
// the shortest path.
func pick(cands []string, srcDir string) string {
	if len(cands) == 1 {
		return cands[0]
	}
	src := splitDir(srcDir)
	type scored struct {
		p           string
		dist, depth int
	}
	list := make([]scored, len(cands))
	for i, c := range cands {
		dir := splitDir(path.Dir(c))
		common := 0
		for common < len(dir) && common < len(src) && dir[common] == src[common] {
			common++
		}
		list[i] = scored{p: c, dist: len(src) - common + len(dir) - common, depth: len(dir)}
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.dist != b.dist {
			return a.dist < b.dist // distance 0 means the same folder
		}
		if a.depth != b.depth {
			return a.depth < b.depth
		}
		if len(a.p) != len(b.p) {
			return len(a.p) < len(b.p)
		}
		return a.p < b.p
	})
	return list[0].p
}

func splitDir(d string) []string {
	d = path.Clean(d)
	if d == "." || d == "/" {
		return nil
	}
	return strings.Split(d, "/")
}

// noteHeadings returns the parsed headings of a vault note, reading it from
// disk once. A missing or unreadable file yields nil.
func (idx *Index) noteHeadings(notePath string) []heading {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if h, ok := idx.headings[notePath]; ok {
		return h
	}
	var h []heading
	if b, err := os.ReadFile(filepath.Join(idx.root, filepath.FromSlash(notePath))); err == nil {
		h = parseHeadings(string(b))
	}
	idx.headings[notePath] = h
	return h
}

func isNote(p string) bool {
	return strings.EqualFold(path.Ext(p), ".md")
}

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".svg": true, ".webp": true, ".avif": true,
}

func isImage(p string) bool {
	return imageExts[strings.ToLower(path.Ext(p))]
}
