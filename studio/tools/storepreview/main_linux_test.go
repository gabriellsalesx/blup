//go:build linux && !android

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

func TestPrepareContainsOnlyDisposableExamples(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	if err := prepare(root, "notes", "light"); err != nil {
		t.Fatal(err)
	}
	g, err := config.LoadGlobal()
	if err != nil || len(g.Sites) != 2 || g.Lang != "en" {
		t.Fatalf("demo config: %+v, %v", g, err)
	}
	for _, dir := range g.Sites {
		site, err := config.LoadSite(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []string{dir, site.ContentDir()} {
			if !strings.HasPrefix(p, root+string(filepath.Separator)) {
				t.Fatalf("demo escaped temporary root: %s", p)
			}
		}
	}
	dir, _ := config.Dir()
	if _, err := os.Stat(filepath.Join(dir, "credentials.toml")); !os.IsNotExist(err) {
		t.Fatal("demo created a credentials file")
	}
	b, err := os.ReadFile(filepath.Join(dir, "studio-prefs.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`"errorReports": "off"`, `"notifications": false`, `"toggleKey": ""`, `"theme": "light"`} {
		if !strings.Contains(string(b), fragment) {
			t.Fatalf("missing safe preference: %s", fragment)
		}
	}
}

func TestSessionFileDoesNotImportServiceCredentials(t *testing.T) {
	file := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(file, []byte(`{"DISPLAY":":99","BLUP_TOKEN":"fixture","CLOUDFLARE_API_TOKEN":"fixture","BLUP_SENTRY_DSN":"fixture"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	values, err := desktopSession(file)
	if err != nil {
		t.Fatal(err)
	}
	if values["DISPLAY"] != ":99" {
		t.Fatal("desktop reference missing")
	}
	for _, key := range []string{"BLUP_TOKEN", "CLOUDFLARE_API_TOKEN", "BLUP_SENTRY_DSN"} {
		if _, ok := values[key]; ok {
			t.Fatalf("service credential imported: %s", key)
		}
	}
}
