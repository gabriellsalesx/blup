package app

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

// Each theme's vault comes with the guides for AI agents, in English, and
// a real build of the site has no trace of them: no page, no feed, no
// sitemap entry, no search entry.
func TestAgentGuidesPerTheme(t *testing.T) {
	_, hasHugo := exec.LookPath("hugo")
	t.Setenv("BLUP_NO_SEARCH", "1")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	for _, model := range []string{"mochi", "nori", "bento"} {
		t.Run(model, func(t *testing.T) {
			var g config.Global
			o := NewSiteOptions{Vault: t.TempDir(), Dir: filepath.Join(t.TempDir(), "site"), Model: model, Title: "Meu Site", Lang: "pt"}
			if model == "bento" {
				o.Sections = []string{"docs", "roadmap", "changelog", "help", "blog"}
			}
			site, err := CreateSite(&g, o)
			if err != nil {
				t.Fatal(err)
			}
			read := func(name string) string {
				b, err := os.ReadFile(filepath.Join(site.ContentDir(), name))
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}
			guide := read("INSTRUCTIONS.md")
			if guide != instructions(model, "pt") || !strings.Contains(guide, "Portuguese") || strings.Contains(guide, "{{LANGUAGE}}") {
				t.Errorf("INSTRUCTIONS.md is not %s's guide:\n%.300s", model, guide)
			}
			if strings.ContainsAny(guide, "—–") {
				t.Error("a dash in the guide")
			}
			for _, name := range []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"} {
				if s := read(name); !strings.Contains(s, "Read INSTRUCTIONS.md before doing anything in this vault.") || len(s) > 300 {
					t.Errorf("%s: %q", name, s)
				}
			}
			if !strings.Contains(read("CLAUDE.md"), "@INSTRUCTIONS.md") {
				t.Error("CLAUDE.md does not import the guide")
			}

			if hasHugo != nil {
				t.Skip("hugo not installed")
			}
			out, err := build(context.Background(), site, "https://example.org/")
			if err != nil {
				t.Fatal(err)
			}
			for _, dir := range []string{"agents", "claude", "gemini", "instructions"} {
				if _, err := os.Stat(filepath.Join(out, dir)); err == nil {
					t.Errorf("%s became a page", dir)
				}
			}
			filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				b, _ := os.ReadFile(p)
				s := strings.ToLower(string(b))
				if strings.Contains(s, "instructions for ai agents") || strings.Contains(s, "instructions.md") || strings.Contains(s, "/instructions/") {
					t.Errorf("%s mentions the guides", p)
				}
				return nil
			})
			for _, f := range []string{"index.xml", "sitemap.xml"} {
				if _, err := os.Stat(filepath.Join(out, f)); err != nil {
					t.Errorf("%s missing: the check above saw nothing", f)
				}
			}
		})
	}
}

// A vault of notes gets the notes guide; one made by an older Blup gets
// the new guides where the person changed nothing, and keeps what they
// changed.
func TestAgentGuidesUpgrade(t *testing.T) {
	site := settingsSite(t)
	site.Kind = config.KindNotes
	root := site.ContentDir()
	os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(legacyAgentsEN), 0o644)
	os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("@AGENTS.md\n"), 0o644)
	os.WriteFile(filepath.Join(root, "GEMINI.md"), []byte("# Mine\nAlways answer in haiku.\n"), 0o644)
	if err := WriteAgentGuides(site); err != nil {
		t.Fatal(err)
	}
	read := func(name string) string {
		b, _ := os.ReadFile(filepath.Join(root, name))
		return string(b)
	}
	if read("AGENTS.md") != pointer("AGENTS.md") || read("CLAUDE.md") != pointer("CLAUDE.md") {
		t.Error("the guides Blup wrote before were not brought up to date")
	}
	if read("GEMINI.md") != "# Mine\nAlways answer in haiku.\n" {
		t.Error("the person's GEMINI.md was rewritten")
	}
	if read("INSTRUCTIONS.md") != instructions("notes", "pt") {
		t.Error("a vault of notes did not get the notes guide")
	}
	// Changed by the person: kept, even when Blup's text changes.
	os.WriteFile(filepath.Join(root, "INSTRUCTIONS.md"), []byte("my own rules\n"), 0o644)
	site.Kind, site.Model = "", "bento"
	if err := WriteAgentGuides(site); err != nil {
		t.Fatal(err)
	}
	if read("INSTRUCTIONS.md") != "my own rules\n" {
		t.Error("the person's INSTRUCTIONS.md was rewritten")
	}
	// Still as Blup wrote it: follows (here, the theme changed).
	os.WriteFile(filepath.Join(root, "INSTRUCTIONS.md"), []byte(instructions("notes", "pt")), 0o644)
	WriteAgentGuides(site)
	if read("INSTRUCTIONS.md") != instructions("bento", "pt") {
		t.Error("an untouched guide did not follow")
	}
}

// The guides are not the person's notes: out of the search and the map.
func TestAgentGuidesNotNotes(t *testing.T) {
	site := settingsSite(t)
	if err := WriteAgentGuides(site); err != nil {
		t.Fatal(err)
	}
	hits, err := SearchNotes(site, "INSTRUCTIONS")
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hits {
		if isAgentGuide(h.Path) {
			t.Errorf("search found %s", h.Path)
		}
	}
	g, _, err := noteGraph(site)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range g.Nodes {
		if isAgentGuide(n.Path) {
			t.Errorf("map shows %s", n.Path)
		}
	}
}
