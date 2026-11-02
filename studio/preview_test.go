package main

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
)

// sleeper is a process standing in for hugo server.
func sleeper(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skip("no sleep command:", err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	return cmd
}

// stopped tells whether the process ended (it was killed).
func stopped(cmd *exec.Cmd) bool {
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

// Two starts of the preview can overlap (a restart after sections changed
// while the window asks for the preview): the second hugo must be stopped,
// not left running unseen; and a start for a site no longer open is too.
func TestAdoptPreview(t *testing.T) {
	a, b := config.NewSite(t.TempDir()), config.NewSite(t.TempDir())
	s := &Studio{site: a}

	first := sleeper(t)
	if url, err := s.adopt(a, first, &app.PreviewGuard{URL: "http://127.0.0.1:1313/"}, nil); err != nil || url != "http://127.0.0.1:1313/" {
		t.Fatalf("first = %q, %v", url, err)
	}
	second := sleeper(t)
	if url, err := s.adopt(a, second, &app.PreviewGuard{URL: "http://127.0.0.1:1314/"}, nil); err != nil || url != "http://127.0.0.1:1313/" {
		t.Errorf("second = %q, %v", url, err)
	}
	if !stopped(second) {
		t.Error("the second hugo kept running")
	}
	if s.preview != first {
		t.Error("the running preview changed")
	}

	s.StopPreview()
	s.site = b
	late := sleeper(t)
	if _, err := s.adopt(a, late, &app.PreviewGuard{URL: "http://127.0.0.1:1315/"}, nil); err == nil {
		t.Error("a preview of a closed site was kept")
	}
	if !stopped(late) || s.preview != nil {
		t.Error("the preview of a closed site kept running")
	}
}

// Both of hugo server's ends of a rebuild reload the preview: after a note
// changed and after the site's config changed (the Site screen's saves).
func TestRebuiltLine(t *testing.T) {
	for line, want := range map[string]bool{
		"Total in 100 ms":   true,
		"Rebuilt in 198 ms": true,
		"Change of config file detected, rebuilding site (#1).": false,
		"Web Server is available at http://localhost:1313/":     false,
	} {
		if rebuilt(line) != want {
			t.Errorf("rebuilt(%q) = %v, want %v", line, !want, want)
		}
	}
}

// blup-map.json comes from the notes (through the theme): only addresses
// of the preview and notes inside the notes folder are kept.
func TestReadPreviewMap(t *testing.T) {
	root := t.TempDir()
	for _, body := range []string{`null`, `{"File": 1}`, `not json`, `[`} {
		pages, err := readPreviewMap(strings.NewReader(body), root)
		if len(pages) != 0 || (body != "null" && err == nil) {
			t.Errorf("%s: %v %v", body, pages, err)
		}
	}
	body := `[
		{"File": "Posts/a.md", "URL": "/posts/a/"},
		{"File": "Posts/b.md", "URL": "https://evil.com/"},
		{"File": "Posts/c.md", "URL": "javascript:alert(1)"},
		{"File": "Posts/d.md", "URL": "//evil.com/x"},
		{"File": "Posts/e.md", "URL": "/\\evil.com"},
		{"File": "Posts/f.md", "URL": ""},
		{"File": "../segredo.md", "URL": "/f/"},
		{"File": "Posts/../../segredo.md", "URL": "/g/"},
		{"File": "/etc/passwd", "URL": "/h/"},
		{"File": "C:/Windows/win.ini", "URL": "/i/"},
		{"File": "", "URL": "/j/"},
		{"File": "Sobre.md", "URL": "/sobre/?x=1#y"}
	]`
	pages, err := readPreviewMap(strings.NewReader(body), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 || pages[0].File != "Posts/a.md" || pages[1].File != "Sobre.md" {
		t.Errorf("pages = %+v", pages)
	}
	big := strings.NewReader("[" + strings.Repeat(" ", maxPreviewMap) + "]")
	if _, err := readPreviewMap(big, root); err == nil {
		t.Error("a map bigger than the limit was read")
	}
	if p, _ := readPreviewMap(strings.NewReader(`[{"File": "a.md", "URL": "/a/"}]`), ""); len(p) != 0 {
		t.Error("pages without an open site")
	}
}
