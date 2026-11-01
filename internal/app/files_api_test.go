package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRenameAndMoveFixLinks(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	write(t, root, "Blog/Meu post.md", "Veja [as dicas](../Guia/Dicas%20boas.md#inicio) e ![foto](../Anexos/a.png).\n")
	write(t, root, "Guia/Dicas boas.md", "Volte ao [post](../Blog/Meu%20post.md). [[Meu post|o post]] e [[Outra]].\n")
	write(t, root, "Sobre.md", "Leia [o post](Blog/Meu%20post.md) e <https://x.dev>.\n")

	// Rename: links and [[wikilinks]] follow the new name.
	newRel, err := RenameItem(site, "Blog/Meu post.md", "Primeiro post")
	if err != nil || newRel != "Blog/Primeiro post.md" {
		t.Fatalf("rename = %q, %v", newRel, err)
	}
	if g := read(t, root, "Guia/Dicas boas.md"); !strings.Contains(g, "](../Blog/Primeiro%20post.md)") || !strings.Contains(g, "[[Primeiro post|o post]]") || !strings.Contains(g, "[[Outra]]") {
		t.Fatalf("guide = %q", g)
	}
	if s := read(t, root, "Sobre.md"); !strings.Contains(s, "](Blog/Primeiro%20post.md)") {
		t.Fatalf("about = %q", s)
	}

	// Move: links to it, and its own relative links, are fixed.
	newRel, err = MoveItem(site, "Blog/Primeiro post.md", "")
	if err != nil || newRel != "Primeiro post.md" {
		t.Fatalf("move = %q, %v", newRel, err)
	}
	if p := read(t, root, "Primeiro post.md"); !strings.Contains(p, "](Guia/Dicas%20boas.md#inicio)") || !strings.Contains(p, "](Anexos/a.png)") {
		t.Fatalf("moved post = %q", p)
	}
	if s := read(t, root, "Sobre.md"); !strings.Contains(s, "](Primeiro%20post.md)") {
		t.Fatalf("about = %q", s)
	}

	// A renamed folder: links into it follow.
	if _, err := RenameItem(site, "Guia", "Manual"); err != nil {
		t.Fatal(err)
	}
	if p := read(t, root, "Primeiro post.md"); !strings.Contains(p, "](Manual/Dicas%20boas.md#inicio)") {
		t.Fatalf("post after folder rename = %q", p)
	}
	if _, err := MoveItem(site, "Manual", "Manual"); err == nil {
		t.Fatal("a folder can't go inside itself")
	}
	if _, err := RenameItem(site, "Sobre.md", "Primeiro post"); err == nil {
		t.Fatal("a taken name must be refused")
	}
}

func TestCopyDeleteRestore(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	write(t, root, "Blog/Post.md", "texto\n")
	c, err := CopyNote(site, "Blog/Post.md")
	if err != nil || c != "Blog/Post 2.md" || read(t, root, c) != "texto\n" {
		t.Fatalf("copy = %q, %v", c, err)
	}
	trashed, err := DeleteItem(site, "Blog/Post.md")
	if err != nil {
		t.Fatal(err)
	}
	items, _ := ListTrash(site)
	if len(items) != 1 || items[0].Origin != "Blog/Post.md" || items[0].Name != "Post" {
		t.Fatalf("trash = %+v", items)
	}
	if err := RestoreSection(site, trashed); err != nil {
		t.Fatal(err)
	}
	if read(t, root, "Blog/Post.md") != "texto\n" {
		t.Fatal("restored to where it was")
	}
	if items, _ := ListTrash(site); len(items) != 0 {
		t.Fatalf("trash = %+v", items)
	}
	if _, err := DeleteItem(site, ""); err == nil {
		t.Fatal("the notes folder itself can't be deleted")
	}
}

// A canvas or a drawing keeps its own extension when renamed or copied.
func TestRenameAndCopyKeepBoardExt(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	write(t, root, "Ideias.canvas", "{}")
	write(t, root, "Mapa.draw", "{}")
	if rel, err := RenameItem(site, "Ideias.canvas", "Projetos"); err != nil || rel != "Projetos.canvas" {
		t.Fatalf("rename canvas = %q, %v", rel, err)
	}
	if rel, err := RenameItem(site, "Mapa.draw", "Fluxo.draw"); err != nil || rel != "Fluxo.draw" {
		t.Fatalf("rename drawing = %q, %v", rel, err)
	}
	if rel, err := CopyNote(site, "Fluxo.draw"); err != nil || rel != "Fluxo 2.draw" {
		t.Fatalf("copy drawing = %q, %v", rel, err)
	}
}

// A canvas's file cards follow a note renamed or a folder moved.
func TestRenameFixesCanvas(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	write(t, root, "Ideias/Uma.md", "texto")
	write(t, root, "Mapa.canvas", "{\n\t\"nodes\":[\n\t\t{\"id\":\"a\",\"type\":\"file\",\"file\":\"Ideias/Uma.md\",\"x\":0}\n\t],\n\t\"edges\":[]\n}")
	if _, err := RenameItem(site, "Ideias/Uma.md", "Duas"); err != nil {
		t.Fatal(err)
	}
	if c := read(t, root, "Mapa.canvas"); !strings.Contains(c, `"file":"Ideias/Duas.md"`) || !strings.Contains(c, "\n\t\t{") {
		t.Fatalf("canvas = %s", c)
	}
	if _, err := RenameItem(site, "Ideias", "Pensamentos"); err != nil {
		t.Fatal(err)
	}
	if c := read(t, root, "Mapa.canvas"); !strings.Contains(c, `"file":"Pensamentos/Duas.md"`) {
		t.Fatalf("canvas after folder = %s", c)
	}
}
