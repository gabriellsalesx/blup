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
	a.Feedback = config.Feedback{Enabled: true, Owner: "ana", Repo: "votos"}
	a.Translations = []string{"es", "en", "pt", "xx"}
	if err := writeShared(a); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(sharedPath(a))
	if err != nil {
		t.Fatal(err)
	}
	for _, local := range []string{"segredo", a.Vault, "meu-site-notas", "votos"} {
		if strings.Contains(string(b), local) {
			t.Fatalf("%q must stay on this computer:\n%s", local, b)
		}
	}

	// Another computer: its own folders, the same settings.
	other := config.NewSite(t.TempDir())
	other.Vault = a.Vault
	other.Folder = a.Folder
	other.Password = "outra"
	other, err = readShared(other)
	if err != nil {
		t.Fatal(err)
	}
	if other.Title != "Meu site" || other.Model != "nori" || other.Cookies.Umami != "abc" || other.Password != "outra" {
		t.Fatalf("other = %+v", other)
	}
	if other.Deploy.Provider != "" || other.Deploy.GitHub.Repo != "" {
		t.Fatalf("the destination came from another computer: %+v", other.Deploy)
	}
	if strings.Contains(string(b), "meu-site\"") || strings.Contains(string(b), "[deploy") {
		t.Fatalf("the destination must stay on this computer:\n%s", b)
	}
	if strings.Join(other.Translations, ",") != "en,es" {
		t.Fatalf("translations = %v", other.Translations)
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

// A site.toml from another computer (or an older Blup) never changes where
// this computer publishes nor its feedback repository; a file without
// translations keeps this computer's.
func TestSharedSiteIgnoresDeployAndFeedback(t *testing.T) {
	s := config.NewSite(t.TempDir())
	s.Vault, s.Lang = t.TempDir(), "pt"
	s.Deploy = config.Deploy{Provider: "github", GitHub: config.GitHub{Owner: "eu", Repo: "meu"}}
	s.Feedback = config.Feedback{Enabled: true, Owner: "eu", Repo: "votos"}
	s.Translations = []string{"en"}
	old := `version = 1
title = "Site"
lang = "pt"
model = "mochi"

[deploy]
provider = "cloudflare"
[deploy.cloudflare]
account = "atacante"
project = "outro"
[deploy.blup]
id = "x"
slug = "atacante"

[feedback]
enabled = true
owner = "atacante"
repo = "spam"
api = "https://evil.example"
`
	p := sharedPath(s)
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := readShared(s)
	if err != nil {
		t.Fatal(err)
	}
	if s.Deploy.Provider != "github" || s.Deploy.GitHub.Repo != "meu" || s.Deploy.Cloudflare.Account != "" || s.Deploy.Blup.Slug != "" {
		t.Fatalf("deploy = %+v", s.Deploy)
	}
	if s.Feedback.Owner != "eu" || s.Feedback.Repo != "votos" || s.Feedback.API != "" {
		t.Fatalf("feedback = %+v", s.Feedback)
	}
	if strings.Join(s.Translations, ",") != "en" {
		t.Fatalf("an old file must keep the translations: %v", s.Translations)
	}

	// The new format: translations travel (also "none"), unknown ones out.
	os.WriteFile(p, []byte("version = 1\ntitle = 'Site'\nlang = 'pt'\nmodel = 'mochi'\ntranslations = ['es', '../x', 'pt']\n"), 0o644)
	if s, err = readShared(s); err != nil || strings.Join(s.Translations, ",") != "es" {
		t.Fatalf("translations = %v, %v", s.Translations, err)
	}
	os.WriteFile(p, []byte("version = 1\ntitle = 'Site'\nlang = 'pt'\nmodel = 'mochi'\ntranslations = []\n"), 0o644)
	if s, err = readShared(s); err != nil || len(s.Translations) != 0 {
		t.Fatalf("translations = %v, %v", s.Translations, err)
	}
}
