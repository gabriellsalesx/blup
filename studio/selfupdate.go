package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/update"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The Studio updating itself, like the local AIs' downloads: Settings >
// About downloads the new version (DownloadUpdate, with "studioUpdate"
// progress events), checks it against the release's signed list
// (internal/update) and, on "Restart and update" (InstallUpdate), puts it
// in place and starts it once this window has closed (only one Studio
// runs at a time). Installs that can't (stores, system packages) keep
// sending the person to the site.

// UpdateProgress is how much of the new version came.
type UpdateProgress struct {
	Done  int64 `json:"done"`
	Total int64 `json:"total"`
}

func init() {
	application.RegisterEvent[UpdateProgress]("studioUpdate")
}

// errNoSelfUpdate: this install updates through the site, a store or the
// system's packages.
var errNoSelfUpdate = errors.New("this Blup Studio is updated from the download page")

// downloaded is the new version waiting for InstallUpdate.
var downloaded struct {
	sync.Mutex
	file, version string
	kind          update.StudioKind
	target        string
}

// DownloadUpdate downloads and checks the newest Blup Studio for this
// install and returns its version. It doesn't install anything: that is
// InstallUpdate, when the person chooses to restart.
func (s *Studio) DownloadUpdate(ctx context.Context) (string, error) {
	kind, target := app.StudioSelfUpdate()
	if kind == "" {
		return "", errNoSelfUpdate
	}
	c := &update.Client{Current: app.Version}
	lctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	rel, err := c.Latest(lctx)
	cancel()
	if err != nil {
		return "", err
	}
	// The AppImage is downloaded next to the one it replaces (same disk: the
	// swap is a rename, and a folder that can't be written fails here, before
	// the download); the others into the cache.
	dir := filepath.Dir(target)
	if kind != update.StudioAppImage {
		cache, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(cache, "blup", "studio-update")
	}
	file, err := c.DownloadStudio(ctx, rel, kind, dir, func(done, total int64) {
		application.Get().Event.Emit("studioUpdate", UpdateProgress{done, total})
	})
	if err != nil {
		return "", err
	}
	downloaded.Lock()
	if downloaded.file != "" && downloaded.file != file {
		os.Remove(downloaded.file)
	}
	downloaded.file, downloaded.version, downloaded.kind, downloaded.target = file, rel.Version, kind, target
	downloaded.Unlock()
	return rel.Version, nil
}

// InstallUpdate puts the downloaded version in place, starts it once this
// Studio has closed and closes it.
func (s *Studio) InstallUpdate() error {
	downloaded.Lock()
	defer downloaded.Unlock()
	if downloaded.file == "" {
		return errors.New("no update downloaded")
	}
	var restart *exec.Cmd
	var err error
	switch downloaded.kind {
	case update.StudioAppImage:
		restart, err = installAppImage(downloaded.file, downloaded.target)
	case update.StudioMac:
		restart, err = installMac(downloaded.file, downloaded.target)
	case update.StudioWindows:
		restart, err = installWindows(downloaded.file, downloaded.target)
	default:
		err = errNoSelfUpdate
	}
	if err != nil {
		return err
	}
	detach(restart)
	if err := restart.Start(); err != nil {
		return fmt.Errorf("starting the new version: %w", err)
	}
	downloaded.file = ""
	go application.Get().Quit()
	return nil
}

// waitThen is a shell line that waits for this Studio to close, then runs
// what follows ("$1" is this process).
const waitThen = `while kill -0 "$1" 2>/dev/null; do sleep 0.2; done; `

// installAppImage swaps the AppImage file (a rename: the running Studio
// keeps the old one open until it closes) and starts the new one.
func installAppImage(file, target string) (*exec.Cmd, error) {
	if err := os.Chmod(file, 0o755); err != nil {
		return nil, err
	}
	if err := os.Rename(file, target); err != nil {
		return nil, fmt.Errorf("%w: %w", update.ErrNotWritable, err)
	}
	cmd := exec.Command("/bin/sh", "-c", waitThen+`exec "$2"`, "blup-update", strconv.Itoa(os.Getpid()), target)
	cmd.Env = outsideAppImage(os.Environ())
	return cmd, nil
}

// outsideAppImage drops what this AppImage's AppRun set for itself (its
// libraries, GStreamer and WebKit paths): the new one sets its own.
func outsideAppImage(env []string) []string {
	out := env[:0:0]
	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		switch {
		case k == "LD_LIBRARY_PATH", k == "WEBKIT_INJECTED_BUNDLE_PATH", k == "GSETTINGS_SCHEMA_DIR",
			k == "APPIMAGE", k == "APPDIR", k == "ARGV0", k == "OWD", strings.HasPrefix(k, "GST_"):
			continue
		}
		out = append(out, kv)
	}
	return out
}

// installMac copies the .app from the disk image next to the installed
// one, swaps them (the running Studio keeps its files open) and starts the
// new one; the old bundle is removed once this Studio has closed.
func installMac(dmg, bundle string) (*exec.Cmd, error) {
	mnt, err := os.MkdirTemp("", "blup-update-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(mnt)
	if out, err := exec.Command("hdiutil", "attach", "-nobrowse", "-readonly", "-noautoopen", "-mountpoint", mnt, dmg).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("opening the update: %w: %s", err, out)
	}
	defer func() {
		if exec.Command("hdiutil", "detach", mnt).Run() != nil {
			_ = exec.Command("hdiutil", "detach", "-force", mnt).Run()
		}
	}()
	apps, _ := filepath.Glob(filepath.Join(mnt, "*.app"))
	if len(apps) != 1 {
		return nil, errors.New("the update has no app inside")
	}
	dir, name := filepath.Dir(bundle), filepath.Base(bundle)
	staged := filepath.Join(dir, "."+name+".new")
	old := filepath.Join(dir, "."+name+".old")
	os.RemoveAll(staged)
	os.RemoveAll(old)
	if out, err := exec.Command("ditto", apps[0], staged).CombinedOutput(); err != nil {
		os.RemoveAll(staged)
		return nil, fmt.Errorf("%w: %v: %s", update.ErrNotWritable, err, out)
	}
	if err := os.Rename(bundle, old); err != nil {
		os.RemoveAll(staged)
		return nil, fmt.Errorf("%w: %w", update.ErrNotWritable, err)
	}
	if err := os.Rename(staged, bundle); err != nil {
		_ = os.Rename(old, bundle)
		os.RemoveAll(staged)
		return nil, fmt.Errorf("%w: %w", update.ErrNotWritable, err)
	}
	// Downloaded by the Studio itself, the files carry no quarantine; this
	// only makes sure.
	_ = exec.Command("xattr", "-dr", "com.apple.quarantine", bundle).Run()
	os.Remove(dmg)
	return exec.Command("/bin/sh", "-c", waitThen+`rm -rf "$3"; open "$2"`, "blup-update", strconv.Itoa(os.Getpid()), bundle, old), nil
}

// installWindows runs the per-user installer silently into the same
// folder once this Studio has closed, then starts it again.
func installWindows(setup, dir string) (*exec.Cmd, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	// NSIS: /S is silent, /D= the folder (last, unquoted, spaces allowed).
	script := fmt.Sprintf(`$ErrorActionPreference='SilentlyContinue'; Wait-Process -Id %d -Timeout 60; `+
		`Start-Process -FilePath %s -ArgumentList %s -Wait; Remove-Item -LiteralPath %s; Start-Process -FilePath %s`,
		os.Getpid(), q(setup), q("/S /D="+dir), q(setup), q(exe))
	return exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-Command", script), nil
}
