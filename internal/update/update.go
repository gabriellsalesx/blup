// Package update checks GitHub Releases for a newer Blup and replaces the
// running executable with it, keeping the previous one for rollback.
//
// Releases are made by GoReleaser (.goreleaser.yaml): one archive per
// OS/arch named blup_<version>_<os>_<arch>.tar.gz (.zip on Windows) and a
// checksums.txt with their SHA-256.
package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Repo is the GitHub repository that publishes the releases.
const Repo = "gabriellsalesx/blup"

var (
	// ErrNoAsset means the release has no build for this OS/arch.
	ErrNoAsset = errors.New("no download for this system in the release")
	// ErrChecksum means the download did not match checksums.txt.
	ErrChecksum = errors.New("downloaded file failed the checksum verification")
	// ErrNotWritable means the executable's folder cannot be written.
	ErrNotWritable = errors.New("cannot replace the blup executable here")
	// ErrNoBackup means there is no previous version to go back to.
	ErrNoBackup = errors.New("no previous version to restore")
)

// Release is a published version.
type Release struct {
	Version string // without the leading "v"
	Notes   string
	URL     string // release page
	assets  map[string]string
}

// Client talks to the GitHub API. Zero value works.
type Client struct {
	HTTP    *http.Client
	BaseURL string // API base, default https://api.github.com (tests override)
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (c *Client) base() string {
	if c.BaseURL != "" {
		return strings.TrimSuffix(c.BaseURL, "/")
	}
	return "https://api.github.com"
}

// Latest returns the newest published release.
func (c *Client) Latest(ctx context.Context) (Release, error) {
	var r struct {
		Tag     string `json:"tag_name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+"/repos/"+Repo+"/releases/latest", nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.http().Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("checking for updates: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return Release{}, err
	}
	rel := Release{Version: strings.TrimPrefix(r.Tag, "v"), Notes: strings.TrimSpace(r.Body), URL: r.HTMLURL, assets: map[string]string{}}
	for _, a := range r.Assets {
		rel.assets[a.Name] = a.URL
	}
	return rel, nil
}

// Newer reports whether version a is newer than b (x.y.z, optional "v").
// Development builds ("dev") are never compared as older.
func Newer(a, b string) bool {
	pa, oka := parse(a)
	pb, okb := parse(b)
	if !oka || !okb {
		return false
	}
	for i := range 3 {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// AssetName is the archive for this system in a release.
func AssetName(version string) string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("blup_%s_%s_%s%s", version, runtime.GOOS, runtime.GOARCH, ext)
}

// Apply downloads rel for this system, verifies it and replaces the
// executable at exe (usually os.Executable()). The old one stays next to it
// as <exe>.old for Rollback.
func (c *Client) Apply(ctx context.Context, rel Release, exe string) error {
	name := AssetName(rel.Version)
	url, ok := rel.assets[name]
	if !ok {
		return ErrNoAsset
	}
	sumsURL, ok := rel.assets["checksums.txt"]
	if !ok {
		return fmt.Errorf("%w: checksums.txt missing", ErrChecksum)
	}
	sums, err := c.get(ctx, sumsURL)
	if err != nil {
		return err
	}
	want := ""
	sc := bufio.NewScanner(bytes.NewReader(sums))
	for sc.Scan() {
		if f := strings.Fields(sc.Text()); len(f) == 2 && f[1] == name {
			want = f[0]
		}
	}
	archive, err := c.get(ctx, url)
	if err != nil {
		return err
	}
	got := sha256.Sum256(archive)
	if want == "" || hex.EncodeToString(got[:]) != want {
		return ErrChecksum
	}
	bin, err := extract(name, archive)
	if err != nil {
		return err
	}
	return replace(exe, bin)
}

// Rollback restores the executable saved by the last Apply.
func Rollback(exe string) error {
	old := exe + ".old"
	if _, err := os.Stat(old); err != nil {
		return ErrNoBackup
	}
	b, err := os.ReadFile(old)
	if err != nil {
		return err
	}
	return replace(exe, b)
}

// replace swaps exe for bin: write bin next to exe, move exe to exe.old,
// move the new file into place. Renames work even for a running program
// (also on Windows), and a failure leaves the original in place.
func replace(exe string, bin []byte) error {
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".blup-new-*")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNotWritable, err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return fmt.Errorf("%w: %v", ErrNotWritable, err)
	}
	if err := os.Rename(tmp.Name(), exe); err != nil {
		os.Rename(old, exe) // put the original back
		return fmt.Errorf("%w: %v", ErrNotWritable, err)
	}
	return nil
}

func extract(name string, data []byte) ([]byte, error) {
	want := "blup"
	if runtime.GOOS == "windows" {
		want = "blup.exe"
	}
	if strings.HasSuffix(name, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if filepath.Base(f.Name) == want {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, fmt.Errorf("%s not found in %s", want, name)
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%s not found in %s", want, name)
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(h.Name) == want && h.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// Cache remembers the last check so the network is asked at most once a day.
type Cache struct {
	Checked time.Time `json:"checked"`
	Latest  string    `json:"latest"`
	Notes   string    `json:"notes"`
	URL     string    `json:"url"`
	Skipped string    `json:"skipped"` // version the user chose not to hear about
}

func cachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "blup", "update.json"), nil
}

// LoadCache reads the cache; a missing file is an empty cache.
func LoadCache() Cache {
	var c Cache
	p, err := cachePath()
	if err != nil {
		return c
	}
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	return c
}

// Save writes the cache.
func (c Cache) Save() error {
	p, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}
