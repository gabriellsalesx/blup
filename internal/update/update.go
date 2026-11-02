// Package update checks GitHub Releases for a newer Blup and replaces the
// running executable with it, keeping the previous one for rollback.
//
// Releases are made by GoReleaser (.goreleaser.yaml): one archive per
// OS/arch named blup_<version>_<os>_<arch>.tar.gz (.zip on Windows), a
// checksums.txt with their SHA-256 and checksums.txt.sig, an Ed25519
// signature over the exact bytes of checksums.txt made with the owner's
// release key (scripts/sign-release). An update needs all three: the
// signature must match a key built into this binary (keys.go), and only
// then is the archive checked against the signed checksums. The archive
// names carry the version, so an old signed release cannot be passed off
// as a new one, and a version that is not newer than the running one is
// refused.
package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
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
	// ErrSignature means checksums.txt.sig is missing or was not made with
	// a release key built into this binary.
	ErrSignature = errors.New("the release signature is missing or invalid")
	// ErrNotNewer means the release is not newer than the running version
	// (an attempt to go back to an older, possibly vulnerable, build).
	ErrNotNewer = errors.New("the release is not newer than this version")
)

const (
	sumsName = "checksums.txt"
	sigName  = "checksums.txt.sig"
	// maxDownload caps any file read from the network.
	maxDownload = 512 << 20
	maxSmall    = 1 << 20
)

// Release is a published version.
type Release struct {
	Version string // without the leading "v"
	Notes   string
	URL     string // release page
	assets  map[string]string
}

// Client talks to the GitHub API. Apply needs Current set.
type Client struct {
	HTTP    *http.Client
	BaseURL string // API base, default https://api.github.com (tests override)
	// Current is the running version; Apply refuses anything not newer.
	Current string
	// Keys overrides the built-in release keys (tests only).
	Keys []ed25519.PublicKey
}

func (c *Client) keys() []ed25519.PublicKey {
	if c.Keys != nil {
		return c.Keys
	}
	return PublicKeys()
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second, CheckRedirect: httpsOnly}
}

// httpsOnly refuses a redirect off HTTPS: GitHub sends the download to its
// file servers, always over HTTPS.
func httpsOnly(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != "https" {
		return fmt.Errorf("update: redirect to %s refused", req.URL.Scheme)
	}
	if len(via) >= 10 {
		return errors.New("update: too many redirects")
	}
	return nil
}

// secure tells whether a download address may be used: HTTPS only, unless
// a test points the client at its own server.
func (c *Client) secure(u string) bool {
	return c.BaseURL != "" || strings.HasPrefix(u, "https://")
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
// as <exe>.old for Rollback. Order: version check, signature on
// checksums.txt, then the archive's SHA-256 against the signed list.
func (c *Client) Apply(ctx context.Context, rel Release, exe string) error {
	if !Newer(rel.Version, c.Current) {
		return fmt.Errorf("%w: %s (running %s)", ErrNotNewer, rel.Version, c.Current)
	}
	name := AssetName(rel.Version)
	url, ok := rel.assets[name]
	if !ok {
		return ErrNoAsset
	}
	sumsURL, ok := rel.assets[sumsName]
	if !ok {
		return fmt.Errorf("%w: %s missing", ErrChecksum, sumsName)
	}
	sigURL, ok := rel.assets[sigName]
	if !ok {
		return fmt.Errorf("%w: %s missing", ErrSignature, sigName)
	}
	if !c.secure(url) || !c.secure(sumsURL) || !c.secure(sigURL) {
		return fmt.Errorf("%w: download not over HTTPS", ErrChecksum)
	}
	sums, err := c.get(ctx, sumsURL, maxSmall)
	if err != nil {
		return err
	}
	sig, err := c.get(ctx, sigURL, maxSmall)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSignature, err)
	}
	if err := VerifyChecksums(sums, sig, c.keys()); err != nil {
		return err
	}
	want, err := checksumFor(sums, name)
	if err != nil {
		return err
	}
	archive, err := c.get(ctx, url, maxDownload)
	if err != nil {
		return err
	}
	got := sha256.Sum256(archive)
	if hex.EncodeToString(got[:]) != want {
		return ErrChecksum
	}
	bin, err := extract(name, archive)
	if err != nil {
		return err
	}
	return replace(exe, bin)
}

// VerifyChecksums checks that sig (base64 of an Ed25519 signature, as
// written by scripts/sign-release) signs the exact bytes of sums with one
// of keys.
func VerifyChecksums(sums, sig []byte, keys []ed25519.PublicKey) error {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil || len(raw) != ed25519.SignatureSize {
		return fmt.Errorf("%w: malformed", ErrSignature)
	}
	for _, k := range keys {
		if len(k) == ed25519.PublicKeySize && ed25519.Verify(k, sums, raw) {
			return nil
		}
	}
	return ErrSignature
}

// checksumFor finds name's SHA-256 in a GoReleaser checksums file. A name
// listed twice, or a malformed hash, is refused.
func checksumFor(sums []byte, name string) (string, error) {
	want := ""
	sc := bufio.NewScanner(bytes.NewReader(sums))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) != 2 || f[1] != name {
			continue
		}
		if want != "" {
			return "", fmt.Errorf("%w: %s listed twice", ErrChecksum, name)
		}
		if b, err := hex.DecodeString(f[0]); err != nil || len(b) != sha256.Size {
			return "", ErrChecksum
		}
		want = strings.ToLower(f[0])
	}
	if want == "" {
		return "", fmt.Errorf("%w: %s not listed", ErrChecksum, name)
	}
	return want, nil
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

func (c *Client) get(ctx context.Context, url string, limit int64) ([]byte, error) {
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
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("GET %s: file too large", url)
	}
	return b, nil
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
