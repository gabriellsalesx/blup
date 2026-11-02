package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

// bentoSite makes a Bento site with sections (en) and builds nothing yet.
func bentoSite(t *testing.T, sections ...string) *config.Site {
	t.Helper()
	t.Setenv("BLUP_NO_SEARCH", "1")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	var g config.Global
	site, err := CreateSite(&g, NewSiteOptions{Vault: t.TempDir(), Dir: filepath.Join(t.TempDir(), "site"), Model: "bento", Title: "Produto", Lang: "en", Sections: sections})
	if err != nil {
		t.Fatal(err)
	}
	return site
}

type built struct {
	t   *testing.T
	dir string
}

func buildBento(t *testing.T, site *config.Site) built {
	t.Helper()
	out, err := build(context.Background(), site, "https://example.org/")
	if err != nil {
		t.Fatal(err)
	}
	return built{t, out}
}

func (b built) read(p string) string {
	b.t.Helper()
	data, err := os.ReadFile(filepath.Join(b.dir, filepath.FromSlash(p)))
	if err != nil {
		b.t.Fatalf("missing %s", p)
	}
	return string(data)
}

func (b built) has(p string) bool {
	_, err := os.Stat(filepath.Join(b.dir, filepath.FromSlash(p)))
	return err == nil
}

// The home is the section: same page, one address. The section's own
// page names the home as canonical, is out of the sitemap, and every
// link to it leads home; the pages inside keep their addresses.
func (b built) homeIs(section, marker string) {
	b.t.Helper()
	home := b.read("index.html")
	if !strings.Contains(home, marker) || strings.Contains(home, "section-cards") {
		b.t.Errorf("the home doesn't show %s", section)
	}
	if strings.Contains(home, "http-equiv=refresh") || b.has("_redirects") {
		b.t.Error("a redirect was made")
	}
	sec := b.read(section + "/index.html")
	if !strings.Contains(sec, "rel=canonical href=https://example.org/>") {
		b.t.Errorf("/%s/ is not canonical to the home", section)
	}
	// A translated site has a sitemap per language (sitemap.xml is
	// their index).
	sitemap := "sitemap.xml"
	if b.has("en/sitemap.xml") {
		sitemap = "en/sitemap.xml"
	}
	if strings.Contains(b.read(sitemap), "<loc>https://example.org/"+section+"/</loc>") {
		b.t.Errorf("/%s/ is in the sitemap", section)
	}
	if !strings.Contains(b.read(sitemap), "<loc>https://example.org/</loc>") {
		b.t.Error("the home is not in the sitemap")
	}
	filepath.WalkDir(b.dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".html" {
			return err
		}
		data, _ := os.ReadFile(p)
		if strings.Contains(string(data), "href=/"+section+"/>") || strings.Contains(string(data), `href="/`+section+`/"`) {
			b.t.Errorf("%s links to /%s/", p, section)
		}
		return nil
	})
}

func (b built) homeHasCards() {
	b.t.Helper()
	if !strings.Contains(b.read("index.html"), "section-cards") {
		b.t.Error("the home lost its section cards")
	}
}

func TestBentoHomeSingleSection(t *testing.T) {
	if _, err := exec.LookPath("hugo"); err != nil {
		t.Skip("hugo not installed")
	}
	t.Run("changelog", func(t *testing.T) {
		site := bentoSite(t, "changelog")
		// Enough releases for a second page: it is the home's.
		for i := range 12 {
			os.WriteFile(filepath.Join(site.ContentDir(), "Changelog", fmt.Sprintf("v9.%d.md", i)), fmt.Appendf(nil, "---\ntitle: v9.%d\ndate: 2026-01-%02d\n---\nFixed.\n", i, i+1), 0o644)
		}
		b := buildBento(t, site)
		b.homeIs("changelog", "changelog-head")
		// The logo still leads home, which is now the section.
		if !strings.Contains(b.read("changelog/v9.3/index.html"), "class=brand href=/") {
			t.Error("the logo doesn't lead home")
		}
		if !b.has("page/2/index.html") || b.has("changelog/page/2/index.html") {
			t.Error("the second page is not /page/2/")
		}
		if !b.has("changelog/v9.3/index.html") && !b.has("changelog/v9-3/index.html") {
			t.Error("a release lost its address")
		}
	})
	t.Run("help", func(t *testing.T) {
		b := buildBento(t, bentoSite(t, "help"))
		b.homeIs("help", `class=help-head`)
	})
}

func TestBentoHomeChosen(t *testing.T) {
	if _, err := exec.LookPath("hugo"); err != nil {
		t.Skip("hugo not installed")
	}
	site := bentoSite(t, "docs", "roadmap", "changelog")
	buildBento(t, site).homeHasCards()

	if err := SaveHome(site, "Nope"); err == nil {
		t.Fatal("a folder that isn't a section was chosen")
	}
	if err := SaveHome(site, "Roadmap"); err != nil {
		t.Fatal(err)
	}
	s, _ := LoadSettings(site)
	if s.Home != "Roadmap" {
		t.Fatalf("settings home = %q", s.Home)
	}
	b := buildBento(t, site)
	b.homeIs("roadmap", "roadmap-head")
	// The menu leads home for the roadmap, and to the others as before.
	if !strings.Contains(b.read("docs/index.html"), "href=/changelog/") {
		t.Error("the other sections lost their links")
	}

	// The chosen section is removed: the home with the sections again.
	os.RemoveAll(filepath.Join(site.ContentDir(), "Roadmap"))
	if s, _ := LoadSettings(site); s.Home != "" {
		t.Fatalf("a removed section is still the home: %q", s.Home)
	}
	buildBento(t, site).homeHasCards()
}

// Each language's home shows its own translation of the section.
func TestBentoHomeTranslated(t *testing.T) {
	if _, err := exec.LookPath("hugo"); err != nil {
		t.Skip("hugo not installed")
	}
	site := bentoSite(t, "changelog")
	site.Translations = []string{"pt"}
	pt := filepath.Join(site.ContentDir(), "Changelog", "pt")
	os.MkdirAll(pt, 0o755)
	os.WriteFile(filepath.Join(pt, "_index.md"), []byte("---\ntitle: Novidades\nblup: changelog\n---\n"), 0o644)
	os.WriteFile(filepath.Join(pt, "v2.2.0.md"), []byte("---\ntitle: v2.2.0\ndate: 2026-01-01\n---\nCorrigido.\n"), 0o644)
	if err := site.Save(); err != nil {
		t.Fatal(err)
	}
	b := buildBento(t, site)
	b.homeIs("changelog", "changelog-head")
	if !strings.Contains(b.read("pt/index.html"), "changelog-head") {
		t.Error("the Portuguese home doesn't show its changelog")
	}
	ptSec := b.read("pt/changelog/index.html")
	if !strings.Contains(ptSec, "rel=canonical href=https://example.org/pt/>") {
		t.Errorf("/pt/changelog/ is not canonical to /pt/")
	}
	if strings.Contains(b.read("pt/sitemap.xml"), "<loc>https://example.org/pt/changelog/</loc>") {
		t.Error("/pt/changelog/ is in the sitemap")
	}
	// Its other languages are the homes too.
	if !strings.Contains(ptSec, "hreflang=en-US href=https://example.org/>") {
		t.Errorf("hreflang of /pt/changelog/ doesn't name the English home:\n%.2000s", ptSec)
	}
}
