package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/hugo"
	"github.com/gabriellsalesx/blup/internal/scaffold"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Studio is what the window calls: the sites, their notes and the preview.
// One site is open at a time.
type Studio struct {
	lang    string // "pt" or "en"
	mu      sync.Mutex
	site    *config.Site
	preview *exec.Cmd
	url     string
	// login is the Blup Cloud sign in waiting for the browser.
	login  *blupcloud.Login
	cancel context.CancelFunc
	// live keeps the open site synced through Blup Cloud (live plans).
	live *app.Live
}

// SiteInfo describes a site in the list.
type SiteInfo struct {
	Dir    string `json:"dir"`
	Title  string `json:"title"`
	Model  string `json:"model"`
	Folder string `json:"folder"` // the notes folder
	Kind   string `json:"kind"`   // "" a site, "notes" a vault of notes
}

// Node is a folder or a note of the site.
type Node struct {
	Name     string `json:"name"`
	Path     string `json:"path"` // relative to the notes folder, with "/"
	Dir      bool   `json:"dir"`
	Children []Node `json:"children,omitempty"`
}

// Sites lists the sites Blup knows (made with the CLI or the Studio).
func (s *Studio) Sites() ([]SiteInfo, error) {
	g, err := config.LoadGlobal()
	if err != nil {
		return nil, err
	}
	out := []SiteInfo{}
	for _, dir := range g.ExistingSites() {
		site, err := config.LoadSite(dir)
		if err != nil {
			continue
		}
		out = append(out, info(site))
	}
	return out, nil
}

func info(site *config.Site) SiteInfo {
	return SiteInfo{Dir: site.Dir(), Title: site.Title, Model: site.Model, Folder: site.ContentDir(), Kind: site.Kind}
}

// Open makes dir the open site (stopping the preview of the previous one)
// and returns its notes.
func (s *Studio) Open(dir string) (Node, error) {
	site, err := config.LoadSite(dir)
	if err != nil {
		return Node{}, err
	}
	s.StopPreview()
	closeTerminals(s)
	s.mu.Lock()
	s.site = site
	s.mu.Unlock()
	watchNotes(site.ContentDir())
	s.startLive(site)
	s.remember(dir)
	return s.Tree()
}

func (s *Studio) current() (*config.Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.site == nil {
		return nil, errors.New("no site open")
	}
	return s.site, nil
}

// Tree is the open site's folders and notes, folders first, by name.
// Hidden folders (.obsidian, .git...) are left out.
func (s *Studio) Tree() (Node, error) {
	site, err := s.current()
	if err != nil {
		return Node{}, err
	}
	root := site.ContentDir()
	authors := app.AuthorsFolder(site)
	attachments := app.AttachmentsFolder(site) // its own screen too
	var walk func(dir, rel string) ([]Node, error)
	walk = func(dir, rel string) ([]Node, error) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		var dirs, notes []Node
		for _, e := range entries {
			name := e.Name()
			// The authors and attachments folders have their own screens.
			if strings.HasPrefix(name, ".") || (rel == "" && (slices.Contains(scaffold.AgentGuides, name) || (authors != "" && name == authors) || name == attachments)) {
				continue
			}
			p := strings.TrimPrefix(rel+"/"+name, "/")
			if e.IsDir() {
				kids, err := walk(filepath.Join(dir, name), p)
				if err != nil {
					return nil, err
				}
				dirs = append(dirs, Node{Name: name, Path: p, Dir: true, Children: kids})
			} else if strings.EqualFold(filepath.Ext(name), ".md") {
				notes = append(notes, Node{Name: strings.TrimSuffix(name, filepath.Ext(name)), Path: p})
			}
		}
		byName := func(a, b Node) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) }
		slices.SortFunc(dirs, byName)
		slices.SortFunc(notes, byName)
		return append(dirs, notes...), nil
	}
	kids, err := walk(root, "")
	return Node{Name: site.Title, Dir: true, Children: kids}, err
}

// notePath turns a path from the window into a file of the notes folder,
// refusing anything outside it.
func (s *Studio) notePath(rel string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	root := filepath.Clean(site.ContentDir())
	p := filepath.Join(root, filepath.FromSlash(rel))
	if p != root && !strings.HasPrefix(p, root+string(filepath.Separator)) {
		return "", errors.New("outside the notes folder")
	}
	return p, nil
}

// Read returns a note's text.
func (s *Studio) Read(rel string) (string, error) {
	p, err := s.notePath(rel)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	return string(b), err
}

// Write saves a note (the preview picks it up by itself).
func (s *Studio) Write(rel, text string) error {
	p, err := s.notePath(rel)
	if err != nil {
		return err
	}
	if filepath.Ext(p) != ".md" {
		return errors.New("only notes (.md) can be saved")
	}
	return os.WriteFile(p, []byte(text), 0o644)
}

