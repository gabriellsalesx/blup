package scaffold

import (
	"context"
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
	"Site/Posts/Trancado.md": "---\npassword: abacaxi\ndate: 2026-01-04\n---\nTRANCADO-CORPO\n\n![](../../Anexos/capa-trancada.png)\n\n## TRANCADO-HEADING\n",
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
	for _, name := range []string{"capa-secreta.png", "capa-trancada.png"} {
		os.WriteFile(filepath.Join(vault, "Anexos", name), png, 0o644)
	}

	// Markup that would run in the browser if it were not escaped.
	active := regexp.MustCompile(`<script>alert|<img src=x|<svg onload|(?i)(href|src)="?(javascript|data|vbscript):|"\s*on[a-z]+=\S*alert`)
	private := []string{"COMENTARIO-", "RASCUNHO-SECRETO", "TRANCADO-", "abacaxi", "capa-secreta", "capa-trancada", "blup-begin"}

	for _, model := range []string{"mochi", "bento", "nori"} {
		t.Run(model, func(t *testing.T) {
			dir := t.TempDir()
			s := config.NewSite(dir)
			s.Title, s.Lang, s.Model, s.Vault, s.Folder = "Hostil", "pt", model, vault, "Site"
			if err := Write(s); err != nil {
				t.Fatal(err)
			}
			cmd := bin.Command(context.Background(), dir, "--baseURL", "https://example.org/", "--logLevel", "error")
			cmd.Env = append(os.Environ(), "HUGO_BLUP_MODE=publish")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("hugo: %v\n%s", err, out)
			}
			public := filepath.Join(dir, "public")
			if n, err := lock.Process(public, ""); err != nil || n != 1 {
				t.Fatalf("lock: %d %v", n, err)
			}
			filepath.WalkDir(public, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				// Known limit: images of protected notes (and of comments)
				// are still published as files under hashed names; only the
				// pages are checked here.
				rel, _ := filepath.Rel(public, p)
				if strings.HasPrefix(rel, "js"+string(filepath.Separator)) || !strings.HasSuffix(p, ".html") && !strings.HasSuffix(p, ".xml") && !strings.HasSuffix(p, ".json") {
					return nil
				}
				b, _ := os.ReadFile(p)
				for _, secret := range private {
					if strings.Contains(string(b), secret) {
						t.Errorf("%s leaks %q", rel, secret)
					}
				}
				if m := active.FindString(string(b)); m != "" {
					t.Errorf("%s has active markup %q", rel, m)
				}
				return nil
			})
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
