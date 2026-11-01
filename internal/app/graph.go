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
	"sync"
	"time"

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
func NoteGraph(site *config.Site) (Graph, error) {
	g, _, err := noteGraph(site)
	return g, err
}

// A note as read for the map: its tags and date, and its links (with the
// line around each, for "mentioned in"). Kept per file and read again only
// when the file changes (its time or size), so opening a note or the map
// in a large vault reads just what changed since.
type noteScan struct {
	mod     time.Time
	size    int64
	tags    []string
	created string
	links   []scanLink
}

type scanLink struct {
	target string // a path ([text](path.md), a canvas' file) or a name ([[Name]])
	wiki   bool
	line   Snippet
}

var (
	scanMu    sync.Mutex
	scanCache = map[string]map[string]*noteScan{} // notes folder -> path -> scan
)

// noteGraph is NoteGraph, with each link's line: links[from] lists, for
// each link of from that reaches a note, where it goes and its line.
func noteGraph(site *config.Site) (Graph, map[string][]resolvedLink, error) {
	root := site.ContentDir()
	var g Graph
	scanMu.Lock()
	old := scanCache[root]
	scanMu.Unlock()
	scans := make(map[string]*noteScan, len(old))
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
		info, err := d.Info()
		if err != nil {
			return nil
		}
		n.Created = info.ModTime().Format("2006-01-02")
		if kind != "drawing" {
			sc := old[rel]
			if sc == nil || !sc.mod.Equal(info.ModTime()) || sc.size != info.Size() {
				sc = scanNote(p, kind, info)
			}
			if sc != nil {
				scans[rel] = sc
				if kind == "note" {
					n.Tags = sc.tags
					if sc.created != "" {
						n.Created = sc.created
					}
				}
			}
		}
		g.Nodes = append(g.Nodes, n)
		return nil
	})
	if err != nil {
		return g, nil, err
	}
	// Only the files still there are kept.
	scanMu.Lock()
	scanCache[root] = scans
	scanMu.Unlock()

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
	links := map[string][]resolvedLink{}
	for _, n := range g.Nodes {
		sc := scans[n.Path]
		if sc == nil {
			continue
		}
		for _, l := range sc.links {
			var to string
			switch {
			case l.wiki:
				to = findName(l.target, n.Path)
			case n.Kind == "canvas" || strings.HasPrefix(l.target, "/"):
				to = find(l.target)
			default:
				to = find(path.Join(path.Dir(n.Path), l.target))
			}
			if to == "" || to == n.Path {
				continue
			}
			links[n.Path] = append(links[n.Path], resolvedLink{to: to, line: l.line})
			if e := (GraphEdge{From: n.Path, To: to}); !seen[e] {
				seen[e] = true
				g.Edges = append(g.Edges, e)
			}
		}
	}
	slices.SortFunc(g.Nodes, func(a, b GraphNode) int { return strings.Compare(a.Path, b.Path) })
	return g, links, nil
}

type resolvedLink struct {
	to   string
	line Snippet
}

// scanNote reads a note (or a canvas) for the map; nil when unreadable.
func scanNote(p, kind string, info fs.FileInfo) *noteScan {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	text := string(b)
	sc := &noteScan{mod: info.ModTime(), size: info.Size()}
	if kind == "canvas" {
		for _, f := range canvasFiles(text) {
			sc.links = append(sc.links, scanLink{target: f})
		}
		return sc
	}
	if kind == "note" {
		sc.tags, sc.created = noteMeta(text)
	}
	text = codeSpan.ReplaceAllString(text, "")
	for _, i := range mdLink.FindAllStringSubmatchIndex(text, -1) {
		raw := strings.Trim(text[i[4]:i[5]], "<>")
		if strings.Contains(raw, "://") || strings.HasPrefix(raw, "mailto:") || strings.HasPrefix(raw, "#") {
			continue
		}
		target, _, _ := strings.Cut(raw, "#")
		target, _, _ = strings.Cut(target, "?")
		if t, err := url.PathUnescape(target); err == nil {
			target = t
		}
		// The link starts at its [ (the match starts at "](").
		at := strings.LastIndex(text[:i[0]], "[")
		if nl := strings.LastIndex(text[:i[0]], "\n"); at < 0 || at < nl {
			at = i[0]
		}
		sc.links = append(sc.links, scanLink{target: target, line: snippet(text, at, i[1])})
	}
	for _, i := range wikiLink.FindAllStringSubmatchIndex(text, -1) {
		sc.links = append(sc.links, scanLink{target: strings.TrimSpace(text[i[2]:i[3]]), wiki: true, line: snippet(text, i[0], i[1])})
	}
	return sc
}

// snippet is the line around the link at text[at:end], as it reads.
func snippet(text string, at, end int) Snippet {
	ls := strings.LastIndex(text[:at], "\n") + 1
	le := strings.Index(text[end:], "\n")
	if le < 0 {
		le = len(text)
	} else {
		le += end
	}
	return Snippet{
		Before: clip(plainLine(text[ls:at], true), 70, true),
		Link:   linkLabel(text[at:end]),
		After:  clip(plainLine(text[end:le], false), 90, false),
	}
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
	g, links, err := noteGraph(site)
	if err != nil {
		return nil, err
	}
	out := []Mention{}
	for _, n := range g.Nodes {
		var lines []Snippet
		found := false
		for _, l := range links[n.Path] {
			if l.to != rel {
				continue
			}
			found = true
			if len(lines) < 3 && n.Kind == "note" {
				lines = append(lines, l.line)
			}
		}
		if found {
			out = append(out, Mention{Path: n.Path, Name: n.Name, Folder: n.Folder, Lines: lines})
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
