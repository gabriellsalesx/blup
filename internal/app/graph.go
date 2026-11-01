package app

import (
	"encoding/json"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"go.yaml.in/yaml/v3"
)

// The notes and the links between them, for the Studio's map of notes
// (and for "mentioned in"): every note, canvas and drawing of the notes
// folder, and an edge for each link from one to another, whatever its
// form ([text](path.md), [[Name]], a note placed on a canvas).

// GraphNode is a note (or a canvas, or a drawing).
type GraphNode struct {
	Path    string   `json:"path"` // relative to the notes folder, with "/"
	Name    string   `json:"name"`
	Folder  string   `json:"folder"`
	Kind    string   `json:"kind"` // note, canvas, drawing
	Tags    []string `json:"tags"`
	Created string   `json:"created"` // 2006-01-02: its created property, else the file's date (the map's animation)
}

// GraphEdge is a link from one note to another.
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Graph is the map of the notes.
type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// Code is not read for links (a link inside code is an example).
var codeSpan = regexp.MustCompile("(?s)```.*?```|`[^`\n]*`")

// NoteGraph reads the notes folder into the map of notes.
func NoteGraph(site *config.Site) (Graph, error) { return noteGraph(site, nil) }

// noteGraph is NoteGraph; found, when given, hears every link of a note
// to another: the note's text (code taken out) and where the link is in it.
func noteGraph(site *config.Site, found func(from, to, text string, at, end int)) (Graph, error) {
	root := site.ContentDir()
	var g Graph
	texts := map[string]string{} // path -> what to read links from
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
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		kind := graphKind(rel)
		// A section's _index.md is the site's list page, not a note.
		if kind == "" || strings.EqualFold(d.Name(), "_index.md") {
			return nil
		}
		folder := path.Dir(rel)
		if folder == "." {
			folder = ""
		}
		n := GraphNode{Path: rel, Name: graphName(rel), Folder: folder, Kind: kind, Tags: []string{}}
		if info, err := d.Info(); err == nil {
			n.Created = info.ModTime().Format("2006-01-02")
		}
		if kind != "drawing" {
			if b, err := os.ReadFile(p); err == nil {
				texts[rel] = string(b)
				if kind == "note" {
					var created string
					n.Tags, created = noteMeta(string(b))
					if created != "" {
						n.Created = created
					}
				}
			}
		}
		g.Nodes = append(g.Nodes, n)
		return nil
	})
	if err != nil {
		return g, err
	}

	// Finding a note: by its path (with or without .md, any case), or by
	// its name, one in the same folder first.
	byPath := map[string]string{}
	byName := map[string][]string{}
	for _, n := range g.Nodes {
		byPath[strings.ToLower(n.Path)] = n.Path
		if n.Kind == "note" {
			byPath[strings.ToLower(strings.TrimSuffix(n.Path, path.Ext(n.Path)))] = n.Path
		}
		byName[strings.ToLower(n.Name)] = append(byName[strings.ToLower(n.Name)], n.Path)
	}
	find := func(p string) string { return byPath[strings.ToLower(strings.TrimPrefix(p, "/"))] }
	findName := func(name, from string) string {
		if p := find(name); p != "" {
			return p
		}
		list := byName[strings.ToLower(path.Base(strings.TrimSuffix(name, ".md")))]
		for _, p := range list {
			if path.Dir(p) == path.Dir(from) {
				return p
			}
		}
		if len(list) > 0 {
			return list[0]
		}
		return ""
	}

	seen := map[GraphEdge]bool{}
	link := func(from, to string) {
		e := GraphEdge{From: from, To: to}
		if to == "" || to == from || seen[e] {
			return
		}
		seen[e] = true
		g.Edges = append(g.Edges, e)
	}
	for _, n := range g.Nodes {
		text, ok := texts[n.Path]
		if !ok {
			continue
		}
		if n.Kind == "canvas" {
			for _, f := range canvasFiles(text) {
				link(n.Path, find(f))
			}
			continue
		}
		text = codeSpan.ReplaceAllString(text, "")
		for _, i := range mdLink.FindAllStringSubmatchIndex(text, -1) {
			m := []string{text[i[0]:i[1]], text[i[2]:i[3]], text[i[4]:i[5]]}
			raw := strings.Trim(m[2], "<>")
			if strings.Contains(raw, "://") || strings.HasPrefix(raw, "mailto:") || strings.HasPrefix(raw, "#") {
				continue
			}
			target, _, _ := strings.Cut(raw, "#")
			target, _, _ = strings.Cut(target, "?")
			if t, err := url.PathUnescape(target); err == nil {
				target = t
			}
			if !strings.HasPrefix(target, "/") {
				target = path.Join(path.Dir(n.Path), target)
			}
			to := find(target)
			link(n.Path, to)
			if found != nil && to != "" && to != n.Path {
				// The link starts at its [ (the match starts at "](").
				at := strings.LastIndex(text[:i[0]], "[")
				if nl := strings.LastIndex(text[:i[0]], "\n"); at < 0 || at < nl {
					at = i[0]
				}
				found(n.Path, to, text, at, i[1])
			}
		}
		for _, i := range wikiLink.FindAllStringSubmatchIndex(text, -1) {
			to := findName(strings.TrimSpace(text[i[2]:i[3]]), n.Path)
			link(n.Path, to)
			if found != nil && to != "" && to != n.Path {
				found(n.Path, to, text, i[0], i[1])
			}
		}
	}
	slices.SortFunc(g.Nodes, func(a, b GraphNode) int { return strings.Compare(a.Path, b.Path) })
	return g, nil
}

