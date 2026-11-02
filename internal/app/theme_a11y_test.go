package app

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

// What Lighthouse asked of the themes (accessibility and SEO) stays done:
// a decorative logo, named task boxes, a description on tag pages, a whole
// outline on lists and the first card's image loaded at once.
func TestThemeLighthouseFixes(t *testing.T) {
	if _, err := exec.LookPath("hugo"); err != nil {
		t.Skip("hugo not installed")
	}
	t.Setenv("BLUP_NO_SEARCH", "1")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var g config.Global
	site, err := CreateSite(&g, NewSiteOptions{Vault: t.TempDir(), Dir: filepath.Join(t.TempDir(), "site"), Model: "mochi", Title: "Diário", Lang: "pt"})
	if err != nil {
		t.Fatal(err)
	}
	b := buildBento(t, site)
	home := b.read("index.html")
	if !strings.Contains(home, `alt width=32`) && !strings.Contains(home, `alt="" width="32"`) {
		t.Error("the logo's image repeats the site's name")
	}
	if !strings.Contains(b.read("tags/index.html"), "<meta name=description content=") {
		t.Error("the tags page has no description")
	}
	if !strings.Contains(b.read("404.html"), "<meta name=description content=") {
		t.Error("the 404 page has no description")
	}
	list := b.read("escrevendo/index.html")
	if !strings.Contains(list, "class=sr-only>Todos os posts</h2>") {
		t.Error("the list skips from h1 to the cards' h3")
	}
	if strings.Count(list, "fetchpriority=high") != 1 {
		t.Error("the first card's image is not loaded first (or more are)")
	}
	if !strings.Contains(b.read("escrevendo/formatacao-que-funciona/index.html"), "aria-label=Tarefa") {
		t.Error("the task boxes have no name")
	}
}
