// Package pagefind downloads the Pagefind binary (pinned release, SHA-256
// checked) and builds the static search index of a generated site.
package pagefind

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Pinned is the Pagefind release used.
const Pinned = "1.5.2"

// ErrUnsupportedPlatform means Pagefind has no build for this OS/arch.
var ErrUnsupportedPlatform = errors.New("no Pagefind build for this platform")

// Ensure returns the cached binary, downloading it on first use.
func Ensure(ctx context.Context) (string, error) {
	dest, err := cachedPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}
	target, err := targetTriple()
	if err != nil {
		return "", err
	}
	asset := fmt.Sprintf("pagefind-v%s-%s.tar.gz", Pinned, target)
	base := "https://github.com/Pagefind/pagefind/releases/download/v" + Pinned + "/"
	sum, err := get(ctx, base+asset+".sha256")
	if err != nil {
		return "", err
	}
	want := strings.Fields(string(sum))
	if len(want) == 0 {
		return "", errors.New("empty Pagefind checksum")
	}
	archive, err := get(ctx, base+asset)
	if err != nil {
		return "", err
	}
	got := sha256.Sum256(archive)
	if hex.EncodeToString(got[:]) != want[0] {
		return "", errors.New("downloaded Pagefind failed the checksum verification")
	}
	exe, err := extract(archive)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dest+".part", exe, 0o755); err != nil {
		return "", err
	}
	return dest, os.Rename(dest+".part", dest)
}

// Index writes the search index of the site in dir to dir/pagefind.
func Index(ctx context.Context, bin, dir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, "--site", dir, "--output-subdir", "pagefind", "--quiet")
	return cmd.CombinedOutput()
}

func targetTriple() (string, error) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return "x86_64-unknown-linux-musl", nil
	case "linux/arm64":
		return "aarch64-unknown-linux-musl", nil
	case "darwin/amd64":
		return "x86_64-apple-darwin", nil
	case "darwin/arm64":
		return "aarch64-apple-darwin", nil
	case "windows/amd64":
		return "x86_64-pc-windows-msvc", nil
	case "windows/arm64":
		return "aarch64-pc-windows-msvc", nil
	case "freebsd/amd64":
		return "x86_64-unknown-freebsd", nil
	}
	return "", ErrUnsupportedPlatform
}

func cachedPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	name := "pagefind"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, "blup", "pagefind-"+Pinned, name), nil
}

func extract(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, errors.New("pagefind not found in archive")
		}
		if err != nil {
			return nil, err
		}
		if name := filepath.Base(h.Name); name == "pagefind" || name == "pagefind.exe" {
			return io.ReadAll(tr)
		}
	}
}

func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
