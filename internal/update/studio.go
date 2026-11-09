package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Blup Studio's own files in a release (release.yml and builds.yml): one
// per kind of install, listed with their SHA-256 in studio-checksums.txt,
// signed in studio-checksums.txt.sig with the same release key as the
// command line's checksums.txt.
const (
	studioSumsName = "studio-checksums.txt"
	studioSigName  = "studio-checksums.txt.sig"
	// maxStudio caps the Studio's download (the AppImage is about 100 MB).
	maxStudio = 1 << 30
)

// StudioKind is how this Blup Studio was installed, when it can update
// itself: the file it downloads and how it puts it in place.
type StudioKind string

const (
	// StudioAppImage: the AppImage file is replaced.
	StudioAppImage StudioKind = "appimage"
	// StudioWindows: the per-user installer runs silently over the install.
	StudioWindows StudioKind = "windows"
	// StudioMac: the .app from the disk image replaces the installed one.
	StudioMac StudioKind = "macos"
)

// StudioAsset is the file of a release for kind (version without "v").
func StudioAsset(kind StudioKind, version string) (string, error) {
	switch kind {
	case StudioAppImage:
		return fmt.Sprintf("blup-studio_%s_x86_64.AppImage", version), nil
	case StudioWindows:
		return fmt.Sprintf("blup-studio_%s_windows_x64-setup.exe", version), nil
	case StudioMac:
		return fmt.Sprintf("blup-studio_%s_macos_universal.dmg", version), nil
	}
	return "", ErrNoAsset
}

// DownloadStudio downloads the Blup Studio file of rel for kind into dir
// and returns its path. Same order as Apply: the version must be newer,
// studio-checksums.txt must carry a release key's signature, and the
// file, hashed while it is written, must match the signed list; anything
// else removes the partial file and fails. progress (optional) hears the
// bytes written and the total (0 when the server does not say).
func (c *Client) DownloadStudio(ctx context.Context, rel Release, kind StudioKind, dir string, progress func(done, total int64)) (string, error) {
	if !Newer(rel.Version, c.Current) {
		return "", fmt.Errorf("%w: %s (running %s)", ErrNotNewer, rel.Version, c.Current)
	}
	name, err := StudioAsset(kind, rel.Version)
	if err != nil {
		return "", err
	}
	url, ok := rel.assets[name]
	if !ok {
		return "", ErrNoAsset
	}
	sumsURL, ok := rel.assets[studioSumsName]
	if !ok {
		return "", fmt.Errorf("%w: %s missing", ErrChecksum, studioSumsName)
	}
	sigURL, ok := rel.assets[studioSigName]
	if !ok {
		return "", fmt.Errorf("%w: %s missing", ErrSignature, studioSigName)
	}
	if !c.secure(url) || !c.secure(sumsURL) || !c.secure(sigURL) {
		return "", fmt.Errorf("%w: download not over HTTPS", ErrChecksum)
	}
	sums, err := c.get(ctx, sumsURL, maxSmall)
	if err != nil {
		return "", err
	}
	sig, err := c.get(ctx, sigURL, maxSmall)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrSignature, err)
	}
	if err := VerifyChecksums(sums, sig, c.keys()); err != nil {
		return "", err
	}
	want, err := checksumFor(sums, name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("%w: %w", ErrNotWritable, err)
	}
	tmp, err := os.CreateTemp(dir, ".blup-studio-*.part")
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNotWritable, err)
	}
	keep := false
	defer func() {
		if !keep {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	got, err := c.stream(ctx, url, tmp, progress)
	if err != nil {
		return "", err
	}
	if got != want {
		return "", ErrChecksum
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, name)
	os.Remove(dest)
	if err := os.Rename(tmp.Name(), dest); err != nil {
		return "", fmt.Errorf("%w: %w", ErrNotWritable, err)
	}
	keep = true
	return dest, nil
}

// stream writes url's body to w, hashing it, and returns its SHA-256.
func (c *Client) stream(ctx context.Context, url string, w io.Writer, progress func(done, total int64)) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.downloader().Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading the update: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	total := max(resp.ContentLength, 0)
	h := sha256.New()
	var done int64
	buf := make([]byte, 256<<10)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			done += int64(n)
			if done > maxStudio {
				return "", fmt.Errorf("GET %s: file too large", url)
			}
			if _, err := w.Write(buf[:n]); err != nil {
				return "", err
			}
			h.Write(buf[:n])
			if progress != nil {
				progress(done, total)
			}
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			return "", fmt.Errorf("downloading the update: %w", rerr)
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
