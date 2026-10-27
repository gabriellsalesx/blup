package wikilinks

import (
	"archive/zip"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// vault creates files (vault-relative slash path -> content) in a temp dir.
func vault(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for p, c := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

var testVault = map[string]string{
	"Home.md":                           "# Home\n",
	"Ação Rápida.md":                    "# Ação Rápida\n\n## Passo 1: Início\n\n## Seção\n\n## Seção\n\n## Título {#custom}\n\n## **Negrito** e `código`\n",
	"Notas/Nota A.md":                   "# Nota A\n\n## Sub Título\n",
	"Notas/Sub/Funda.md":                "hello\n",
	"Notas/Dup.md":                      "dup in Notas\n",
	"Outros/Dup.md":                     "dup in Outros\n",
	"Outros/Deep/More/Dup.md":           "deep dup\n",
	"Outros/Leitor.md":                  "",
	"Anexos/Foto 1.png":                 "png",
	"Anexos/doc.pdf":                    "pdf",
	"Anexos/Música (ao vivo).mp3":       "mp3",
	"Notas/Imagem.PNG":                  "png",
	".obsidian/app.json":                "{}",
	".trash/Lixo.md":                    "",
	"Projetos/2026/Plano.md":            "# Plano\n\nSetext Title\n============\n",
	"Projetos/Pasta/Nota.md":            "",
	"Outro Projeto/Pasta/Nota.md":       "",
	"Caso/MAIÚSCULA.md":                 "",
	"Notas/Self.md":                     "## Minha Seção\n",
	"Notas/v1.2 release.md":             "",
	"Projetos/2026/Sub/Profundo.md":     "",
	"Projetos/2026/Sub/Anexo Local.png": "png",
}

func TestConvertText(t *testing.T) {
	root := vault(t, testVault)
	idx, err := NewIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, note, in, want string
		problems             int // problems expected (warnings included)
	}{
		{"simple", "Home.md", "See [[Nota A]].", "See [Nota A](Notas/Nota%20A.md).", 0},
		{"alias", "Home.md", "[[Nota A|a nota]]", "[a nota](Notas/Nota%20A.md)", 0},
		{"accents", "Home.md", "[[Ação Rápida]]", "[Ação Rápida](Ação%20Rápida.md)", 0},
		{"relative up", "Notas/Sub/Funda.md", "[[Home]]", "[Home](../../Home.md)", 0},
		{"heading", "Home.md", "[[Nota A#Sub Título]]", "[Nota A > Sub Título](Notas/Nota%20A.md#sub-título)", 0},
		{"heading alias", "Home.md", "[[Nota A#Sub Título|aqui]]", "[aqui](Notas/Nota%20A.md#sub-título)", 0},
		{"heading punctuation matched", "Home.md", "[[Ação Rápida#Passo 1 Início]]", "[Ação Rápida > Passo 1 Início](Ação%20Rápida.md#passo-1-início)", 0},
		{"heading duplicate first", "Home.md", "[[Ação Rápida#Seção]]", "[Ação Rápida > Seção](Ação%20Rápida.md#seção)", 0},
		{"heading custom id", "Home.md", "[[Ação Rápida#Título]]", "[Ação Rápida > Título](Ação%20Rápida.md#custom)", 0},
		{"heading markup", "Home.md", "[[Ação Rápida#Negrito e código]]", "[Ação Rápida > Negrito e código](Ação%20Rápida.md#negrito-e-código)", 0},
		{"heading setext", "Home.md", "[[Plano#Setext Title]]", "[Plano > Setext Title](Projetos/2026/Plano.md#setext-title)", 0},
		{"heading missing", "Home.md", "[[Nota A#Não Existe!]]", "[Nota A > Não Existe!](Notas/Nota%20A.md#não-existe)", 1},
		{"nested heading", "Home.md", "[[Nota A#Nota A#Sub Título]]", "[Nota A > Sub Título](Notas/Nota%20A.md#sub-título)", 0},
		{"same note heading", "Notas/Self.md", "## Minha Seção\n[[#Minha Seção]]", "## Minha Seção\n[Minha Seção](#minha-seção)", 0},
		{"same note heading alias", "Notas/Self.md", "## Minha Seção\n[[#Minha Seção|topo]]", "## Minha Seção\n[topo](#minha-seção)", 0},
		{"same note uses current text", "Notas/Self.md", "## Nova\n[[#Nova]]", "## Nova\n[Nova](#nova)", 0},
		{"block ref", "Home.md", "[[Nota A#^abc123]]", "[Nota A](Notas/Nota%20A.md)", 0},
		{"block ref alias", "Home.md", "[[Nota A#^abc123|ver]]", "[ver](Notas/Nota%20A.md)", 0},
		{"same note block ref", "Home.md", "[[#^abc]]", "[[#^abc]]", 1},
		{"image embed", "Home.md", "![[Foto 1.png]]", "![Foto 1](Anexos/Foto%201.png)", 0},
		{"image width", "Home.md", "![[Foto 1.png|300]]", "![Foto 1|300](Anexos/Foto%201.png)", 0},
		{"image size", "Home.md", "![[Foto 1.png|300x200]]", "![Foto 1|300x200](Anexos/Foto%201.png)", 0},
		{"image caption", "Home.md", "![[Foto 1.png|Uma foto]]", "![Uma foto](Anexos/Foto%201.png)", 0},
		{"image case insensitive", "Home.md", "![[imagem.png]]", "![Imagem](Notas/Imagem.PNG)", 0},
		{"image near note", "Projetos/2026/Sub/Profundo.md", "![[Anexo Local.png]]", "![Anexo Local](Anexo%20Local.png)", 0},
		{"note embed", "Home.md", "![[Nota A]]", "[Nota A](Notas/Nota%20A.md)", 1},
		{"note section embed", "Home.md", "![[Nota A#Sub Título]]", "[Nota A > Sub Título](Notas/Nota%20A.md#sub-título)", 1},
		{"pdf embed", "Home.md", "![[doc.pdf]]", "[doc.pdf](Anexos/doc.pdf)", 1},
		{"pdf link page", "Home.md", "[[doc.pdf#page=3]]", "[doc.pdf > page=3](Anexos/doc.pdf#page=3)", 0},
		{"attachment parens", "Home.md", "[[Música (ao vivo).mp3]]", "[Música (ao vivo).mp3](Anexos/Música%20%28ao%20vivo%29.mp3)", 0},
		{"explicit .md", "Home.md", "[[Nota A.md]]", "[Nota A.md](Notas/Nota%20A.md)", 0},
		{"dotted name", "Home.md", "[[v1.2 release]]", "[v1.2 release](Notas/v1.2%20release.md)", 0},
		{"folder suffix", "Home.md", "[[Pasta/Nota]]", "[Pasta/Nota](Projetos/Pasta/Nota.md)", 0},
		{"folder suffix longer", "Home.md", "[[Outro Projeto/Pasta/Nota]]", "[Outro Projeto/Pasta/Nota](Outro%20Projeto/Pasta/Nota.md)", 0},
		{"folder suffix near", "Outro Projeto/x.md", "[[Pasta/Nota]]", "[Pasta/Nota](Pasta/Nota.md)", 0},
		{"relative target", "Notas/Sub/Funda.md", "[[../Nota A]]", "[../Nota A](../Nota%20A.md)", 0},
		{"absolute target", "Notas/Sub/Funda.md", "[[/Home]]", "[/Home](../../Home.md)", 0},
		{"duplicate same folder", "Notas/Nota A.md", "[[Dup]]", "[Dup](Dup.md)", 0},
		{"duplicate same folder 2", "Outros/Leitor.md", "[[Dup]]", "[Dup](Dup.md)", 0},
		{"duplicate shortest", "Home.md", "[[Dup]]", "[Dup](Notas/Dup.md)", 0},
		{"duplicate closest", "Outros/Deep/x.md", "[[Dup]]", "[Dup](../Dup.md)", 0},
		{"case fallback", "Home.md", "[[maiúscula]]", "[maiúscula](Caso/MAIÚSCULA.md)", 0},
		{"unresolved", "Home.md", "x [[Não Existe]] y", "x [[Não Existe]] y", 1},
		{"trash ignored", "Home.md", "[[Lixo]]", "[[Lixo]]", 1},
		{"bracket escaping in alias", "Home.md", "[[Home|a [b] c]]", "[a \\[b\\] c](Home.md)", 0},
		{"alias with brackets escaped", "Home.md", "[[Home|a ]b c]]", "[a \\]b c](Home.md)", 0},
		{"table pipe", "Home.md", "| [[Nota A\\|alias]] |", "| [alias](Notas/Nota%20A.md) |", 0},
		{"two links", "Home.md", "[[Home]] e [[Nota A]]", "[Home](Home.md) e [Nota A](Notas/Nota%20A.md)", 0},
		{"escaped", "Home.md", `\[[Home]]`, `\[[Home]]`, 0},
		{"inline code", "Home.md", "`[[Home]]` e [[Home]]", "`[[Home]]` e [Home](Home.md)", 0},
		{"double backtick code", "Home.md", "``a ` [[Home]]`` [[Home]]", "``a ` [[Home]]`` [Home](Home.md)", 0},
		{"unclosed backtick", "Home.md", "` [[Home]]", "` [Home](Home.md)", 0},
		{"fenced backticks", "Home.md", "```\n[[Home]]\n```\n[[Home]]", "```\n[[Home]]\n```\n[Home](Home.md)", 0},
		{"fenced tildes", "Home.md", "~~~md\n[[Home]]\n~~~\n[[Home]]", "~~~md\n[[Home]]\n~~~\n[Home](Home.md)", 0},
		{"fence longer close", "Home.md", "````\n```\n[[Home]]\n````\n[[Home]]", "````\n```\n[[Home]]\n````\n[Home](Home.md)", 0},
		{"fence in list", "Home.md", "- item\n  ```\n  [[Home]]\n  ```\n- [[Home]]", "- item\n  ```\n  [[Home]]\n  ```\n- [Home](Home.md)", 0},
		{"fence in quote", "Home.md", "> ```\n> [[Home]]\n> ```\n> [[Home]]", "> ```\n> [[Home]]\n> ```\n> [Home](Home.md)", 0},
		{"unclosed fence", "Home.md", "```\n[[Home]]", "```\n[[Home]]", 0},
		{"frontmatter", "Home.md", "---\nup: \"[[Home]]\"\n---\n[[Home]]", "---\nup: \"[[Home]]\"\n---\n[Home](Home.md)", 0},
		{"not frontmatter", "Home.md", "texto\n---\n[[Home]]\n---\n", "texto\n---\n[Home](Home.md)\n---\n", 0},
		{"unclosed frontmatter", "Home.md", "---\n[[Home]]", "---\n[Home](Home.md)", 0},
		{"crlf", "Home.md", "---\r\na: [[Home]]\r\n---\r\n[[Home]]\r\n```\r\n[[Home]]\r\n```\r\n[[Nota A]]\r\n", "---\r\na: [[Home]]\r\n---\r\n[Home](Home.md)\r\n```\r\n[[Home]]\r\n```\r\n[Nota A](Notas/Nota%20A.md)\r\n", 0},
		{"no links", "Home.md", "plain [text](x.md) and [ [no] ]", "plain [text](x.md) and [ [no] ]", 0},
		{"empty", "Home.md", "[[]]", "[[]]", 1},
		{"spaces trimmed", "Home.md", "[[ Nota A | alias ]]", "[alias](Notas/Nota%20A.md)", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _, probs := ConvertText(idx, tc.note, tc.in)
			if got != tc.want {
				t.Errorf("got\n%q\nwant\n%q", got, tc.want)
			}
			if len(probs) != tc.problems {
				t.Errorf("problems = %d (%+v), want %d", len(probs), probs, tc.problems)
			}
		})
	}
}

