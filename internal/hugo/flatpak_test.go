package hugo

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFlatpakRequiresPackagedHugo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Flatpak is a Linux installation")
	}
	t.Setenv("FLATPAK_ID", "io.getblup.Studio")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	cached, _ := cachedPath()
	if err := os.MkdirAll(filepath.Dir(cached), 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := []byte("#!/bin/sh\nprintf 'hugo v" + Pinned + "+extended linux/amd64\\n'\n")
	if err := os.WriteFile(cached, fixture, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := Find(t.Context()); ok {
		t.Fatal("Flatpak must not run a previously downloaded Hugo")
	}
	if _, err := Download(t.Context(), func(int64, int64) { t.Fatal("download started") }); !errors.Is(err, ErrBundledMissing) {
		t.Fatalf("missing packaged Hugo: %v", err)
	}
	dir := t.TempDir()
	packaged := filepath.Join(dir, "hugo")
	if err := os.WriteFile(packaged, fixture, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got, ok := Find(t.Context())
	if !ok || got.Path != packaged || got.Version != Pinned {
		t.Fatalf("packaged Hugo not found: %+v, %v", got, ok)
	}
}
