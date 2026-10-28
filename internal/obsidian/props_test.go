package obsidian

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetProperties(t *testing.T) {
	p := filepath.Join(t.TempDir(), "_index.md")
	os.WriteFile(p, []byte("---\ntitle: Site # keep\ntags: [a]\n---\nHello **world**.\n"), 0o644)
	err := SetProperties(p,
		Prop{"site_name", "Acme: Cloud"},
		Prop{"tags", nil},
		Prop{"menu", []string{"Docs", "GitHub | https://github.com/acme"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	want := "---\ntitle: Site # keep\nsite_name: 'Acme: Cloud'\nmenu:\n  - Docs\n  - GitHub | https://github.com/acme\n---\nHello **world**.\n"
	if string(b) != want {
		t.Fatalf("got\n%s\nwant\n%s", b, want)
	}
	n, err := ReadNote(p)
	if err != nil {
		t.Fatal(err)
	}
	if n.String("site_name") != "Acme: Cloud" || len(n.List("menu")) != 2 || n.Body != "Hello **world**.\n" {
		t.Fatalf("read back: %+v", n)
	}
}

func TestSetPropertiesNewNote(t *testing.T) {
	p := filepath.Join(t.TempDir(), "Autores", "Ana.md")
	if err := SetProperties(p, Prop{"role", "Designer"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "---\nrole: Designer\n---\n" {
		t.Fatalf("got %q", b)
	}
	if n, _ := ReadNote(filepath.Join(t.TempDir(), "none.md")); len(n.Props) != 0 {
		t.Fatal("missing note must be empty")
	}
}

func TestSetBody(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.md")
	os.WriteFile(p, []byte("---\nrole: X\n---\nold\n"), 0o644)
	if err := SetBody(p, "new bio\n"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "---\nrole: X\n---\nnew bio\n" {
		t.Fatalf("got %q", b)
	}
}
