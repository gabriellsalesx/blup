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
func NoteGraph(site *config.Site) (Graph, error) {
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
		for _, m := range mdLink.FindAllStringSubmatch(text, -1) {
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
			link(n.Path, find(target))
		}
		for _, m := range wikiLink.FindAllStringSubmatch(text, -1) {
			link(n.Path, findName(strings.TrimSpace(m[1]), n.Path))
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
