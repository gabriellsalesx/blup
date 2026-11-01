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
	for _, model := range []string{"mochi", "bento"} {
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
			if model == "mochi" {
				for _, list := range []string{"index.html", "index.xml"} {
					if strings.Contains(read(list), "/oculto/") {
						t.Errorf("%s lists the hidden note", list)
					}
				}
			}
			if model == "bento" {
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
	s.Title, s.Lang, s.Model, s.Vault, s.Folder = "Cozinha", "pt", "mochi", vault, "Site"
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

func TestCookieParams(t *testing.T) {
	if p := cookieParams(config.Cookies{}); len(p) != 0 {
		t.Fatalf("nothing set: %v", p)
	}
	// The privacy link only matters with the banner.
	if p := cookieParams(config.Cookies{Privacy: "https://x.dev/p"}); len(p) != 0 {
		t.Fatalf("privacy without banner: %v", p)
	}
	p := cookieParams(config.Cookies{
		Banner: true, Privacy: "https://x.dev/p", GoogleAnalytics: "G-AB12CD", TagManager: "GTM-AB12CD", MetaPixel: "123456",
		Umami: "u-id", Plausible: "x.dev", PlausibleScript: "https://stats.x.dev/js/script.js",
	})
	if p["banner"] != true || p["privacy"] != "https://x.dev/p" || p["google_analytics"] != "G-AB12CD" || p["meta_pixel"] != "123456" || p["google_tag_manager"] != "GTM-AB12CD" {
		t.Fatalf("params = %v", p)
	}
	if u := p["umami"].(map[string]any); u["id"] != "u-id" || u["src"] != config.UmamiCloudScript {
		t.Fatalf("umami = %v", u)
	}
	if pl := p["plausible"].(map[string]any); pl["domain"] != "x.dev" || pl["src"] != "https://stats.x.dev/js/script.js" {
		t.Fatalf("plausible = %v", pl)
	}
}

// TestBuildTranslated: a note's translation in a folder named after the
// language (Receitas Doces/en/Bolo de Cenoura.md) is that page in the
// language's site, linked to the original; the folder itself is not a page.
func TestBuildTranslated(t *testing.T) {
	bin, ok := hugo.Find(context.Background())
	if !ok {
		t.Skip("hugo not installed")
	}
	vault := t.TempDir()
	if err := os.CopyFS(vault, os.DirFS("testdata/vault")); err != nil {
		t.Fatal(err)
	}
	site := filepath.Join(vault, "Site")
	write := func(rel, text string) {
		p := filepath.Join(site, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("Receitas Doces/en/Bolo de Cenoura.md", "---\ntitle: Carrot Cake\ntranslationKey: bolo\n---\nA cake. See [the guide](../01%20-%20Guia/01%20-%20Instalar.md).\n")
	// A translation under its own name, linked by translationKey.
	write("Receitas Doces/Bolo de Cenoura.md", "---\ntranslationKey: bolo\n---\nSem título no frontmatter nem H1.\n\n## Modo de preparo\nMisture tudo.\n")
	write("Receitas Doces/es/Pastel de zanahoria.md", "---\ntitle: Pastel de zanahoria\ntranslationKey: bolo\nai_translated: true\n---\nUn pastel.\n")
	write("Receitas Doces/es/_index.md", "---\ntitle: Recetas dulces\n---\n")
	write("01 - Guia/es/Instalar.md", "---\ntitle: Instalar\n---\nVer [el pastel](../Receitas%20Doces/Bolo%20de%20Cenoura.md).\n")
	write("en/Início.md", "---\ntitle: Welcome to the kitchen\n---\nHello.\n")

	dir := t.TempDir()
	s := config.NewSite(dir)
	s.Title, s.Lang, s.Model, s.Vault, s.Folder = "Cozinha", "pt", "mochi", vault, "Site"
	s.Translations = []string{"en", "es"}
	if err := Write(s); err != nil {
		t.Fatal(err)
	}
	cmd := bin.Command(context.Background(), dir, "--baseURL", "https://example.org/", "--logLevel", "warn")
	cmd.Env = append(os.Environ(), "HUGO_BLUP_MODE=publish")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hugo: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "ERROR") {
		t.Errorf("hugo output:\n%s", out)
	}
	read := func(p string) string {
		b, err := os.ReadFile(filepath.Join(dir, "public", p))
		if err != nil {
			t.Fatalf("missing %s", p)
		}
		return string(b)
	}
	if pt := read("receitas-doces/bolo-de-cenoura/index.html"); !strings.Contains(pt, "/en/receitas-doces/bolo-de-cenoura/") || !strings.Contains(pt, "/es/receitas-doces/pastel-de-zanahoria/") {
		t.Error("the original should link to its translations")
	}
	en := read("en/receitas-doces/bolo-de-cenoura/index.html")
	if !strings.Contains(en, "Carrot Cake") || !strings.Contains(en, `lang="en`) && !strings.Contains(en, "lang=en") {
		t.Error("english page: title or lang wrong")
	}
	// A link to a note without a translation goes to the original.
	if !strings.Contains(en, `href="/01---guia/01---instalar/"`) && !strings.Contains(en, "href=/01---guia/01---instalar/") {
		t.Errorf("english page: link to an untranslated note should reach the original")
	}
	// Made by AI: the page says so and links to the original.
	if p := read("es/receitas-doces/pastel-de-zanahoria/index.html"); !strings.Contains(p, "Traducido con IA") || !strings.Contains(p, "/receitas-doces/bolo-de-cenoura/") || !strings.Contains(p, "Português") {
		t.Error("spanish page: the AI translation mark is missing")
	}
	if p := read("en/receitas-doces/bolo-de-cenoura/index.html"); strings.Contains(p, "Translated with AI") {
		t.Error("a reviewed translation shows no AI mark")
	}
	// The section in Spanish has its translated name.
	if p := read("es/receitas-doces/index.html"); !strings.Contains(p, "Recetas dulces") {
		t.Error("spanish section: translated name missing")
	}
	// A link to the original reaches its translation in the same language.
	if g := read("es/01---guia/instalar/index.html"); !strings.Contains(g, "/es/receitas-doces/pastel-de-zanahoria/") {
		t.Error("spanish page: a link should reach the spanish translation")
	}
	read("en/inicio/index.html")

	// The preview's map: each note's page, translations by their folder.
	if err := os.WriteFile(filepath.Join(dir, "hugo.toml"), mustConfig(t, s, true), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = bin.Command(context.Background(), dir, "--logLevel", "warn").CombinedOutput()
	if err != nil {
		t.Fatalf("hugo preview: %v\n%s", err, out)
	}
	m := read("blup-map.json")
	for _, want := range []string{`"file":"Receitas Doces/en/Bolo de Cenoura.md","url":"/en/receitas-doces/bolo-de-cenoura/"`, `"file":"Receitas Doces/es/Pastel de zanahoria.md","url":"/es/receitas-doces/pastel-de-zanahoria/"`, `"file":"en/Início.md"`} {
		if !strings.Contains(m, want) {
			t.Errorf("blup-map: missing %s in %s", want, m)
		}
	}
	for _, gone := range []string{"receitas-doces/en", "en/receitas-doces/en", "es/es"} {
		if _, err := os.Stat(filepath.Join(dir, "public", gone)); err == nil {
			t.Errorf("%s should not be a page", gone)
		}
	}
}

func mustConfig(t *testing.T, s *config.Site, preview bool) []byte {
	t.Helper()
	b, err := HugoConfig(s, preview)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
