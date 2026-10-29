package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPasteImageAndSnapshot(t *testing.T) {
	site := settingsSite(t)
	md, err := PasteImage(site, "Escrevendo/Post.md", "image.png", []byte("png"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "](../" + attachmentsFolder(site) + "/"; !strings.Contains(md, want) || !strings.Contains(md, "%20") {
		t.Fatalf("link: %s", md)
	}
	md2, _ := PasteImage(site, "Top.md", "foto.png", []byte("png"))
	md3, _ := PasteImage(site, "Top.md", "foto.png", []byte("png"))
	if md2 != "![foto]("+attachmentsFolder(site)+"/foto.png)" || md3 != "![foto]("+attachmentsFolder(site)+"/foto%202.png)" {
		t.Fatalf("names: %s %s", md2, md3)
	}
	if _, err := PasteImage(site, "Top.md", "x.exe", []byte("x")); err == nil {
		t.Fatal("accepted a non-image")
	}

	root := site.ContentDir()
	os.WriteFile(filepath.Join(root, "A.md"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(root, "B.md"), []byte("b"), 0o644)
	snap, err := TakeSnapshot(site)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "A.md"), []byte("changed"), 0o644)
	os.Remove(filepath.Join(root, "B.md"))
	os.WriteFile(filepath.Join(root, "C.md"), []byte("new"), 0o644)
	ch, err := SnapshotChanges(site, snap)
	if err != nil || len(ch) != 3 || ch[0] != (Change{"A.md", "changed"}) || ch[1] != (Change{"B.md", "removed"}) || ch[2] != (Change{"C.md", "added"}) {
		t.Fatalf("changes: %v %v", ch, err)
	}
	if err := UndoSnapshot(site, snap); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(filepath.Join(root, "A.md"))
	_, errB := os.Stat(filepath.Join(root, "B.md"))
	_, errC := os.Stat(filepath.Join(root, "C.md"))
	if string(a) != "a" || errB != nil || errC == nil {
		t.Fatalf("undo: %q %v %v", a, errB, errC)
	}
	if ch, _ := SnapshotChanges(site, snap); len(ch) != 0 {
		t.Fatalf("after undo: %v", ch)
	}
}
