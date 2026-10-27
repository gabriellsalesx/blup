package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/hugo"
)

// TestBuild generates a site from testdata/vault for both models and builds
// it with the installed Hugo, checking the Obsidian-specific rendering.
func TestBuild(t *testing.T) {
	bin, ok := hugo.Find(context.Background())
	if !ok {
		t.Skip("hugo not installed")
	}
	vault, _ := filepath.Abs("testdata/vault")
	for _, model := range []string{"blog", "docs"} {
		t.Run(model, func(t *testing.T) {
			dir := t.TempDir()
			if d := os.Getenv("BLUP_KEEP"); d != "" {
				dir = filepath.Join(d, model)
			}
			s := config.NewSite(dir)
			s.Title, s.Lang, s.Model, s.Vault, s.Folder = "Cozinha", "pt", model, vault, "Site"
			s.Description = "Receitas"
			if err := Write(s); err != nil {
				t.Fatal(err)
			}
			out, err := bin.Command(context.Background(), dir, "--minify", "--baseURL", "https://example.org/", "--logLevel", "warn").CombinedOutput()
			if err != nil {
				t.Fatalf("hugo: %v\n%s", err, out)
			}
			if strings.Contains(string(out), "WARN") || strings.Contains(string(out), "ERROR") {
				t.Errorf("hugo output:\n%s", out)
			}
			read := func(p string) string {
				b, err := os.ReadFile(filepath.Join(dir, "public", p))
				if err != nil {
					t.Fatalf("missing %s", p)
				}
				return string(b)
			}
			home := read("inicio/index.html")
			for _, want := range []string{
				`href=/receitas-doces/bolo-de-cenoura/#modo-de-preparo`,
				`<mark>marcado</mark>`,
				`<math`,
				`data-callout=tip`,
				`<details class=callout data-callout=faq`,
				`.webp`,
				`width=300`,
				`<br>`,
				`<title>Bem-vindo à cozinha · Cozinha</title>`,
				`og:image" content="https://example.org/og/inicio.png"`,
			} {
				if !strings.Contains(home, want) {
					t.Errorf("inicio: missing %q", want)
				}
			}
			if strings.Count(home, "Bem-vindo à cozinha</h1>") != 1 {
				t.Error("inicio: H1 should appear once")
			}
			if bolo := read("receitas-doces/bolo-de-cenoura/index.html"); !strings.Contains(bolo, "<title>Bolo de Cenoura · Cozinha</title>") {
				t.Error("bolo: title should fall back to the file name")
			}
			read("og/inicio.png")
			for _, gone := range []string{"privado", "modelos", "receitas-doces/Bolo", "receitas-doces/bolo"} {
				if _, err := os.Stat(filepath.Join(dir, "public", gone)); err == nil {
					t.Errorf("%s should not be published", gone)
				}
			}
			if model == "docs" {
				g := read("01---guia/index.html")
				if i, j := strings.Index(g, ">Instalar<"), strings.Index(g, ">Configurar<"); i < 0 || j < 0 || i > j {
					t.Error("docs: sidebar order/titles wrong")
				}
			}
		})
	}
}
