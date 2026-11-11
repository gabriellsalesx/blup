package pagefind

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type rejectNetwork struct{ calls int }

func (r *rejectNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	r.calls++
	return nil, errors.New("network must not be used")
}

func TestFlatpakUsesPackagedPagefind(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Flatpak is a Linux installation")
	}
	t.Setenv("FLATPAK_ID", "io.getblup.Studio")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	transport := &rejectNetwork{}
	old := downloader
	downloader = &http.Client{Transport: transport}
	t.Cleanup(func() { downloader = old })
	for _, c := range []struct {
		version string
		wantOK  bool
	}{
		{Pinned, true},
		{"1.5.1", false},
		{Pinned + "-beta.1", false},
		{"not-a-version", false},
	} {
		t.Run(c.version, func(t *testing.T) {
			dir := t.TempDir()
			tool := filepath.Join(dir, "pagefind")
			if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf 'pagefind "+c.version+"\\n'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)
			got, err := Ensure(t.Context())
			if c.wantOK {
				if err != nil || got != tool {
					t.Fatalf("packaged Pagefind: %q, %v", got, err)
				}
			} else if !errors.Is(err, ErrBundledMissing) {
				t.Fatalf("incompatible packaged Pagefind must fail without downloading: %v", err)
			}
		})
	}
	// A stale cache from the old bundle must not bypass the package's tool.
	t.Setenv("PATH", t.TempDir())
	cached, _ := cachedPath()
	if err := os.MkdirAll(filepath.Dir(cached), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cached, []byte("old downloaded executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Ensure(context.Background()); !errors.Is(err, ErrBundledMissing) {
		t.Fatalf("downloaded cache used in Flatpak: %v", err)
	}
	if transport.calls != 0 {
		t.Fatalf("Flatpak made %d HTTP requests", transport.calls)
	}
}

func TestNativePagefindKeepsItsCache(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("XDG cache fixture")
	}
	t.Setenv("FLATPAK_ID", "")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cached, _ := cachedPath()
	if err := os.MkdirAll(filepath.Dir(cached), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cached, []byte("cached"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Ensure(t.Context())
	if err != nil || got != cached {
		t.Fatalf("native cache: %q, %v", got, err)
	}
}
