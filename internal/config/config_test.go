package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Sites created before the themes had names still load, as Mochi and Bento.
func TestLoadSiteOldModelNames(t *testing.T) {
	for old, want := range map[string]string{"blog": "mochi", "docs": "bento", "mochi": "mochi", "bento": "bento"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, SiteFile), []byte("model = \""+old+"\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		s, err := LoadSite(dir)
		if err != nil {
			t.Fatal(err)
		}
		if s.Model != want {
			t.Errorf("model %q loaded as %q, want %q", old, s.Model, want)
		}
	}
}
