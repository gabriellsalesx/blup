package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Only the two themes are accepted; anything else is a clear error.
func TestLoadSiteModel(t *testing.T) {
	for model, ok := range map[string]bool{"mochi": true, "bento": true, "blog": false, "docs": false, "": false} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, SiteFile), []byte("model = \""+model+"\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		s, err := LoadSite(dir)
		var unknown UnknownModelError
		switch {
		case ok && (err != nil || s.Model != model):
			t.Errorf("model %q: %v", model, err)
		case !ok && !errors.As(err, &unknown):
			t.Errorf("model %q should be rejected, got %v", model, err)
		}
	}
}

func TestSiteVersion(t *testing.T) {
	dir := t.TempDir()
	s := NewSite(dir)
	s.Model = "mochi"
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSite(dir)
	if err != nil || got.Version != SiteVersion {
		t.Fatalf("saved version = %d, %v", got.Version, err)
	}
	os.WriteFile(filepath.Join(dir, SiteFile), []byte("version = 99\nmodel = \"mochi\"\n"), 0o644)
	if _, err := LoadSite(dir); !errors.Is(err, ErrNewerSite) {
		t.Fatalf("want ErrNewerSite, got %v", err)
	}
}

// A blup.toml with the site's password of protected notes is readable only
// by the user.
func TestSitePasswordFileMode(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("no Unix modes on Windows")
	}
	dir := t.TempDir()
	s := NewSite(dir)
	s.Model = "mochi"
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	mode := func() os.FileMode {
		info, err := os.Stat(filepath.Join(dir, SiteFile))
		if err != nil {
			t.Fatal(err)
		}
		return info.Mode().Perm()
	}
	if got := mode(); got != 0o644 {
		t.Fatalf("without a password: %o", got)
	}
	s.Password = "girassol"
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if got := mode(); got != 0o600 {
		t.Fatalf("with a password: %o, want 600", got)
	}
}

func TestBindNotesKey(t *testing.T) {
	for _, tc := range []struct {
		name             string
		in               Credentials
		account          string
		changed          bool
		key, notesAcount string
	}{
		{"same account", Credentials{NotesKey: "k", NotesAccount: "a"}, "a", false, "k", "a"},
		{"other account", Credentials{NotesKey: "k", NotesAccount: "a"}, "b", true, "", ""},
		{"legacy key adopted", Credentials{NotesKey: "k"}, "a", true, "k", "a"},
		{"no key", Credentials{}, "a", false, "", ""},
		{"account without key", Credentials{NotesAccount: "a"}, "a", true, "", ""},
		{"unknown account", Credentials{NotesKey: "k", NotesAccount: "a"}, "", false, "k", "a"},
	} {
		c := tc.in
		if got := c.BindNotesKey(tc.account); got != tc.changed || c.NotesKey != tc.key || c.NotesAccount != tc.notesAcount {
			t.Errorf("%s: changed %v, %+v", tc.name, got, c)
		}
	}
}
