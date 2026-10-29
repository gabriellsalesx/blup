package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttachments(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	folder := attachmentsFolder(site)
	write(t, root, folder+"/minha foto.png", "png")
	write(t, root, folder+"/sobra.pdf", "pdf")
	write(t, root, "Post.md", "---\nimage: \"[[minha foto.png]]\"\n---\n![a](../"+folder+"/minha%20foto.png)\n")

	list, err := ListAttachments(site)
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %+v %v", list, err)
	}
	for _, a := range list {
		if a.Name == "minha foto.png" && (len(a.Uses) != 1 || a.Kind != "image") || a.Name == "sobra.pdf" && len(a.Uses) != 0 {
			t.Fatalf("uses: %+v", a)
		}
	}
	name, err := RenameAttachment(site, "minha foto.png", "capa nova")
	if err != nil || name != "capa nova.png" {
		t.Fatalf("rename: %q %v", name, err)
	}
	if p := read(t, root, "Post.md"); !strings.Contains(p, "[[capa nova.png]]") || !strings.Contains(p, "/capa%20nova.png)") {
		t.Fatalf("links: %s", p)
	}
	src := filepath.Join(t.TempDir(), "sobra.pdf")
	os.WriteFile(src, []byte("x"), 0o644)
	added, err := AddAttachments(site, []string{src})
	if err != nil || len(added) != 1 || added[0] != "sobra 2.pdf" {
		t.Fatalf("add: %v %v", added, err)
	}
	if _, err := DeleteAttachment(site, "sobra.pdf"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, folder, "sobra.pdf")); err == nil {
		t.Fatal("not deleted")
	}
}
