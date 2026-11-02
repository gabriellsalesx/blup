package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/scaffold"
)

func TestCreateVault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	var g config.Global

	// A new vault: a folder in Documents with a welcome note, listed.
	v, err := CreateVault(&g, "Minhas notas", "")
	if err != nil {
		t.Fatal(err)
	}
	if !v.IsNotes() || v.ContentDir() == "" {
		t.Fatalf("vault = %+v", v)
	}
	// The welcome note, and the guides for AI agents.
	notes, _ := os.ReadDir(v.ContentDir())
	if len(notes) != 1+len(scaffold.AgentGuides) {
		t.Fatalf("welcome note and guides: %d files", len(notes))
	}
	if !slices.Contains(g.Sites, v.Dir()) {
		t.Fatal("not in the list")
	}
	again, err := config.LoadSite(v.Dir())
	if err != nil || !again.IsNotes() || again.Title != "Minhas notas" {
		t.Fatalf("reloaded = %+v, %v", again, err)
	}

	// Over a folder that already has notes: nothing is written there but
	// the guides for AI agents.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ideia.md"), []byte("# Ideia\n"), 0o644)
	w, err := CreateVault(&g, "Ideias", dir)
	if err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1+len(scaffold.AgentGuides) || w.ContentDir() != dir {
		t.Fatalf("existing folder changed: %d files, notes in %s", len(entries), w.ContentDir())
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "ideia.md")); string(b) != "# Ideia\n" {
		t.Fatal("the note changed")
	}
	if w.Dir() == v.Dir() {
		t.Fatal("two vaults share a folder")
	}
	if _, err := CreateVault(&g, "  ", ""); err == nil {
		t.Fatal("a vault without a name")
	}
}
