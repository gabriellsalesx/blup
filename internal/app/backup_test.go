package app

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupSite(t *testing.T) {
	site := settingsSite(t)
	site.Title = "Meu Cofre"
	root := site.ContentDir()
	write(t, root, "Nota.md", "Texto.")
	write(t, root, "Pasta/Outra.md", "Mais.")
	write(t, root, ".git/HEAD", "ref")
	dir := t.TempDir()
	for i := 1; i <= 8; i++ { // older copies, the oldest beyond 7 goes
		write(t, dir, "Meu Cofre 2020-01-0"+string(rune('0'+i))+".zip", "x")
	}
	p, err := BackupSite(site, dir)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	zr.Close()
	if !names["Nota.md"] || !names["Pasta/Outra.md"] || names[".git/HEAD"] {
		t.Fatalf("zip = %v", names)
	}
	if again, err := BackupSite(site, dir); err != nil || again != p {
		t.Fatalf("second backup today = %s %v", again, err)
	}
	left, _ := filepath.Glob(filepath.Join(dir, "Meu Cofre *.zip"))
	if len(left) != BackupKeep {
		t.Fatalf("copies = %d", len(left))
	}
	if _, err := os.Stat(filepath.Join(dir, "Meu Cofre 2020-01-01.zip")); err == nil {
		t.Fatal("the oldest copy should be gone")
	}
	if _, err := BackupSite(site, filepath.Join(root, "bk")); err == nil {
		t.Fatal("a folder inside the notes should be refused")
	}
}
