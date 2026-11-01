package app

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gabriellsalesx/blup/internal/config"
	"golang.org/x/text/unicode/norm"
)

// SearchHit is a note found by SearchNotes: by its name, its text, or
// both; Lines are up to three lines where the text has the words.
type SearchHit struct {
	Path   string    `json:"path"`
	Name   string    `json:"name"`
	Folder string    `json:"folder"`
	InName bool      `json:"inName"`
	Lines  []Snippet `json:"lines"` // Link holds the words found
}

const (
	searchMax     = 60 // notes
	searchLines   = 3  // lines per note
	searchMaxFile = 2 << 20
)

// SearchNotes finds query in the notes' names and texts, ignoring case
// and accents ("acao" finds "Ação"). Notes whose name has it come first.
func SearchNotes(site *config.Site, query string) ([]SearchHit, error) {
	q := fold(strings.TrimSpace(query))
	if utf8.RuneCountInString(q) < 2 {
		return []SearchHit{}, nil
	}
	root := site.ContentDir()
	var named, texts []SearchHit
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
		if len(named)+len(texts) >= searchMax*2 || !strings.EqualFold(path.Ext(d.Name()), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		name := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
		folder := path.Dir(rel)
		if folder == "." {
			folder = ""
		}
		hit := SearchHit{Path: rel, Name: name, Folder: folder, InName: strings.Contains(fold(name), q)}
		if info, err := d.Info(); err == nil && info.Size() <= searchMaxFile {
			if b, err := os.ReadFile(p); err == nil {
				hit.Lines = findLines(string(b), q)
			}
		}
		switch {
		case hit.InName:
			named = append(named, hit)
		case len(hit.Lines) > 0:
			texts = append(texts, hit)
		}
		return nil
	})
	out := append(named, texts...)
	if len(out) > searchMax {
		out = out[:searchMax]
	}
	return out, err
}

// findLines are the lines of text (front matter left out) with q, folded.
func findLines(text, q string) []Snippet {
	_, body := splitFrontMatter(text)
	var out []Snippet
	for line := range strings.Lines(body) {
		line = strings.TrimRight(line, "\r\n")
		f, at := foldIndex(line)
		i := strings.Index(f, q)
		if i < 0 {
			continue
		}
		// Back to the line's own bytes: at maps each folded byte to it.
		s, e := at[i], at[i+len(q)]
		out = append(out, Snippet{
			Before: clip(plainLine(line[:s], true), 50, true),
			Link:   line[s:e],
			After:  clip(plainLine(line[e:], false), 90, false),
		})
		if len(out) == searchLines {
			break
		}
	}
	return out
}

// splitFrontMatter cuts the YAML block at the top of a note.
func splitFrontMatter(text string) (fm, body string) {
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return "", text
	}
	rest := text[strings.Index(text, "\n")+1:]
	for i := 0; i < len(rest); {
		j := strings.IndexByte(rest[i:], '\n')
		line := rest[i:]
		if j >= 0 {
			line = rest[i : i+j]
		}
		if strings.TrimRight(line, "\r") == "---" {
			if j < 0 {
				return rest[:i], ""
			}
			return rest[:i], rest[i+j+1:]
		}
		if j < 0 {
			break
		}
		i += j + 1
	}
	return "", text
}

// fold is s in lower case without accents.
func fold(s string) string {
	f, _ := foldIndex(s)
	return f
}

// foldIndex is fold(s) and, for each byte of it (and one past the end),
// where that byte's character starts in s.
func foldIndex(s string) (string, []int) {
	var b strings.Builder
	b.Grow(len(s))
	at := make([]int, 0, len(s)+1)
	for i, r := range s {
		r = foldRune(r)
		n := utf8.RuneLen(r)
		if n < 0 {
			r, n = utf8.RuneError, 3
		}
		for range n {
			at = append(at, i)
		}
		b.WriteRune(r)
	}
	at = append(at, len(s))
	return b.String(), at
}

// foldRune is r lower case, its accent taken off (ç → c, ã → a).
func foldRune(r rune) rune {
	if r < utf8.RuneSelf {
		return unicode.ToLower(r)
	}
	r = unicode.ToLower(r)
	if d := norm.NFD.String(string(r)); d != "" {
		base, _ := utf8.DecodeRuneInString(d)
		if base != utf8.RuneError {
			return base
		}
	}
	return r
}
