package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

func settingsSite(t *testing.T) *config.Site {
	t.Helper()
	site := config.NewSite(t.TempDir())
	site.Vault, site.Folder, site.Lang, site.Model, site.Title = t.TempDir(), "Site", "pt", "mochi", "Velho"
	os.MkdirAll(filepath.Join(site.ContentDir(), "Posts"), 0o755)
	os.WriteFile(filepath.Join(site.ContentDir(), "_index.md"), []byte("---\ntitle: Início\n---\nOi\n"), 0o644)
	os.WriteFile(filepath.Join(site.ContentDir(), "Posts", "Um.md"), []byte("# Um\n"), 0o644)
	return site
}

func TestSettingsRoundTrip(t *testing.T) {
	site := settingsSite(t)
	if err := SaveAbout(site, " Meu Blog ", "Sobre coisas", "Ana"); err != nil {
		t.Fatal(err)
	}
	if err := SaveMenu(site, []MenuItem{{"Início", "/"}, {"Posts", "Posts"}, {"", "x"}}, "Assine", "https://x.dev/news"); err != nil {
		t.Fatal(err)
	}
	if err := SaveSocial(site, []string{"https://github.com/ana", "ana@x.dev", " "}); err != nil {
		t.Fatal(err)
	}
	if err := SaveSocial(site, []string{"not a link"}); err == nil {
		t.Fatal("a bad link must be refused")
	}
	if err := SetPassword(site, "segredo"); err != nil {
		t.Fatal(err)
	}
	img := filepath.Join(t.TempDir(), "logo.png")
	os.WriteFile(img, []byte("png"), 0o644)
	file, err := SetImage(site, "logo", img)
	if err != nil || !strings.HasSuffix(file, filepath.Join("Anexos", "logo.png")) {
		t.Fatalf("logo = %q, %v", file, err)
	}

	s, err := LoadSettings(site)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Meu Blog" || s.Description != "Sobre coisas" || s.Author != "Ana" || !s.HasPassword || s.Logo != file {
		t.Fatalf("settings = %+v", s)
	}
	if len(s.Menu) != 2 || s.Menu[1] != (MenuItem{"Posts", "Posts"}) || s.ButtonText != "Assine" {
		t.Fatalf("menu = %+v, button %q", s.Menu, s.ButtonText)
	}
	if strings.Join(s.Social, ",") != "https://github.com/ana,mailto:ana@x.dev" {
		t.Fatalf("social = %v", s.Social)
	}
	if len(s.MenuOptions) != 3 || s.MenuOptions[1].Target != "Posts" {
		t.Fatalf("menu options = %+v", s.MenuOptions)
	}
	home, _ := obsidian.ReadNote(homeNotePath(site))
	if home.String("title") != "Início" || strings.TrimSpace(home.Body) != "Oi" {
		t.Fatal("other properties and the text of the home note must stay")
	}
	if _, err := os.Stat(filepath.Join(site.Dir(), config.SiteFile)); err != nil {
		t.Fatal("blup.toml must be saved")
	}
}

func TestSaveCookies(t *testing.T) {
	site := settingsSite(t)
	if _, err := SaveCookies(site, config.Cookies{GoogleAnalytics: "nope"}, false, ""); err == nil {
		t.Fatal("a bad ID must be refused")
	}
	note, err := SaveCookies(site, config.Cookies{Banner: true, GoogleAnalytics: " G-AB12CD ", Privacy: "https://x.dev/p"}, true, "ana@x.dev")
	if err != nil {
		t.Fatal(err)
	}
	if note != "Política de privacidade.md" || site.Cookies.GoogleAnalytics != "G-AB12CD" || site.Cookies.Privacy != "" {
		t.Fatalf("note %q, cookies %+v", note, site.Cookies)
	}
	if s, _ := LoadSettings(site); s.PrivacyNote != note {
		t.Fatalf("privacy note = %q", s.PrivacyNote)
	}
	if err := SetFeedback(site, true); err == nil {
		t.Fatal("comments can't be turned on without their repository")
	}
}
