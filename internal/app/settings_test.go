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

func TestRemoveSection(t *testing.T) {
	site := settingsSite(t)
	site.Model = "bento"
	if err := AddSection(site, "roadmap"); err != nil {
		t.Fatal(err)
	}
	folder := existingSections(site)["roadmap"]
	if folder == "" {
		t.Fatal("roadmap not added")
	}
	SaveMenu(site, []MenuItem{{"Início", "/"}, {"Roadmap", folder}}, "", "")

	trashed, err := RemoveSection(site, "roadmap")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := existingSections(site)["roadmap"]; ok {
		t.Fatal("the section must be gone from the notes")
	}
	if _, err := os.Stat(filepath.Join(trashed, "_index.md")); err != nil {
		t.Fatal("the section must be in the trash:", err)
	}
	if s, _ := LoadSettings(site); len(s.Menu) != 1 {
		t.Fatalf("menu = %+v", s.Menu)
	}
	if err := RestoreSection(site, trashed); err != nil {
		t.Fatal(err)
	}
	if existingSections(site)["roadmap"] != folder {
		t.Fatal("the section must be back")
	}
	if err := RestoreSection(site, "/etc"); err == nil {
		t.Fatal("only the Blup trash can be restored")
	}
}

func TestTrash(t *testing.T) {
	site := settingsSite(t)
	site.Model = "bento"
	AddSection(site, "help")
	trashed, err := RemoveSection(site, "help")
	if err != nil {
		t.Fatal(err)
	}
	items, err := ListTrash(site)
	if err != nil || len(items) != 1 || items[0].Path != trashed || items[0].Notes == 0 {
		t.Fatalf("trash = %+v, %v", items, err)
	}
	// Back with the same name taken: refused, the trash keeps it.
	AddSection(site, "help")
	if err := RestoreSection(site, trashed); err == nil {
		t.Fatal("restoring over an existing folder must be refused")
	}
	if err := DeleteFromTrash(site, site.ContentDir()); err == nil {
		t.Fatal("only the trash can be deleted from")
	}
	if err := DeleteFromTrash(site, trashed); err != nil {
		t.Fatal(err)
	}
	if items, _ := ListTrash(site); len(items) != 0 {
		t.Fatalf("trash = %+v", items)
	}
	RemoveSection(site, "help")
	if err := EmptyTrash(site); err != nil {
		t.Fatal(err)
	}
	if items, _ := ListTrash(site); len(items) != 0 {
		t.Fatal("trash must be empty")
	}
}

// The site's name next to the logo: shown by default, hidden with
// header_name: false, and back to the default (no property) when shown.
func TestShowName(t *testing.T) {
	site := settingsSite(t)
	if s, _ := LoadSettings(site); !s.ShowName {
		t.Fatal("the name should show by default")
	}
	if err := SetShowName(site, false); err != nil {
		t.Fatal(err)
	}
	if s, _ := LoadSettings(site); s.ShowName {
		t.Fatal("hidden, it still shows")
	}
	if err := SetShowName(site, true); err != nil {
		t.Fatal(err)
	}
	n, _ := obsidian.ReadNote(homeNotePath(site))
	if _, ok := n.Props["header_name"]; ok {
		t.Fatalf("shown again should drop the property: %v", n.Props)
	}
}