func graphKind(rel string) string {
	l := strings.ToLower(rel)
	switch {
	case strings.HasSuffix(l, ".md"):
		return "note"
	case strings.HasSuffix(l, ".canvas"):
		return "canvas"
	case strings.HasSuffix(l, ".draw"):
		return "drawing"
	}
	return ""
}

func graphName(rel string) string {
	base := path.Base(rel)
	return strings.TrimSuffix(base, fileExt(base))
}

// noteMeta reads a note's tags property (a list, or a text with commas)
// and its created date.
func noteMeta(text string) (tags []string, created string) {
	out := []string{}
	front, _ := splitFront(text)
	if front == "" {
		return out, ""
	}
	var fm struct {
		Tags    any `yaml:"tags"`
		Created any `yaml:"created"`
	}
	if yaml.Unmarshal([]byte(front), &fm) != nil {
		return out, ""
	}
	switch c := fm.Created.(type) {
	case string:
		created = c
	case interface{ Format(string) string }:
		created = c.Format("2006-01-02")
	}
	if len(created) > 10 {
		created = created[:10]
	}
	var raw []string
	switch x := fm.Tags.(type) {
	case string:
		raw = strings.Split(x, ",")
	case []any:
		for _, i := range x {
			if s, ok := i.(string); ok {
				raw = append(raw, s)
			}
		}
	}
	for _, t := range raw {
		if t = strings.TrimPrefix(strings.TrimSpace(t), "#"); t != "" {
			out = append(out, t)
		}
	}
	return out, created
}

// canvasFiles are the files placed on a canvas (JSON Canvas file cards).
func canvasFiles(text string) []string {
	var c struct {
		Nodes []struct {
			Type string `json:"type"`
			File string `json:"file"`
		} `json:"nodes"`
	}
	if json.Unmarshal([]byte(text), &c) != nil {
		return nil
	}
	var out []string
	for _, n := range c.Nodes {
		if n.Type == "file" && n.File != "" {
			out = append(out, n.File)
		}
	}
	return out
}

// Mention is a note that links to another one, and the lines where it
// does, for "mentioned in": each line in three parts, the link (as it
// reads) between what comes before and after it, plain text.
type Mention struct {
	Path   string    `json:"path"`
	Name   string    `json:"name"`
	Folder string    `json:"folder"`
	Lines  []Snippet `json:"lines"`
}

// Snippet is a line around a link.
type Snippet struct {
	Before string `json:"before"`
	Link   string `json:"link"`
	After  string `json:"after"`
}

// Mentions are the notes (and canvases) that link to rel, with up to
// three lines of each, in the order of the notes.
func Mentions(site *config.Site, rel string) ([]Mention, error) {
	lines := map[string][]Snippet{}
	g, err := noteGraph(site, func(from, to, text string, at, end int) {
		if to != rel || len(lines[from]) >= 3 {
			return
		}
		ls := strings.LastIndex(text[:at], "\n") + 1
		le := strings.Index(text[end:], "\n")
		if le < 0 {
			le = len(text)
		} else {
			le += end
		}
		lines[from] = append(lines[from], Snippet{
			Before: clip(plainLine(text[ls:at], true), 70, true),
			Link:   linkLabel(text[at:end]),
			After:  clip(plainLine(text[end:le], false), 90, false),
		})
	})
	if err != nil {
		return nil, err
	}
	out := []Mention{}
	for _, n := range g.Nodes {
		for _, e := range g.Edges {
			if e.From == n.Path && e.To == rel {
				out = append(out, Mention{Path: n.Path, Name: n.Name, Folder: n.Folder, Lines: lines[n.Path]})
				break
			}
		}
	}
	return out, nil
}

var (
	lineMarks  = regexp.MustCompile(`^\s*(?:#{1,6}\s+|(?:>\s?)+|[-*+]\s+(?:\[[ xX]\]\s+)?|\d+[.)]\s+)*`)
	inlineMark = regexp.MustCompile(`\*\*|__|~~|==|%%`)
	mdText     = regexp.MustCompile(`!?\[([^\]\n]*)\]\([^)\n]*\)`)
	wikiText   = regexp.MustCompile(`!?\[\[([^\]|#\n]+)(?:#[^\]|\n]*)?(?:\|([^\]\n]*))?\]\]`)
)

// plainLine is a piece of a line as it reads: links as their text, no
// Markdown marks (start: the piece begins the line, with its # > - marks).
func plainLine(s string, start bool) string {
	if start {
		s = lineMarks.ReplaceAllString(s, "")
	}
	s = mdText.ReplaceAllString(s, "$1")
	s = wikiText.ReplaceAllStringFunc(s, func(m string) string { return linkLabel(m) })
	return inlineMark.ReplaceAllString(s, "")
}

// linkLabel is how a link reads: [text](…) is text, [[Name|text]] is
// text, [[Name]] is Name.
func linkLabel(s string) string {
	if m := wikiText.FindStringSubmatch(s); m != nil {
		if m[2] != "" {
			return m[2]
		}
		return strings.TrimSpace(m[1])
	}
	if m := mdText.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return s
}

// clip keeps up to n letters of s, the end of it (tail) or its start,
// with … where it was cut.
func clip(s string, n int, tail bool) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if tail {
		return "…" + strings.TrimLeft(string(r[len(r)-n:]), " ")
	}
	return strings.TrimRight(string(r[:n]), " ") + "…"
}
