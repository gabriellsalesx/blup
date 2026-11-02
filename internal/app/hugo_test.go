package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/lock"
)

// encryptLocked fails closed: protected markers with a manifest that is
// missing, empty or null never pass as "nothing to encrypt".
func TestEncryptLockedFailsClosed(t *testing.T) {
	site := config.NewSite(t.TempDir())
	page := `<p>x</p><blup-begin data-id="abc"></blup-begin><p>SEGREDO</p><blup-end></blup-end>`
	for name, manifest := range map[string]string{"missing": "", "empty": "{}", "null": "null"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			os.MkdirAll(filepath.Join(dir, "p"), 0o755)
			os.WriteFile(filepath.Join(dir, "p", "index.html"), []byte(page), 0o644)
			if manifest != "" {
				os.WriteFile(filepath.Join(dir, lock.Manifest), []byte(manifest), 0o644)
			}
			err := encryptLocked(context.Background(), site, dir)
			if err == nil {
				t.Fatal("published a protected post in clear")
			}
			if manifest != "null" && !errors.Is(err, lock.ErrLeftover) {
				t.Fatalf("want ErrLeftover, got %v", err)
			}
		})
	}
	clean := t.TempDir()
	os.WriteFile(filepath.Join(clean, "index.html"), []byte("<p>ok</p>"), 0o644)
	if err := encryptLocked(context.Background(), site, clean); err != nil {
		t.Fatal(err)
	}
}
