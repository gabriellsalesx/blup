package app

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// swapToLink turns the folder dir into a link to out, as a synced or
// downloaded change could between the check (Within) and the write.
func swapToLink(t *testing.T, dir, out string) {
	t.Helper()
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(out, dir); err != nil {
		t.Skip("no symlinks:", err)
	}
}

// The writes in the notes go through os.Root: a folder swapped for a link
// to somewhere else after the path was checked is not followed.
func TestWritesRefuseSwappedLink(t *testing.T) {
	root, out := t.TempDir(), t.TempDir()
	dir := filepath.Join(root, "Posts")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(out, "a.md"), []byte("SEGREDO"), 0o644)
	swapToLink(t, dir, out)

	p := filepath.Join(dir, "a.md")
	if err := WriteInside(root, p, []byte("x")); err == nil {
		t.Error("WriteInside wrote through the link")
	}
	if err := writeNew(root, filepath.Join(dir, "novo.md"), []byte("x")); err == nil {
		t.Error("writeNew wrote through the link")
	}
	if err := MkdirInside(root, filepath.Join(dir, "sub")); err == nil {
		t.Error("MkdirInside made a folder through the link")
	}
	os.WriteFile(filepath.Join(root, "b.md"), []byte("b"), 0o644)
	if err := renameInside(root, filepath.Join(root, "b.md"), filepath.Join(dir, "b.md")); err == nil {
		t.Error("renameInside moved through the link")
	}
	if err := renameInside(root, p, filepath.Join(root, "roubado.md")); err == nil {
		t.Error("renameInside took a file from outside")
	}
	if b, _ := os.ReadFile(filepath.Join(out, "a.md")); string(b) != "SEGREDO" {
		t.Errorf("the file outside changed: %q", b)
	}
	for _, n := range []string{"novo.md", "sub", "b.md"} {
		if _, err := os.Lstat(filepath.Join(out, n)); err == nil {
			t.Errorf("%s landed outside", n)
		}
	}
	// A link to nothing (planted) is not made into a file elsewhere.
	os.Symlink(filepath.Join(out, "plantado.md"), filepath.Join(root, "plantado.md"))
	if err := WriteInside(root, filepath.Join(root, "plantado.md"), []byte("x")); err == nil {
		t.Error("WriteInside followed a dangling link out")
	}
	if _, err := os.Lstat(filepath.Join(out, "plantado.md")); err == nil {
		t.Error("the planted link made a file outside")
	}
	// Paths outside root never reach os.Root.
	for _, bad := range []string{filepath.Join(root, "..", "x.md"), filepath.Join(out, "x.md")} {
		if err := WriteInside(root, bad, []byte("x")); err == nil {
			t.Errorf("WriteInside(%s) worked", bad)
		}
	}
	// Links that stay inside still work.
	os.MkdirAll(filepath.Join(root, "Real"), 0o755)
	os.Symlink("Real", filepath.Join(root, "Atalho"))
	if err := WriteInside(root, filepath.Join(root, "Atalho", "c.md"), []byte("c")); err != nil {
		t.Errorf("a link inside: %v", err)
	}
}

// The same with the swap racing the writes: whatever wins, nothing lands
// outside the notes.
func TestAttachmentsRaceWithLinkSwap(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	out := t.TempDir()
	dir := filepath.Join(root, attachmentsFolder(site))
	os.MkdirAll(dir, 0o755)
	if err := os.Symlink(out, filepath.Join(root, "probe")); err != nil {
		t.Skip("no symlinks:", err)
	}
	src := filepath.Join(t.TempDir(), "foto.png")
	os.WriteFile(src, []byte("png"), 0o644)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			os.RemoveAll(dir)
			os.Symlink(out, dir)
			os.Remove(dir)
			os.MkdirAll(dir, 0o755)
		}
	})
	for range 300 {
		AddAttachments(site, []string{src})
		PasteImage(site, "Post.md", "x.png", []byte("png"))
	}
	close(stop)
	wg.Wait()
	if entries, _ := os.ReadDir(out); len(entries) > 0 {
		t.Errorf("%d files landed outside the notes", len(entries))
	}
}
