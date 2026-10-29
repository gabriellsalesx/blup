package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindSite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	base := t.TempDir()
	notes := filepath.Join(base, "vault", "Site")
	os.MkdirAll(filepath.Join(notes, "Posts"), 0o755)
	siteDir := filepath.Join(base, "site")
	os.MkdirAll(siteDir, 0o755)
	os.WriteFile(filepath.Join(siteDir, "blup.toml"), []byte("version = 1\ntitle = 'X'\nmodel = 'mochi'\nvault = '"+filepath.Join(base, "vault")+"'\nfolder = 'Site'\n"), 0o644)

	for name, dir := range map[string]string{"site folder": siteDir, "notes": notes, "inside notes": filepath.Join(notes, "Posts"), "vault": filepath.Join(base, "vault"), "parent": base} {
		if got := FindSite(dir); got != siteDir {
			t.Errorf("%s: FindSite = %q, want %q", name, got, siteDir)
		}
	}
	if got := FindSite(t.TempDir()); got != "" {
		t.Errorf("empty folder found %q", got)
	}
}
