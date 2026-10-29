package app

import (
	"strings"
	"testing"
)

func TestAuthors(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	write(t, root, "Escritores/_index.md", "---\nblup: authors\n---\n")
	write(t, root, "Escritores/Ana.md", "---\nrole: Editora\n---\nBio da Ana.\n")
	write(t, root, "Post.md", "---\ntitle: Post\nauthors: [Ana]\n---\nTexto.\n")

	list, err := ListAuthors(site)
	if err != nil || len(list) == 0 || list[len(list)-1].Name != "Ana" || list[len(list)-1].Posts != 1 || list[len(list)-1].Bio != "Bio da Ana." {
		t.Fatalf("list: %+v %v", list, err)
	}
	// Rename: the note and the post follow.
	if err := SaveAuthor(site, "Ana", Author{Name: "Ana Lima", Role: "Editora-chefe", Bio: "Nova bio.", Social: []string{"https://x.com/ana", " "}}); err != nil {
		t.Fatal(err)
	}
	if n := read(t, root, "Escritores/Ana Lima.md"); !strings.Contains(n, "Editora-chefe") || !strings.Contains(n, "Nova bio.") || !strings.Contains(n, "https://x.com/ana") {
		t.Fatalf("note: %s", n)
	}
	if p := read(t, root, "Post.md"); !strings.Contains(p, "Ana Lima") {
		t.Fatalf("post: %s", p)
	}
	// New author, then a taken name.
	if err := SaveAuthor(site, "", Author{Name: "Bruno"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveAuthor(site, "", Author{Name: "bruno"}); err == nil {
		t.Fatal("a taken name was accepted")
	}
	if err := DeleteAuthor(site, "Bruno"); err != nil {
		t.Fatal(err)
	}
	list, _ = ListAuthors(site)
	for _, a := range list {
		if a.Name == "Bruno" {
			t.Fatal("deleted author still listed")
		}
	}
}
