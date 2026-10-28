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
