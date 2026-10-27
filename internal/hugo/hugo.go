// Package hugo finds a usable Hugo binary — downloading the pinned release
// when needed — and runs it.
package hugo

import (
	"archive/tar"
	"archive/zip"
	"bufio"
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
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Pinned is the release downloaded when no suitable Hugo is installed.
const Pinned = "0.169.0"

// Minimum is the oldest Hugo the theme supports.
const Minimum = "0.160.0"

// ErrUnsupportedPlatform means there is no archive we can unpack for this
// OS/arch (macOS only ships .pkg installers).
var ErrUnsupportedPlatform = errors.New("no Hugo download for this platform")

// Binary is a Hugo executable and its version.
type Binary struct {
	Path    string
	Version string
}

// Find returns an installed Hugo that is new enough: the one in PATH or the
// one previously downloaded by us.
func Find(ctx context.Context) (Binary, bool) {
	candidates := []string{}
	if p, err := exec.LookPath("hugo"); err == nil {
		candidates = append(candidates, p)
	}
	if p, err := cachedPath(); err == nil {
		candidates = append(candidates, p)
	}
	for _, p := range candidates {
		v, err := version(ctx, p)
		if err == nil && compare(v, Minimum) >= 0 {
			return Binary{Path: p, Version: v}, true
		}
	}
	return Binary{}, false
}

// Download fetches the pinned release, checks its SHA-256 against the
// official checksums file and unpacks the binary into the cache folder.
func Download(ctx context.Context, progress func(done, total int64)) (Binary, error) {
	asset, err := assetName()
	if err != nil {
		return Binary{}, err
	}
	base := "https://github.com/gohugoio/hugo/releases/download/v" + Pinned + "/"
	sums, err := get(ctx, base+"hugo_"+Pinned+"_checksums.txt", nil)
	if err != nil {
		return Binary{}, err
	}
	want := ""
	sc := bufio.NewScanner(bytes.NewReader(sums))
	for sc.Scan() {
		if f := strings.Fields(sc.Text()); len(f) == 2 && f[1] == asset {
			want = f[0]
		}
	}
	if want == "" {
		return Binary{}, fmt.Errorf("checksum for %s not found", asset)
	}
	archive, err := get(ctx, base+asset, progress)
	if err != nil {
		return Binary{}, err
	}
	sum := sha256.Sum256(archive)
	if hex.EncodeToString(sum[:]) != want {
		return Binary{}, errors.New("downloaded Hugo failed the checksum verification")
	}
	exe, err := extract(asset, archive)
	if err != nil {
		return Binary{}, err
	}
	dest, err := cachedPath()
	if err != nil {
		return Binary{}, err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return Binary{}, err
	}
	tmp := dest + ".part"
	if err := os.WriteFile(tmp, exe, 0o755); err != nil {
		return Binary{}, err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return Binary{}, err
	}
	return Binary{Path: dest, Version: Pinned}, nil
}

// Command builds an exec.Cmd running Hugo inside dir.
func (b Binary) Command(ctx context.Context, dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, b.Path, args...)
	cmd.Dir = dir
	return cmd
}

func cachedPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	name := "hugo"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, "blup", "hugo-"+Pinned, name), nil
}

func assetName() (string, error) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64", "linux/arm64", "windows/amd64", "windows/arm64", "freebsd/amd64":
	default:
		return "", ErrUnsupportedPlatform
	}
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("hugo_%s_%s-%s%s", Pinned, runtime.GOOS, runtime.GOARCH, ext), nil
}

func extract(asset string, data []byte) ([]byte, error) {
	if strings.HasSuffix(asset, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if f.Name == "hugo.exe" {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, errors.New("hugo.exe not found in archive")
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, errors.New("hugo not found in archive")
		}
		if err != nil {
			return nil, err
		}
		if h.Name == "hugo" {
			return io.ReadAll(tr)
		}
	}
}

func get(ctx context.Context, url string, progress func(done, total int64)) ([]byte, error) {
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
	if progress == nil {
		return io.ReadAll(resp.Body)
	}
	var buf bytes.Buffer
	chunk := make([]byte, 64<<10)
	for {
		n, err := resp.Body.Read(chunk)
		buf.Write(chunk[:n])
		progress(int64(buf.Len()), resp.ContentLength)
		if err == io.EOF {
			return buf.Bytes(), nil
		}
		if err != nil {
			return nil, err
		}
	}
}

var versionRE = regexp.MustCompile(`v(\d+\.\d+\.\d+)`)

func version(ctx context.Context, path string) (string, error) {
	out, err := exec.CommandContext(ctx, path, "version").Output()
	if err != nil {
		return "", err
	}
	m := versionRE.FindSubmatch(out)
	if m == nil {
		return "", errors.New("unknown hugo version output")
	}
	return string(m[1]), nil
}

// compare returns -1, 0 or 1 comparing two x.y.z versions.
func compare(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := range 3 {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
