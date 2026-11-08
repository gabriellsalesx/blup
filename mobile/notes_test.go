package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenable(t *testing.T) {
	for raw, want := range map[string]bool{
		"https://getblup.io":             true,
		"HTTP://example.com/a":           true,
		"mailto:alguem@example.com":      true,
		"tel:+5585999999999":             true,
		"intent://x#Intent;scheme=a;end": false,
		"file:///data/data/x":            false,
		"content://io.getblup.notes/a":   false,
		"javascript:alert(1)":            false,
		"Notas/ideia.md":                 false,
		"":                               false,
	} {
		if got := openable(raw); got != want {
			t.Errorf("openable(%q) = %v, want %v", raw, got, want)
		}
	}
}

// The search finds a note by its name or its text, in any case.
func TestSearch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, err := vaultDir()
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"Mercado.md": "Comprar Café", "Ideias/Viagem.md": "Praia"} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s := &NotesService{}
	for term, want := range map[string]string{"café": "Mercado.md", "VIAGEM": filepath.Join("Ideias", "Viagem.md")} {
		got, err := s.Search(term)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != want {
			t.Errorf("Search(%q) = %v, want [%s]", term, got, want)
		}
	}
	if got, _ := s.Search("  "); len(got) != 0 {
		t.Errorf("an empty search found %v", got)
	}
	text, err := s.Read("Mercado.md")
	if err != nil || text != "Comprar Café" {
		t.Errorf("Read = %q, %v", text, err)
	}
	if _, err := s.Read("../fora.md"); err == nil {
		t.Error("Read went outside the vault")
	}
}

// The list reads again what changed since the last time, and lets go of
// what was deleted; the backlinks are always the current ones.
func TestListCache(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, err := vaultDir()
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("A.md", "primeira versão")
	write("B.md", "[[A]]")
	s := &NotesService{}
	find := func(title string) *Note {
		list, err := s.List()
		if err != nil {
			t.Fatal(err)
		}
		for i := range list {
			if list[i].Title == title {
				return &list[i]
			}
		}
		return nil
	}
	if a := find("A"); a == nil || a.Preview != "primeira versão" || len(a.Backlinks) != 1 {
		t.Fatalf("first list: %+v", a)
	}
	write("A.md", "segunda versão, maior")
	if err := os.Remove(filepath.Join(dir, "B.md")); err != nil {
		t.Fatal(err)
	}
	if a := find("A"); a == nil || a.Preview != "segunda versão, maior" || len(a.Backlinks) != 0 {
		t.Fatalf("after the change: %+v", a)
	}
	if find("B") != nil {
		t.Fatal("a deleted note is still listed")
	}
}

func TestClip(t *testing.T) {
	for _, c := range []struct {
		in   string
		n    int
		want string
	}{{"ação", 2, "aç"}, {"abc", 5, "abc"}, {"", 3, ""}, {"abc", 0, ""}} {
		if got := clip(c.in, c.n); got != c.want {
			t.Errorf("clip(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}
