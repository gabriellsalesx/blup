package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBuildFlags(t *testing.T) {
	o, rest, err := parseBuildFlags([]string{"--url", "https://x.pages.dev/bento", "--saida=../landing/bento", "~/Sites/blup"})
	if err != nil {
		t.Fatal(err)
	}
	if o.url != "https://x.pages.dev/bento/" || filepath.Base(o.out) != "bento" || len(rest) != 1 || rest[0] != "~/Sites/blup" {
		t.Fatalf("got %+v %v", o, rest)
	}
	if _, _, err := parseBuildFlags([]string{"--url", "bento"}); err == nil {
		t.Fatal("relative url must fail")
	}
	if _, _, err := parseBuildFlags([]string{"--out"}); err == nil {
		t.Fatal("missing value must fail")
	}
}

func TestClearOutRefusesForeignFolder(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("landing"), 0o644)
	if err := clearOut(dir); err == nil {
		t.Fatal("a folder Blup didn't fill must not be wiped")
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		t.Fatal("file was deleted")
	}
	os.WriteFile(filepath.Join(dir, buildMarker), nil, 0o644)
	if err := clearOut(dir); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("not emptied: %v", entries)
	}
}
