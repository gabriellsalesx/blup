package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

func TestSaveLogoLink(t *testing.T) {
	site := settingsSite(t)
	for _, bad := range []string{"javascript:alert(1)", "empresa.com", "//evil.com"} {
		if err := SaveLogoLink(site, bad); err == nil {
			t.Errorf("%q saved", bad)
		}
	}
	if site.LogoLink != "" {
		t.Fatal("a refused link was kept")
	}
	if err := SaveLogoLink(site, " https://empresa.com.br "); err != nil || site.LogoLink != "https://empresa.com.br" {
		t.Fatalf("%q, %v", site.LogoLink, err)
	}
	if s, _ := LoadSettings(site); s.LogoLink != "https://empresa.com.br" {
		t.Fatalf("settings: %q", s.LogoLink)
	}
	// It travels with the notes; a bad one from elsewhere is dropped.
	sh := shareOf(site)
	other := config.NewSite(t.TempDir())
	sh.apply(other)
	if other.LogoLink != "https://empresa.com.br" {
		t.Fatalf("shared: %q", other.LogoLink)
	}
	sh.LogoLink = "javascript:alert(1)"
	sh.apply(other)
	if other.LogoLink != "" {
		t.Fatalf("a bad shared link was taken: %q", other.LogoLink)
	}
	if err := SaveLogoLink(site, ""); err != nil || site.LogoLink != "" {
		t.Fatal("not back to the home page")
	}
}

// A real build of each theme: the logo leads to the link when set, else
// to the home of the page's language; nothing else changes.
func TestLogoLinkBuild(t *testing.T) {
	if _, err := exec.LookPath("hugo"); err != nil {
		t.Skip("hugo not installed")
	}
	t.Setenv("BLUP_NO_SEARCH", "1")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	brand := func(t *testing.T, html string) string {
		t.Helper()
		i := strings.Index(html, "class=brand href=")
		if i < 0 {
			t.Fatalf("no logo link in:\n%.500s", html)
		}
		rest := html[i+len("class=brand href="):]
		return rest[:strings.IndexAny(rest, " >")]
	}
	for _, model := range []string{"mochi", "nori", "bento"} {
		t.Run(model, func(t *testing.T) {
			var g config.Global
			o := NewSiteOptions{Vault: t.TempDir(), Dir: filepath.Join(t.TempDir(), "site"), Model: model, Title: "Site", Lang: "en"}
			if model == "bento" {
				o.Sections = []string{"docs", "changelog"}
			}
			site, err := CreateSite(&g, o)
			if err != nil {
				t.Fatal(err)
			}
			site.Translations = []string{"pt"}
			site.Save()
			read := func(dir, p string) string {
				b, err := os.ReadFile(filepath.Join(dir, p))
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}
			out := buildBento(t, site).dir
			if got := brand(t, read(out, "index.html")); got != "/" {
				t.Errorf("default logo link %q", got)
			}
			if got := brand(t, read(out, "pt/index.html")); got != "/pt/" {
				t.Errorf("Portuguese default logo link %q", got)
			}

			if err := SaveLogoLink(site, "https://empresa.com.br/"); err != nil {
				t.Fatal(err)
			}
			out = buildBento(t, site).dir
			for _, p := range []string{"index.html", "pt/index.html", "404.html"} {
				if got := brand(t, read(out, p)); got != "https://empresa.com.br/" {
					t.Errorf("%s: logo link %q", p, got)
				}
			}
			if err := SaveLogoLink(site, "/sobre/"); err != nil {
				t.Fatal(err)
			}
			out = buildBento(t, site).dir
			if got := brand(t, read(out, "pt/index.html")); got != "/sobre/" {
				t.Errorf("path logo link %q", got)
			}
		})
	}
}
