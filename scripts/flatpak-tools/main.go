// Command flatpak-tools prepares Hugo and Pagefind for the project's direct
// Flatpak bundle. It uses the same pinned, checksum-verified downloads as the
// native application. Flathub builds these tools from source separately.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gabriellsalesx/blup/internal/hugo"
	"github.com/gabriellsalesx/blup/internal/pagefind"
)

func main() {
	out := flag.String("out", "", "directory for the bundle's hugo and pagefind executables")
	flag.Parse()
	if *out == "" || runtime.GOOS != "linux" {
		fmt.Fprintln(os.Stderr, "flatpak-tools requires Linux and -out <directory>")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	if err := prepare(ctx, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func prepare(ctx context.Context, dir string) error {
	h, err := hugo.Download(ctx, nil)
	if err != nil {
		return fmt.Errorf("preparing Hugo: %w", err)
	}
	p, err := pagefind.Ensure(ctx)
	if err != nil {
		return fmt.Errorf("preparing Pagefind: %w", err)
	}
	for _, tool := range []struct{ src, name string }{{h.Path, "hugo"}, {p, "pagefind"}} {
		if err := copyTool(tool.src, filepath.Join(dir, tool.name)); err != nil {
			return fmt.Errorf("preparing %s: %w", tool.name, err)
		}
	}
	fmt.Printf("Prepared Hugo %s and Pagefind %s for the direct Flatpak bundle\n", h.Version, pagefind.Pinned)
	return nil
}

func copyTool(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.CreateTemp(filepath.Dir(dst), ".tool-*")
	if err != nil {
		return err
	}
	defer os.Remove(out.Name())
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Chmod(0o755); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(out.Name(), dst)
}