func TestChangesAndProblemsDetail(t *testing.T) {
	root := vault(t, testVault)
	idx, err := NewIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	in := "a\n\nx [[Home]] y\n```\n[[Nope]]\n```\n![[Nota A]] [[Nope]]\n"
	_, ch, pr := ConvertText(idx, "Notas/Sub/Funda.md", in)
	if len(ch) != 2 {
		t.Fatalf("changes = %+v", ch)
	}
	if ch[0] != (Change{File: "Notas/Sub/Funda.md", Line: 3, Old: "[[Home]]", New: "[Home](../../Home.md)"}) {
		t.Errorf("change 0 = %+v", ch[0])
	}
	if ch[1].Line != 7 || ch[1].Old != "![[Nota A]]" {
		t.Errorf("change 1 = %+v", ch[1])
	}
	if len(pr) != 2 {
		t.Fatalf("problems = %+v", pr)
	}
	if !pr[0].Warning || pr[0].Line != 7 {
		t.Errorf("embed warning = %+v", pr[0])
	}
	if pr[1].Warning || pr[1].Link != "[[Nope]]" || pr[1].Line != 7 {
		t.Errorf("unresolved = %+v", pr[1])
	}
}

func TestEscapePath(t *testing.T) {
	got := escapePath("../Pasta Nova/C# (1) [x] 100%?.md")
	want := "../Pasta%20Nova/C%23%20%281%29%20%5Bx%5D%20100%25%3F.md"
	if got != want {
		t.Errorf("escapePath = %q, want %q", got, want)
	}
}

