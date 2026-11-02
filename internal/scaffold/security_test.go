package scaffold

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/hugo"
	"github.com/gabriellsalesx/blup/internal/lock"
)

// hostileNotes are notes written to break the site: HTML and scripts in
// every field the theme prints, links with dangerous schemes, and private
// text (comments, drafts, protected notes) that must never be published.
var hostileNotes = map[string]string{
	"Site/_index.md": "---\nsite_name: \"Nome<script>alert('sn')</script>\"\nsocial:\n  - \"javascript:alert('social')\"\n---\nInício\n",
	"Site/Posts/XSS.md": `---
title: "<script>alert('title')</script>T\"itulo"
description: "\"><script>alert('desc')</script>"
tags: ["a\"b", "<img src=x onerror=alert('tag')>", "</script><script>alert('tagjs')</script>"]
image: "javascript:alert('img')"
date: 2026-01-01
---
# H1 <script>alert('h1')</script>

[js](javascript:alert('link')) [js2](JaVaScRiPt:alert('link2')) [data](data:text/html,x)
[t](https://ok.example "titulo\" onmouseover=alert('t')")
<script>alert('raw')</script>
<img src=x onerror=alert('rawimg')>

> [!note] Titulo <img src=x onerror=alert('callout')>
> corpo

## Heading <img src=x onerror=alert('hd')>
`,
	// A comment longer than the summary: the summary ends inside it.
	"Site/Posts/Comentario.md": "---\ndate: 2026-01-02\n---\n%%\nCOMENTARIO-RESUMO " + strings.Repeat("palavra ", 90) +
		"\n\nsegundo " + strings.Repeat("palavra ", 90) + "\n\n## COMENTARIO-TITULO\n\n![](../../Anexos/capa-secreta.png)\n%%\n" +
		"Texto público " + strings.Repeat("palavra ", 90) + "\n\n## Ação visível\n",
	"Site/Posts/Rascunho.md": "---\ndraft: true\ndate: 2026-01-03\n---\nRASCUNHO-SECRETO\n",
	"Site/Posts/Trancado.md": "---\npassword: abacaxi\ndate: 2026-01-04\n---\nTRANCADO-CORPO\n\n![](../../Anexos/capa-trancada.png)\n\n![](../../Anexos/capa-publica.png)\n\n[pdf](../../Anexos/trancado-doc.pdf)\n\n## TRANCADO-HEADING\n",
	// The same image in a public note and in the protected one: published.
	"Site/Posts/Publico.md": "---\ndate: 2026-01-05\n---\nTexto aberto\n\n![](../../Anexos/capa-publica.png)\n\n" +
		"[a](Escondido.md) [b](Privado.md) [c](Rascunho.md) [d](Texto.md) [e](Trancado.md)\n",
	// Kept off the site like drafts, also with text values.
	"Site/Posts/Escondido.md":        "---\npublish: false\ndate: 2026-01-06\ntags: [ESCONDIDO-TAG]\n---\nESCONDIDO-CORPO\n\n![](../../Anexos/capa-escondida.png)\n",
	"Site/Posts/Privado.md":          "---\nprivate: \"yes\"\ndate: 2026-01-07\ntags: [PRIVADO-TAG]\n---\nPRIVADO-CORPO\n",
	"Site/Posts/Sigilo [x] {a,b}.md": "---\nprivate: true\n---\nSIGILO-CORPO\n",
	"Site/Posts/Texto.md":            "---\npublish: \"no\"\ndate: 2026-01-08\n---\nTEXTO-CORPO\n",
	"Site/Posts/Aberto.md":           "---\npublish: true\nprivate: false\ndate: 2026-01-09\n---\nVISIVEL-CORPO\n",
	// A comment left open hides the rest of the note (code kept as written).
	"Site/Posts/Aberta.md": "---\ndate: 2026-01-10\n---\nAntes do comentário `%%codigo%%`\n\n```\n%% dentro do código\n```\n\n%% SEMFIM-SEGREDO\n\n## SEMFIM-TITULO\n\n![](../../Anexos/capa-semfim.png)\n\n```\nSEMFIM-CODIGO\n```\n",
}