var serverURL = regexp.MustCompile(`https?://(localhost|127\.0\.0\.1):\d+/\S*`)

// StartPreview runs the site's live preview (hugo server, as blup preview)
// and returns its address. Saving a note updates it.
func (s *Studio) StartPreview() (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	if site.IsNotes() {
		return "", errors.New("a vault of notes has no site to preview")
	}
	s.mu.Lock()
	if s.preview != nil {
		defer s.mu.Unlock()
		return s.url, nil
	}
	s.mu.Unlock()

	ctx := context.Background()
	bin, ok := hugo.Find(ctx)
	if !ok {
		if bin, err = hugo.Download(ctx, nil); err != nil {
			return "", err
		}
	}
	if err := scaffold.Write(site); err != nil {
		return "", err
	}
	cmd := bin.Command(ctx, site.Dir(), "server", "--config", scaffold.PreviewConfig, "--renderToMemory", "--buildDrafts",
		"--disableLiveReload", "--bind", "127.0.0.1", "--baseURL", "http://localhost/", "--logLevel", "info")
	cmd.Env = append(os.Environ(), "HUGO_BLUP_MODE=preview")
	if !canPlayMP3() {
		// No player the window can't play (see media.go).
		cmd.Env = append(cmd.Env, "HUGO_BLUP_NO_MP3=1")
	}
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		return "", err
	}
	go func() { cmd.Wait(); pw.Close() }()

	found := make(chan string, 1)
	var log strings.Builder
	go func() {
		sc := bufio.NewScanner(pr)
		sent := false
		for sc.Scan() {
			line := sc.Text()
			switch {
			case !sent && serverURL.MatchString(line):
				found <- serverURL.FindString(line)
				sent = true
			case !sent:
				log.WriteString(line + "\n")
			case strings.Contains(line, "Total in"):
				// A rebuild is done: the window swaps in the new page.
				application.Get().Event.Emit("previewChanged", "")
			}
		}
		if !sent {
			close(found)
		}
	}()
	select {
	case url, ok := <-found:
		if !ok {
			return "", errors.New("the preview stopped: " + strings.TrimSpace(log.String()))
		}
		s.mu.Lock()
		s.preview, s.url = cmd, url
		s.mu.Unlock()
		return url, nil
	case <-time.After(2 * time.Minute):
		cmd.Process.Kill()
		return "", errors.New("the preview took too long to start")
	}
}

// StopPreview stops the preview, if running.
func (s *Studio) StopPreview() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.preview != nil && s.preview.Process != nil {
		s.preview.Process.Kill()
	}
	s.preview, s.url = nil, ""
}

func (s *Studio) shutdown() {
	s.StopPreview()
	closeTerminals(s)
}

// PreviewPage is the preview address of a note's page ("" when the preview
// is off or the page is unknown). The preview lists every note's page in
// blup-map.json.
func (s *Studio) PreviewPage(rel string) (string, error) {
	base, pages, err := s.previewMap()
	if err != nil || base == "" {
		return "", err
	}
	for _, p := range pages {
		if strings.EqualFold(p.File, rel) {
			u, err := url.Parse(base)
			if err != nil {
				return "", err
			}
			u.Path = p.URL
			return u.String(), nil
		}
	}
	return "", nil
}

// NoteAt is the note behind a page of the preview (its address path, as
// the reader followed a link there), "" for a page without a note.
func (s *Studio) NoteAt(page string) (string, error) {
	_, pages, err := s.previewMap()
	if err != nil {
		return "", err
	}
	want := cleanPage(page)
	for _, p := range pages {
		if cleanPage(p.URL) == want {
			return p.File, nil
		}
	}
	return "", nil
}

// cleanPage compares page addresses: unescaped, with a trailing "/".
func cleanPage(p string) string {
	if u, err := url.PathUnescape(p); err == nil {
		p = u
	}
	p = strings.TrimSuffix(p, "index.html")
	if !strings.HasSuffix(p, "/") && path.Ext(p) == "" {
		p += "/"
	}
	return p
}

type mapPage struct{ File, URL string }

// previewMap reads blup-map.json, the page of each note, from the running
// preview ("" base when the preview is off).
func (s *Studio) previewMap() (string, []mapPage, error) {
	s.mu.Lock()
	base := s.url
	s.mu.Unlock()
	if base == "" {
		return "", nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(base, "/")+"/blup-map.json", nil)
	if err != nil {
		return "", nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()
	var pages []mapPage
	if err := json.NewDecoder(res.Body).Decode(&pages); err != nil {
		return base, nil, nil
	}
	return base, pages, nil
}

// Language is the Studio's language: "pt" or "en".
func (s *Studio) Language() string { return s.lang }
