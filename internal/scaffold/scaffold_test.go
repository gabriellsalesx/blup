package scaffold

import (
	"bytes"
	"context"
	"encoding/xml"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	write("Privacidade.md", "---\ntitle: Privacidade\nblup: privacy\n---\nPolítica.\n")
	write("es/Privacidad.md", "---\ntitle: Privacidad\nblup: privacy\ntranslationKey: privacidade\n---\nPolítica.\n")
	write("es/_index.md", "---\ntitle: Inicio\nsite_name: Cocina ES\n---\nHola.\n")
	write("Autores/_index.md", "---\ntitle: Autores\nblup: authors\n---\n")
	write("Autores/Ana.md", "---\nrole: Confeiteira\n---\nFaz bolos.\n")
	write("Autores/es/Ana.md", "---\nrole: Pastelera\ntranslationKey: Autores/Ana\n---\nHace pasteles.\n")
	write("Receitas Doces/Torta.md", "---\nauthors: [Ana]\ntranslationKey: torta\n---\nTorta.\n")
	write("Receitas Doces/es/Tarta.md", "---\nauthors: [Ana]\ntranslationKey: torta\n---\nTarta.\n")
	write("01 - Guia/es/Instalar.md", "---\ntitle: Instalar\n---\nVer [el pastel](../Receitas%20Doces/Bolo%20de%20Cenoura.md).\n")
	write("en/Início.md", "---\ntitle: Welcome to the kitchen\n---\nHello.\n")

	dir := t.TempDir()
	s := config.NewSite(dir)
	s.Title, s.Lang, s.Model, s.Vault, s.Folder = "Cozinha", "pt", "mochi", vault, "Site"
	s.Translations = []string{"en", "es"}
	// Feedback on: the original's issue (7) is every version's.
	s.Feedback = config.Feedback{Enabled: true, Owner: "ana", Repo: "cozinha-feedback"}
	if err := Write(s); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "blup_feedback.json"), []byte(`{"issues":{"receitas doces/bolo de cenoura.md":7}}`), 0o644); err != nil {
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
	// The footer: this language's privacy page and site name, not the
	// first language's (cached parts are per language).
	if p := read("es/receitas-doces/pastel-de-zanahoria/index.html"); !strings.Contains(p, "/es/privacidad/") || !strings.Contains(p, "Cocina ES") {
		t.Error("spanish page: footer/settings come from another language")
	}
	if p := read("receitas-doces/bolo-de-cenoura/index.html"); strings.Contains(p, "/es/privacidad/") || strings.Contains(p, "Cocina ES") {
		t.Error("portuguese page: footer/settings come from the spanish site")
	}
	// English has no translated policy: its footer links the original.
	if p := read("en/receitas-doces/bolo-de-cenoura/index.html"); !strings.Contains(p, `href="/privacidade/"`) && !strings.Contains(p, "href=/privacidade/") {
		t.Error("english page: the footer should link the original privacy policy")
	}
	// One conversation for every language: the translation shows the
	// original's issue, and its comments are marked with its language.
	for _, p := range []string{"receitas-doces/bolo-de-cenoura/index.html", "es/receitas-doces/pastel-de-zanahoria/index.html"} {
		if page := read(p); !strings.Contains(page, `data-fb-issue="7"`) && !strings.Contains(page, "data-fb-issue=7") {
			t.Errorf("%s: should show the original's conversation (issue 7)", p)
		}
	}
	if page := read("es/receitas-doces/pastel-de-zanahoria/index.html"); !strings.Contains(page, `data-fb-lang="es"`) && !strings.Contains(page, "data-fb-lang=es") {
		t.Error("spanish page: the feedback script should know the page's language")
	}
	// Authors: each language reads the profile in its language; the
	// profiles are never pages.
	if p := read("es/authors/ana/index.html"); !strings.Contains(p, "Pastelera") || strings.Contains(p, "Confeiteira") {
		t.Error("spanish author page: the profile should be the spanish one")
	}
	if p := read("authors/ana/index.html"); !strings.Contains(p, "Confeiteira") {
		t.Error("portuguese author page: the original profile")
	}
	for _, gone := range []string{"es/autores", "autores"} {
		if _, err := os.Stat(filepath.Join(dir, "public", gone)); err == nil {
			t.Errorf("%s should not be published", gone)
		}
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

// TestChangelogMap: the changelog opens with a year of days, each version
// on its day, colored by what it was, linking to it.
func TestChangelogMap(t *testing.T) {
	bin, ok := hugo.Find(context.Background())
	if !ok {
		t.Skip("hugo not installed")
	}
	vault := t.TempDir()
	write := func(rel, text string) {
		p := filepath.Join(vault, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	day := func(ago int) string { return time.Now().AddDate(0, 0, -ago).Format("2006-01-02") }
	write("_index.md", "---\ntitle: Produto\n---\n")
	write("Changelog/_index.md", "---\ntitle: Changelog\nblup: changelog\n---\n")
	write("Changelog/v1.1.md", "---\ntitle: Versão 1.1\ndate: "+day(3)+"\ntags: [Correção]\n---\nCorrigido.\n")
	write("Changelog/v1.0.md", "---\ntitle: Versão 1.0\ndate: "+day(40)+"\ntags: [Novidade, Melhoria]\n---\nNovo.\n")
	write("Changelog/v0.9.md", "---\ntitle: Versão 0.9\ndate: "+day(40)+"\ntype: security\n---\nSeguro.\n")
	write("Changelog/v0.1.md", "---\ntitle: Antiga\ndate: "+day(800)+"\n---\nVelha.\n")

	dir := t.TempDir()
	s := config.NewSite(dir)
	s.Title, s.Lang, s.Model, s.Vault = "Produto", "pt", "bento", vault
	if err := Write(s); err != nil {
		t.Fatal(err)
	}
	out, err := bin.Command(context.Background(), dir, "--logLevel", "warn").CombinedOutput()
	if err != nil {
		t.Fatalf("hugo: %v\n%s", err, out)
	}
	b, err := os.ReadFile(filepath.Join(dir, "public", "changelog", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	if strings.Count(page, `<a class="cl-day k-fix"`) != 1 || !strings.Contains(page, `href="/changelog/v1.1/" data-tip=`) {
		t.Error("the fix should be one red day linking to its version")
	}
	// Two versions the same day: the most serious color (security), stronger.
	if !strings.Contains(page, `<a class="cl-day k-security many"`) {
		t.Error("two versions on a day: security color, marked many")
	}
	if !strings.Contains(page, "Versão 1.0 · Novidade") || !strings.Contains(page, "Versões nos últimos 12 meses: 3") {
		t.Error("the day's box should list its versions; the total counts the year")
	}
	if strings.Contains(page, "Antiga ·") {
		t.Error("a version older than a year is out of the map")
	}
	if i, j := strings.Index(page, "cl-map"), strings.Index(page, `class="changelog"`); i < 0 || j < i {
		t.Error("the map comes before the list of versions")
	}
}

// The feedback section is the board: tabs by kind, the new-request form and
// the block of votes and comments a request's page reuses. Without
// comments turned on, a note says so.
func TestFeedbackBoard(t *testing.T) {
	bin, ok := hugo.Find(context.Background())
	if !ok {
		t.Skip("hugo not installed")
	}
	for _, on := range []bool{true, false} {
		vault := t.TempDir()
		for rel, text := range map[string]string{
			"_index.md":          "---\ntitle: Produto\n---\n",
			"Feedback/_index.md": "---\ntitle: Feedback\nblup: feedback\ndescription: Peça e vote.\nexamples:\n  - title: Modo escuro\n    kind: bug\n    status: planned\n    votes: 7\n---\n",
		} {
			p := filepath.Join(vault, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		dir := t.TempDir()
		s := config.NewSite(dir)
		s.Title, s.Lang, s.Model, s.Vault = "Produto", "pt", "bento", vault
		if on {
			s.Feedback = config.Feedback{Enabled: true, Owner: "ana", Repo: "produto-feedback"}
		}
		if err := Write(s); err != nil {
			t.Fatal(err)
		}
		out, err := bin.Command(context.Background(), dir, "--logLevel", "warn").CombinedOutput()
		if err != nil {
			t.Fatalf("hugo: %v\n%s", err, out)
		}
		b, err := os.ReadFile(filepath.Join(dir, "public", "feedback", "index.html"))
		if err != nil {
			t.Fatal(err)
		}
		page := string(b)
		if !on {
			// The board still works on its examples (the script loads), and
			// says why it can't take requests.
			if !strings.Contains(page, "data-fb-board") || strings.Contains(page, "data-fbb-issues") || !strings.Contains(page, "js/feedback") || !strings.Contains(page, "O quadro de feedback precisa dos comentários") {
				t.Error("without comments the board works on its examples and says why it takes no requests")
			}
			// The examples still show (as the other sections' notes); the
			// notice saying so is the preview's only.
			if !strings.Contains(page, `<li class="fbb-item fbb-example" data-kind="bug" data-status="planned"`) || strings.Contains(page, "data-fbb-note") {
				t.Error("without comments the examples show, without the preview's notice")
			}
			continue
		}
		for _, want := range []string{
			`data-fb-board`, `data-fbb-issues="https://github.com/ana/produto-feedback/issues"`,
			`data-kind="bug" aria-selected="false"`, `data-kind="improvement"`, `<b data-count>1</b>`, `<dialog class="fbb-dialog"`,
			`<template data-fbb-block>`,
			`<a class="fbb-title" href="#ex-0">Modo escuro</a>`, `data-fbb-ex="0"`, `data-fbb-example-block`, `role="listbox"`, `<use href="#i-message-square"/>`, `<use href="#i-chevron-up"/>`,
			`<span class="fbb-status s-planned">Planejado</span>`, `<span class="fbb-cat">Relatos de bug</span>`,
		} {
			if !strings.Contains(page, want) {
				t.Errorf("the board page is missing %q", want)
			}
		}
		// The published board shows the examples without a word; the
		// preview says they are examples, to be removed from the note.
		if strings.Contains(page, "data-fbb-note") {
			t.Error("the published board shows the preview's notice")
		}
		cmd := bin.Command(context.Background(), dir, "--logLevel", "warn", "--destination", "preview")
		cmd.Env = append(os.Environ(), "HUGO_BLUP_MODE=preview")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("hugo: %v\n%s", err, out)
		}
		b, err = os.ReadFile(filepath.Join(dir, "preview", "feedback", "index.html"))
		if err != nil || !strings.Contains(string(b), "data-fbb-note") {
			t.Errorf("the preview's board does not say the requests are examples (%v)", err)
		}
	}
}

// A blog's podcast (blup: podcast), in Mochi and Nori: the episodes are the
// posts with a narration (not hidden, not protected), with square WebP
// covers on the page; the feed is valid RSS for podcast apps, with square
// JPEG covers and the MP3's real size.
func TestPodcastPage(t *testing.T) {
	bin, ok := hugo.Find(context.Background())
	if !ok {
		t.Skip("hugo not installed")
	}
	var png bytes.Buffer
	if err := pngEncode(&png); err != nil {
		t.Fatal(err)
	}
	mp3 := bytes.Repeat([]byte{0xFF, 0xFB, 0x90, 0x00}, 300)
	for _, model := range []string{"mochi", "nori"} {
		t.Run(model, func(t *testing.T) {
			vault := t.TempDir()
			files := map[string][]byte{
				"_index.md":         []byte("---\ntitle: Blog\n---\n"),
				"Podcast/_index.md": []byte("---\ntitle: Podcast\nblup: podcast\ndescription: Ouça os posts.\noutputs: [html, podcast]\npodcast_email: dono@exemplo.com\npodcast_category: Education\n---\nOs episódios do blog.\n\n%%\nComentário do autor.\n%%\n"),
				"Blog/Um.md":        []byte("---\ntitle: Episódio um\ndate: 2026-10-02\nimage: \"[[capa.png]]\"\naudio: \"[[um.mp3]]\"\naudio_seconds: 75\n---\nTexto um.\n"),
				"Blog/Dois.md":      []byte("---\ntitle: Episódio dois\ndate: 2026-10-01\naudio: \"[[dois.mp3]]\"\n---\nTexto dois.\n"),
				"Blog/Mudo.md":      []byte("---\ntitle: Sem narração\ndate: 2026-10-03\n---\nNada.\n"),
				"Blog/Escondido.md": []byte("---\ntitle: Escondido\nhidden: true\naudio: \"[[um.mp3]]\"\n---\n"),
				"Blog/Senha.md":     []byte("---\ntitle: Secreto\npassword: abc\naudio: \"[[um.mp3]]\"\n---\nSegredo.\n"),
				"Anexos/capa.png":   png.Bytes(),
				"Anexos/um.mp3":     mp3,
				"Anexos/dois.mp3":   mp3[:400],
			}
			for rel, b := range files {
				p := filepath.Join(vault, filepath.FromSlash(rel))
				os.MkdirAll(filepath.Dir(p), 0o755)
				if err := os.WriteFile(p, b, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			dir := t.TempDir()
			s := config.NewSite(dir)
			s.Title, s.Lang, s.Model, s.Vault = "Blog", "pt", model, vault
			if err := Write(s); err != nil {
				t.Fatal(err)
			}
			if out, err := bin.Command(context.Background(), dir, "--logLevel", "warn", "--baseURL", "https://blog.exemplo.com/").CombinedOutput(); err != nil {
				t.Fatalf("hugo: %v\n%s", err, out)
			}
			page, err := os.ReadFile(filepath.Join(dir, "public", "podcast", "index.html"))
			if err != nil {
				t.Fatal(err)
			}
			html := string(page)
			if n := strings.Count(html, "data-pod-ep "); n != 2 {
				t.Errorf("episodes on the page = %d, want 2", n)
			}
			// Under the title, the note's text (as on the other sections); the
			// description is for Google and sharing, not on the page.
			if !strings.Contains(html, "Os episódios do blog.") || strings.Contains(html, "Comentário do autor") || !strings.Contains(html, `name="description" content="Ouça os posts."`) {
				t.Error("the lede should be the note's text and the description only the page's summary")
			}
			for _, want := range []string{"Episódio um", "Episódio dois", "webp", "data-pod-player", "css/podcast", `type="application/rss+xml"`, "1:15"} {
				if !strings.Contains(html, want) {
					t.Errorf("the page is missing %q", want)
				}
			}
			for _, not := range []string{"Sem narração", "Escondido", "Secreto"} {
				if strings.Contains(html, not) {
					t.Errorf("%q should not be an episode", not)
				}
			}
			// The blog's other pages don't load the podcast's styles.
			if blog, _ := os.ReadFile(filepath.Join(dir, "public", "blog", "index.html")); strings.Contains(string(blog), "css/podcast") || strings.Contains(string(blog), "data-pod-player") {
				t.Error("the blog's section page loads the podcast")
			}
			raw, err := os.ReadFile(filepath.Join(dir, "public", "podcast", "feed.xml"))
			if err != nil {
				t.Fatal(err)
			}
			var feed struct {
				Channel struct {
					Title    string `xml:"title"`
					Email    string `xml:"owner>email"`
					Category struct {
						Text string `xml:"text,attr"`
					} `xml:"category"`
					Items []struct {
						Title     string `xml:"title"`
						Enclosure struct {
							URL    string `xml:"url,attr"`
							Length int    `xml:"length,attr"`
							Type   string `xml:"type,attr"`
						} `xml:"enclosure"`
						Image struct {
							Href string `xml:"href,attr"`
						} `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
						Duration string `xml:"duration"`
					} `xml:"item"`
				} `xml:"channel"`
			}
			if err := xml.Unmarshal(raw, &feed); err != nil {
				t.Fatalf("feed: %v\n%s", err, raw)
			}
			c := feed.Channel
			if c.Email != "dono@exemplo.com" || c.Category.Text != "Education" || len(c.Items) != 2 || !strings.Contains(string(raw), "<description>Ouça os posts.</description>") {
				t.Fatalf("channel = %+v", c)
			}
			one := c.Items[0]
			if one.Title != "Episódio um" || one.Enclosure.Length != len(mp3) || one.Enclosure.Type != "audio/mpeg" || !strings.HasPrefix(one.Enclosure.URL, "https://blog.exemplo.com/") || one.Duration != "75" {
				t.Errorf("first episode = %+v", one)
			}
			if !strings.HasSuffix(one.Image.Href, ".jpg") || !strings.HasPrefix(one.Image.Href, "https://") {
				t.Errorf("episode cover = %q, want a full address of a JPEG", one.Image.Href)
			}
		})
	}
}

// pngEncode writes a small PNG (a post's cover).
func pngEncode(w io.Writer) error {
	img := image.NewRGBA(image.Rect(0, 0, 600, 400))
	for i := range img.Pix {
		img.Pix[i] = byte(i)
	}
	return png.Encode(w, img)
}