// TestBuildHostile builds a site from hostile notes with every model and
// checks the output: everything escaped, nothing private published.
func TestBuildHostile(t *testing.T) {
	bin, ok := hugo.Find(context.Background())
	if !ok {
		t.Skip("hugo not installed")
	}
	vault := t.TempDir()
	png, err := os.ReadFile("testdata/vault/Anexos/Foto Teste.png")
	if err != nil {
		t.Fatal(err)
	}
	for rel, body := range hostileNotes {
		p := filepath.Join(vault, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	os.MkdirAll(filepath.Join(vault, "Anexos"), 0o755)
	// Attachments used only where nothing may be published: inside a
	// comment, by a protected note, by a hidden one. capa-publica is also
	// used by a public note.
	secretFiles := []string{"capa-secreta", "capa-trancada", "capa-escondida", "capa-semfim", "trancado-doc"}
	for _, name := range []string{"capa-secreta.png", "capa-trancada.png", "capa-escondida.png", "capa-semfim.png", "capa-publica.png"} {
		os.WriteFile(filepath.Join(vault, "Anexos", name), png, 0o644)
	}
	os.WriteFile(filepath.Join(vault, "Anexos", "trancado-doc.pdf"), []byte("%PDF-1.4 TRANCADO-PDF"), 0o644)

	// Markup that would run in the browser if it were not escaped.
	active := regexp.MustCompile(`<script>alert|<img src=x|<svg onload|(?i)(href|src)="?(javascript|data|vbscript):|"\s*on[a-z]+=\S*alert`)
	private := []string{"COMENTARIO-", "RASCUNHO-SECRETO", "TRANCADO-", "abacaxi", "blup-begin",
		"ESCONDIDO-", "PRIVADO-", "SIGILO-", "TEXTO-CORPO", "SEMFIM-", "scondido", "Privado.md", "/privado/", "Texto.md", "/texto/", "Rascunho.md", ".md\""}
	private = append(private, secretFiles...)

	for _, model := range []string{"mochi", "bento", "nori"} {
		t.Run(model, func(t *testing.T) {
			dir := t.TempDir()
			s := config.NewSite(dir)
			s.Title, s.Lang, s.Model, s.Vault, s.Folder = "Hostil", "pt", model, vault, "Site"
			if err := Write(s); err != nil {
				t.Fatal(err)
			}
			// Not minified: the minifier writes "<" bare inside quoted
			// attributes (safe), which the check below would flag.
			// internal/lock tests the bare attributes of --minify.
			cmd := bin.Command(context.Background(), dir, "--baseURL", "https://example.org/", "--logLevel", "error")
			cmd.Env = append(os.Environ(), "HUGO_BLUP_MODE=publish")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("hugo: %v\n%s", err, out)
			}
			public := filepath.Join(dir, "public")
			if n, err := lock.Process(public, ""); err != nil || n != 1 {
				t.Fatalf("lock: %d %v", n, err)
			}
			// The protected note's attachments travel inside its ciphertext.
			locked, err := os.ReadFile(filepath.Join(public, "posts", "trancado", "index.html"))
			if err != nil {
				t.Fatal(err)
			}
			plain := decryptPage(t, string(locked), "abacaxi")
			for _, want := range []string{"TRANCADO-CORPO", `src="data:image/webp;base64,`, `href="data:application/pdf;base64,`, `download="trancado-doc.pdf"`} {
				if !strings.Contains(plain, want) {
					t.Errorf("protected note, opened, lacks %q:\n%s", want, plain)
				}
			}
			if strings.Contains(plain, "/vault/") {
				t.Errorf("protected note still points at published attachments:\n%s", plain)
			}
			if _, err := lock.Prune(public); err != nil {
				t.Fatal(err)
			}
			publicImage := false
			filepath.WalkDir(public, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				rel, _ := filepath.Rel(public, p)
				for _, secret := range secretFiles {
					if strings.Contains(rel, secret) {
						t.Errorf("attachment published: %s", rel)
					}
				}
				if strings.Contains(rel, "capa-publica") {
					publicImage = true
				}
				if strings.HasPrefix(rel, "js"+string(filepath.Separator)) || !strings.HasSuffix(p, ".html") && !strings.HasSuffix(p, ".xml") && !strings.HasSuffix(p, ".json") {
					return nil
				}
				b, _ := os.ReadFile(p)
				for _, secret := range private {
					if strings.Contains(strings.ToLower(string(b)), strings.ToLower(secret)) {
						t.Errorf("%s leaks %q", rel, secret)
					}
				}
				if m := active.FindString(string(b)); m != "" {
					t.Errorf("%s has active markup %q", rel, m)
				}
				return nil
			})
			if !publicImage {
				t.Error("the image of the public note was not published")
			}
			for _, gone := range []string{"escondido", "privado", "texto"} {
				if _, err := os.Stat(filepath.Join(public, "posts", gone)); err == nil {
					t.Errorf("hidden note %s was published", gone)
				}
			}
			if _, err := os.Stat(filepath.Join(public, "posts", "aberto", "index.html")); err != nil {
				t.Error("a note with publish: true is missing")
			}
			open, err := os.ReadFile(filepath.Join(public, "posts", "publico", "index.html"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(open), `/posts/trancado/"`) {
				t.Error("the link to the protected note is gone")
			}
			semfim, err := os.ReadFile(filepath.Join(public, "posts", "aberta", "index.html"))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"Antes do comentário", "<code>%%codigo%%</code>", "%% dentro do código"} {
				if !strings.Contains(string(semfim), want) {
					t.Errorf("the note with an open comment lost %q", want)
				}
			}
			page, err := os.ReadFile(filepath.Join(public, "posts", "comentario", "index.html"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(page), "Texto público") {
				t.Error("the text after the comment is missing")
			}
			if model == "mochi" && !strings.Contains(string(page), `>Ação visível</a>`) {
				t.Error("the table of contents lost a visible heading")
			}
		})
	}
}

// decryptPage opens the protected region of a built page, as lock.js does.
func decryptPage(t *testing.T, page, password string) string {
	t.Helper()
	attr := func(name string) []byte {
		m := regexp.MustCompile(`data-` + name + `="?([^"\s>]+)`).FindStringSubmatch(page)
		if m == nil {
			t.Fatalf("no data-%s in the page", name)
		}
		b, err := base64.StdEncoding.DecodeString(m[1])
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	plain, err := lock.Decrypt(password, lock.Sealed{Salt: attr("salt"), IV: attr("iv"), Ciphertext: attr("ct"), Iterations: lock.Iterations})
	if err != nil {
		t.Fatal(err)
	}
	return string(plain)
}
