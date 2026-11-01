package app

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/scaffold"
)

// The preview's search index is built in the system's temporary folder,
// often another disk (/tmp in memory on Linux), where a rename fails with
// EXDEV. The index must still land in the site, replacing the old one.
func TestPlaceSearchAcrossDisks(t *testing.T) {
	site := config.NewSite(t.TempDir())
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "pagefind", "index"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "pagefind", "index", "a.pf"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(site.Dir(), filepath.FromSlash(scaffold.PreviewStatic), "pagefind")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "old.pf"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	defer func(r func(string, string) error) { rename = r }(rename)
	rename = func(src, dst string) error {
		return &os.LinkError{Op: "rename", Old: src, New: dst, Err: syscall.EXDEV}
	}
	if err := placeSearch(site, tmp); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dest, "index", "a.pf")); err != nil || string(b) != "new" {
		t.Errorf("index = %q, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "old.pf")); !os.IsNotExist(err) {
		t.Errorf("the old index stayed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "pagefind")); !os.IsNotExist(err) {
		t.Errorf("the temporary index stayed: %v", err)
	}
}
