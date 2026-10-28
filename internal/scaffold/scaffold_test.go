package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/hugo"
	"github.com/gabriellsalesx/blup/internal/lock"
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
			cmd := bin.Command(context.Background(), dir, "--minify", "--baseURL", "https://example.org/", "--logLevel", "warn")
			cmd.Env = append(os.Environ(), "HUGO_BLUP_MODE=publish")
			out, err := cmd.CombinedOutput()
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
				`<details class=callout data-callout=question`,
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
			// Protected note: encrypted, nothing readable anywhere.
			if n, err := lock.Process(filepath.Join(dir, "public"), ""); err != nil || n != 1 {
				t.Fatalf("lock: %d %v", n, err)
			}
			if _, err := os.Stat(filepath.Join(dir, "public", lock.Manifest)); err == nil {
				t.Error("passwords manifest was published")
			}
			filepath.WalkDir(filepath.Join(dir, "public"), func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				b, _ := os.ReadFile(p)
				for _, secret := range []string{"SEGREDO-SUPREMO", "abacaxi", "blup-begin"} {
					if strings.Contains(string(b), secret) {
						t.Errorf("%s leaks %q", p, secret)
					}
				}
				return nil
			})
			if s := read("receitas-doces/receita-secreta/index.html"); !strings.Contains(s, "data-blup-cipher") {
				t.Error("protected page has no ciphertext")
			}
			// Hidden note: published, out of the lists, not indexed.
			if !strings.Contains(read("oculto/index.html"), `content="noindex"`) && !strings.Contains(read("oculto/index.html"), "content=noindex") {
				t.Error("hidden page should be noindex")
			}
			if model == "blog" {
				for _, list := range []string{"index.html", "index.xml"} {
					if strings.Contains(read(list), "/oculto/") {
						t.Errorf("%s lists the hidden note", list)
					}
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

// TestBuildWithoutCLI: plain "hugo" (no HUGO_BLUP_MODE) must never publish a
// protected note in clear text.
func TestBuildWithoutCLI(t *testing.T) {
	bin, ok := hugo.Find(context.Background())
	if !ok {
		t.Skip("hugo not installed")
	}
	vault, _ := filepath.Abs("testdata/vault")
	dir := t.TempDir()
	s := config.NewSite(dir)
	s.Title, s.Lang, s.Model, s.Vault, s.Folder = "Cozinha", "pt", "blog", vault, "Site"
	if err := Write(s); err != nil {
		t.Fatal(err)
	}
	cmd := bin.Command(context.Background(), dir, "--logLevel", "error")
	cmd.Env = append(os.Environ(), "HUGO_BLUP_MODE=")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hugo: %v\n%s", err, out)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "public", "receitas-doces", "receita-secreta", "index.html"))
	if strings.Contains(string(b), "SEGREDO-SUPREMO") || len(b) == 0 {
		t.Fatal("protected note leaked (or missing) without the CLI")
	}
	if m, _ := os.ReadFile(filepath.Join(dir, "public", lock.Manifest)); strings.Contains(string(m), "abacaxi") {
		t.Fatal("manifest leaked the password without the CLI")
	}
}
