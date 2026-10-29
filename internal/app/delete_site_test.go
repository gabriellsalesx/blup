package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

func TestDeleteSite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	base := t.TempDir()
	notes := filepath.Join(base, "vault", "Site")
	os.MkdirAll(notes, 0o755)
	os.WriteFile(filepath.Join(notes, "Post.md"), []byte("x"), 0o644)
	siteDir := filepath.Join(base, "site")
	os.MkdirAll(siteDir, 0o755)
	os.WriteFile(filepath.Join(siteDir, "blup.toml"), []byte("version = 1\ntitle = 'X'\nmodel = 'mochi'\nvault = '"+filepath.Join(base, "vault")+"'\nfolder = 'Site'\n"), 0o644)
	if err := RememberSite(siteDir); err != nil {
		t.Fatal(err)
	}
	parts, err := PartsOf(siteDir)
	if err != nil || parts.NotesOwned || parts.NotesDir != notes {
		t.Fatalf("parts: %+v %v", parts, err)
	}
	problems, err := DeleteSite(context.Background(), siteDir, DeleteChoice{Site: true, Notes: true})
	if err != nil || len(problems) != 0 {
		t.Fatalf("delete: %v %v", problems, err)
	}
	for _, p := range []string{siteDir, notes} {
		if _, err := os.Stat(p); err == nil {
			t.Fatalf("%s still there", p)
		}
	}
	if _, err := os.Stat(filepath.Join(data, "Trash", "files", "Site", "Post.md")); err != nil {
		t.Fatal("notes not in the trash")
	}
	if _, err := os.Stat(filepath.Join(base, "vault")); err != nil {
		t.Fatal("the vault itself was deleted")
	}
	if g, _ := config.LoadGlobal(); len(g.Sites) != 0 {
		t.Fatalf("still listed: %v", g.Sites)
	}
}
