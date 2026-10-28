package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// TestStarter writes every starter site and checks the notes are there, the
// placeholders were filled and the authors folder is marked. With
// BLUP_STARTER_OUT set, the sites are written there to be built and looked at.
func TestStarter(t *testing.T) {
	out := os.Getenv("BLUP_STARTER_OUT")
	for _, lang := range []string{"pt", "en"} {
		for _, model := range []string{"mochi", "bento"} {
			vault := t.TempDir()
			if out != "" {
				vault = filepath.Join(out, lang+"-"+model, "vault")
				os.RemoveAll(vault)
			}
			site := config.NewSite(t.TempDir())
			site.Vault, site.Folder, site.Lang, site.Model = vault, "Site", lang, model
			site.Title, site.Author = "Teste", "Ana Lima"
			if err := writeSamples(site, docsKinds); err != nil {
				t.Fatal(err)
			}
			if err := writeSiteNotes(site); err != nil {
				t.Fatal(err)
			}
			if obsidian.CountNotes(site.ContentDir()) < 5 {
				t.Fatalf("%s/%s: too few notes", lang, model)
			}
			if obsidian.Sections(site.ContentDir())["authors"] == "" {
				t.Fatalf("%s/%s: no authors folder", lang, model)
			}
			filepath.Walk(site.ContentDir(), func(p string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || filepath.Ext(p) != ".md" {
					return err
				}
				b, _ := os.ReadFile(p)
				if strings.Contains(string(b), "{{") {
					t.Errorf("%s: placeholder left", p)
				}
				return nil
			})
		}
	}
}
