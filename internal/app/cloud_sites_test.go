package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

func TestSharedSiteTravelsWithoutLocalThings(t *testing.T) {
	a := config.NewSite(t.TempDir())
	a.Title, a.Lang, a.Model, a.Author = "Meu site", "pt", "nori", "Ana"
	a.Vault, a.Folder, a.Password = t.TempDir(), "Site", "segredo"
	a.Deploy.Provider, a.Deploy.GitHub.Repo = "github", "meu-site"
	a.Sync = config.NoteSync{Owner: "ana", Repo: "meu-site-notas"}
	a.Cookies.Umami = "abc"
	if err := writeShared(a); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(sharedPath(a))
	if err != nil {
		t.Fatal(err)
	}
	for _, local := range []string{"segredo", a.Vault, "meu-site-notas"} {
		if strings.Contains(string(b), local) {
			t.Fatalf("%q must stay on this computer:\n%s", local, b)
		}
	}

	// Another computer: its own folders, the same settings.
	other := config.NewSite(t.TempDir())
	other.Vault = a.Vault
	other.Folder = a.Folder
	other.Password = "outra"
	if err := readShared(other); err != nil {
		t.Fatal(err)
	}
	if other.Title != "Meu site" || other.Model != "nori" || other.Deploy.GitHub.Repo != "meu-site" || other.Cookies.Umami != "abc" || other.Password != "outra" {
		t.Fatalf("other = %+v", other)
	}
	if _, err := parseShared([]byte("version = 99\ntitle = 'x'\n")); err != config.ErrNewerSite {
		t.Fatalf("newer: %v", err)
	}
}

func TestFreeDir(t *testing.T) {
	root := t.TempDir()
	site := filepath.Join(root, "meu-site")
	if freeDir(site) != site {
		t.Fatal("a missing folder is free")
	}
	os.MkdirAll(site, 0o755)
	if freeDir(site) != site {
		t.Fatal("an empty folder is free")
	}
	os.WriteFile(filepath.Join(site, "x"), nil, 0o644)
	if got := freeDir(site); got != site+"-2" {
		t.Fatalf("got %s", got)
	}
	notes := filepath.Join(root, "Meu site")
	os.MkdirAll(notes, 0o755)
	os.WriteFile(filepath.Join(notes, "a.md"), nil, 0o644)
	if got := freeDir(notes); got != notes+" 2" {
		t.Fatalf("got %s", got)
	}
}