func TestSlug(t *testing.T) {
	tests := map[string]string{
		"Hello World":         "hello-world",
		"Ação Rápida":         "ação-rápida",
		"Passo 1: Início":     "passo-1-início",
		"  trim  me ":         "trim--me",
		"a - b":               "a---b",
		"snake_case & C++":    "snake_case--c",
		"日本語 テキスト":            "日本語-テキスト",
		"!!!":                 "heading",
		"Version 2.0 (beta)?": "version-20-beta",
		"Ünïcödé Çase":        "ünïcödé-çase",
	}
	for in, want := range tests {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseHeadings(t *testing.T) {
	text := "---\ntitle: x\n# not\n---\n# A\n```\n# code\n```\n## A\n### A ###\nB\n---\n- item\n---\n## __init__ e snake_case\n## [Link](x.md) e [[Nota|alias]]\n#semtag\n"
	got := parseHeadings(text)
	var ids []string
	for _, h := range got {
		ids = append(ids, h.id)
	}
	want := []string{"a", "a-1", "a-2", "b", "init-e-snake_case", "link-e-alias"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Errorf("ids = %v, want %v", ids, want)
	}
}

func TestResolve(t *testing.T) {
	idx := newIndex("", []string{"a/X.md", "b/c/X.md", "X.png", "d/X.png", "e/f/Y.md", "g/f/Y.md"})
	tests := []struct{ note, target, want string }{
		{"z.md", "X", "a/X.md"},
		{"b/c/z.md", "X", "b/c/X.md"},
		{"b/z.md", "X", "b/c/X.md"},
		{"z.md", "X.png", "X.png"},
		{"d/z.md", "X.png", "d/X.png"},
		{"z.md", "f/Y", "e/f/Y.md"},
		{"g/z.md", "f/Y", "g/f/Y.md"},
		{"z.md", "g/f/Y.md", "g/f/Y.md"},
		{"z.md", "x", "a/X.md"},
	}
	for _, tc := range tests {
		got, ok := idx.Resolve(tc.note, tc.target)
		if !ok || got != tc.want {
			t.Errorf("Resolve(%q, %q) = %q, %v; want %q", tc.note, tc.target, got, ok, tc.want)
		}
	}
	if _, ok := idx.Resolve("z.md", "nope"); ok {
		t.Error("unexpected match for nope")
	}
}

func TestIndexSkipsDotDirs(t *testing.T) {
	root := vault(t, testVault)
	idx, err := NewIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range idx.Files() {
		if strings.HasPrefix(f, ".") {
			t.Errorf("indexed %s", f)
		}
	}
}

func TestConvertTree(t *testing.T) {
	files := map[string]string{
		"Site/Index.md":     "---\ntitle: \"[[Fora]]\"\n---\n[[Página]] e [[Fora]]\r\n",
		"Site/Página.md":    "![[foto.png|200]]\r\nsem mudança [[Inexistente]]\r\n",
		"Site/Sem links.md": "nada\n",
		"Site/foto.png":     "png",
		"Fora.md":           "[[Index]]\n",
		"Site/.hidden/x.md": "[[Index]]\n",
	}
	root := vault(t, files)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(filepath.Join(root, "Site", "Index.md"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	idx, err := NewIndex(root)
	if err != nil {
		t.Fatal(err)
	}

	// Dry run writes nothing.
	rep, err := ConvertTree(idx, root, "Site", true)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Scanned != 3 || rep.Files != 2 {
		t.Fatalf("dry report = %+v", rep)
	}
	sort.Strings(rep.Changed)
	if strings.Join(rep.Changed, ",") != "Site/Index.md,Site/Página.md" {
		t.Errorf("changed = %v", rep.Changed)
	}
	b, _ := os.ReadFile(filepath.Join(root, "Site", "Index.md"))
	if string(b) != files["Site/Index.md"] {
		t.Fatal("dry run modified a file")
	}
	// Problems: [[Fora]] outside Site (warning) and [[Inexistente]].
	var warn, errs int
	for _, p := range rep.Problems {
		if p.Warning {
			warn++
		} else {
			errs++
		}
	}
	if warn != 1 || errs != 1 {
		t.Errorf("problems = %+v", rep.Problems)
	}

	// Backup only the changed files.
	zipPath := filepath.Join(t.TempDir(), "bk", "backup.zip")
	if err := BackupFiles(root, rep.Changed, zipPath); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	zr.Close()
	sort.Strings(names)
	if strings.Join(names, ",") != "Site/Index.md,Site/Página.md" {
		t.Errorf("zip entries = %v", names)
	}

	// Real run.
	if _, err := ConvertTree(idx, root, "Site", false); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(root, "Site", "Index.md"))
	want := "---\ntitle: \"[[Fora]]\"\n---\n[Página](Página.md) e [Fora](../Fora.md)\r\n"
	if string(b) != want {
		t.Errorf("Index.md = %q, want %q", b, want)
	}
	b, _ = os.ReadFile(filepath.Join(root, "Site", "Página.md"))
	if want := "![foto|200](foto.png)\r\nsem mudança [[Inexistente]]\r\n"; string(b) != want {
		t.Errorf("Página.md = %q, want %q", b, want)
	}
	b, _ = os.ReadFile(filepath.Join(root, "Fora.md"))
	if string(b) != files["Fora.md"] {
		t.Error("file outside subdir was modified")
	}
	b, _ = os.ReadFile(filepath.Join(root, "Site", ".hidden", "x.md"))
	if string(b) != files["Site/.hidden/x.md"] {
		t.Error("file in dot dir was modified")
	}
	if runtime.GOOS != "windows" {
		fi, _ := os.Stat(filepath.Join(root, "Site", "Index.md"))
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("mode = %v, want 0600", fi.Mode().Perm())
		}
	}
	// No temp files left behind.
	entries, _ := os.ReadDir(filepath.Join(root, "Site"))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".blup-") {
			t.Errorf("leftover temp file %s", e.Name())
		}
	}

	// Second run is a no-op.
	rep, err = ConvertTree(idx, root, "Site", false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Files != 0 {
		t.Errorf("second run changed %v", rep.Changed)
	}
}

func TestConvertTreeWholeVaultAndBackup(t *testing.T) {
	root := vault(t, map[string]string{
		"A.md":           "[[B]]\n",
		"x/B.md":         "[[A]]\n",
		".obsidian/c.md": "[[A]]\n",
	})
	zipPath := filepath.Join(root, "backup.zip")
	if err := Backup(root, "", zipPath); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(zr.File) != 2 {
		t.Errorf("zip has %d files", len(zr.File))
	}
	zr.Close()

	idx, err := NewIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := ConvertTree(idx, root, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Files != 2 || len(rep.Problems) != 0 {
		t.Errorf("report = %+v", rep)
	}
	b, _ := os.ReadFile(filepath.Join(root, "x", "B.md"))
	if string(b) != "[A](../A.md)\n" {
		t.Errorf("B.md = %q", b)
	}
}

func TestConvertTreeBadSubdir(t *testing.T) {
	root := vault(t, map[string]string{"A.md": ""})
	idx, _ := NewIndex(root)
	for _, sub := range []string{"../x", "nope"} {
		if _, err := ConvertTree(idx, root, sub, true); err == nil {
			t.Errorf("subdir %q: expected error", sub)
		}
	}
}
