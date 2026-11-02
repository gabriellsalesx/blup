package scaffold

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

// Hidden notes and their translations are left out of the published
// site's mounts (never of the preview's), and their links are quiet.
func TestUnpublishedNotes(t *testing.T) {
	vault := t.TempDir()
	notes := map[string]string{
		"Site/Posts/Aberta.md":         "---\ntitle: a\n---\n",
		"Site/Posts/Fechada [1].md":    "---\npublish: false\n---\n",
		"Site/Posts/Chave.md":          "---\nprivate: yes\ntranslationKey: k1\n---\n",
		"Site/Posts/Rascunho.md":       "---\ndraft: true\n---\n",
		"Site/Posts/en/Fechada [1].md": "---\ntitle: closed\n---\n",
		"Site/Posts/en/Key.md":         "---\ntranslationKey: k1\n---\n",
		"Site/Posts/en/Open.md":        "---\ntitle: open\n---\n",
		"Site/Posts/en/Proprio.md":     "---\npublish: \"no\"\n---\n",
		"Site/.obsidian/Ignorada.md":   "---\npublish: false\n---\n",
		"Site/Posts/Quebrada.md":       "---\n: [\n---\n",
	}
	for rel, body := range notes {
		p := filepath.Join(vault, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	s := config.NewSite(t.TempDir())
	s.Title, s.Lang, s.Model, s.Vault, s.Folder = "x", "pt", "mochi", vault, "Site"
	s.Translations = []string{"en"}

	hidden, quiet := unpublished(s)
	want := []string{"Posts/Chave.md", "Posts/Fechada [1].md", "Posts/en/Fechada [1].md", "Posts/en/Key.md", "Posts/en/Proprio.md"}
	if got := sortedKeys(hidden); !slices.Equal(got, want) {
		t.Fatalf("hidden = %v, want %v", got, want)
	}
	if !slices.Contains(quiet, "Posts/Rascunho.md") || !slices.Contains(quiet, "Posts/Chave.md") || slices.Contains(quiet, "Posts/Aberta.md") {
		t.Fatalf("quiet = %v", quiet)
	}

	b, err := HugoConfig(s, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, pat := range []string{`'! Posts/Fechada ?1?.md'`, `'! Posts/Chave.md'`, `'! Fechada ?1?.md'`, `'! Key.md'`, `'! Proprio.md'`} {
		if !strings.Contains(string(b), pat) {
			t.Errorf("hugo.toml lacks %s:\n%s", pat, b)
		}
	}
	if strings.Contains(string(b), "Aberta.md") || strings.Contains(string(b), "Open.md") {
		t.Error("a public note is excluded")
	}
	prev, _ := HugoConfig(s, true)
	if strings.Contains(string(prev), "Fechada") {
		t.Error("the preview shows hidden notes, as it shows drafts")
	}
}

func TestLiteralGlob(t *testing.T) {
	for in, want := range map[string]string{
		"Posts/Nota.md":     "Posts/Nota.md",
		"a*b?c[d]{e,f}!.md": "a?b?c?d??e?f??.md",
		" lead/trail .md":   "?lead/trail .md",
		"Ação/Café ☕.md":    "Ação/Café ☕.md",
	} {
		if got := literalGlob(in); got != want {
			t.Errorf("literalGlob(%q) = %q, want %q", in, got, want)
		}
	}
}
