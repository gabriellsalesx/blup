package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

func TestWriteSiteNotes(t *testing.T) {
	vault := t.TempDir()
	site := config.NewSite(t.TempDir())
	site.Vault, site.Folder, site.Lang, site.Model = vault, "Site", "pt", "mochi"
	site.Title, site.Description, site.Author = "Meu: site", "Sobre tudo", "Ana Lima"
	if err := ensureHome(site); err != nil {
		t.Fatal(err)
	}
	if err := writeSiteNotes(site); err != nil {
		t.Fatal(err)
	}
	home, err := obsidian.ReadNote(filepath.Join(vault, "Site", "_index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if home.String("site_name") != "Meu: site" || home.String("site_description") != "Sobre tudo" || home.String("title") == "" {
		t.Fatalf("home props: %v", home.Props)
	}
	if obsidian.Sections(filepath.Join(vault, "Site"))["authors"] != "Autores" {
		t.Fatal("authors folder not marked")
	}
	if _, err := os.Stat(filepath.Join(vault, "Site", "Autores", "Ana Lima.md")); err != nil {
		t.Fatal(err)
	}
	// Running again finds the same note, whatever the case of the name.
	p, err := authorNotePath(site, "ana lima")
	if err != nil || filepath.Base(p) != "Ana Lima.md" {
		t.Fatalf("got %s %v", p, err)
	}
}
