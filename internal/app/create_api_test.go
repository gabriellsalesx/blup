package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

func TestCreateSite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var g config.Global
	notes, dir := filepath.Join(t.TempDir(), "Notas"), filepath.Join(t.TempDir(), "site")

	if _, err := CreateSite(&g, NewSiteOptions{Vault: notes, Dir: dir, Model: "bento", Title: "Meu Produto"}); err == nil {
		t.Fatal("Bento without sections must be refused")
	}
	site, err := CreateSite(&g, NewSiteOptions{
		Vault: notes, Dir: dir, Model: "bento", Sections: []string{"docs", "roadmap"},
		Title: " Meu Produto ", Description: "Um produto", Author: "Ana", Lang: "pt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if site.Title != "Meu Produto" || site.Vault != notes || site.Folder != "" {
		t.Fatalf("site = %+v", site)
	}
	have := obsidian.Sections(notes)
	if have["docs"] == "" || have["roadmap"] == "" || have["help"] != "" {
		t.Fatalf("sections = %v", have)
	}
	if _, err := os.Stat(filepath.Join(dir, config.SiteFile)); err != nil {
		t.Fatal("blup.toml missing")
	}
	if g2, _ := config.LoadGlobal(); len(g2.Sites) != 1 || g2.Sites[0] != dir {
		t.Fatalf("sites = %v", g2.Sites)
	}
	home, _ := obsidian.ReadNote(filepath.Join(notes, "_index.md"))
	if home.String("site_name") != "Meu Produto" {
		t.Fatalf("home note = %v", home.Props)
	}
	// The same place again: refused.
	if _, err := CreateSite(&g, NewSiteOptions{Vault: notes, Dir: dir, Title: "X"}); err == nil {
		t.Fatal("a folder with a site must be refused")
	}
	if CheckSiteDir(dir) == "" || CheckSiteDir(filepath.Join(t.TempDir(), "new")) != "" {
		t.Fatal("CheckSiteDir")
	}
}

// TestDefaultNotesDirStaysInBlup: a title from another computer can't
// place the notes folder outside Documents/Blup.
func TestDefaultNotesDirStaysInBlup(t *testing.T) {
	base := filepath.Dir(DefaultNotesDir("x"))
	for _, title := range []string{"../../.config/autostart", `..\..\x`, "..", ".", "  ", "/etc", "a/../../b"} {
		got := DefaultNotesDir(title)
		if filepath.Dir(got) != base {
			t.Errorf("%q -> %q, outside %q", title, got, base)
		}
	}
	if got := filepath.Base(DefaultNotesDir("Meu Diário")); got != "Meu Diário" {
		t.Errorf("plain title changed: %q", got)
	}
}
