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
	// Every read is a few KB: the window hears it at most ten times a
	// second, and the end.
	var last time.Time
	file, err := c.DownloadStudio(ctx, rel, kind, dir, func(done, total int64) {
		if done != total && time.Since(last) < 100*time.Millisecond {
			return
		}
		last = time.Now()
		application.Get().Event.Emit("studioUpdate", UpdateProgress{done, total})
	})
	if err != nil {
		return "", err
	}
	// Next to the AppImage, under a name of its own: the cleaning when the
	// Studio starts removes only this, never a version the person keeps
	// in the same folder.
	if kind == update.StudioAppImage {
		staged := filepath.Join(dir, stagedAppImage)
		if err := os.Rename(file, staged); err != nil {
			os.Remove(file)
			return "", fmt.Errorf("%w: %w", update.ErrNotWritable, err)
		}
		file = staged
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
	detach(restart, true)
	if err := restart.Start(); err != nil {
		// A job that refuses breaking away (Windows): start it inside.
		again := exec.Command(restart.Path, restart.Args[1:]...)
		again.Env = restart.Env
		detach(again, false)
		if err2 := again.Start(); err2 != nil {
			return fmt.Errorf("starting the new version: %w", err)
		}
	}
	downloaded.file = ""
	// As the tray's Quit: closing the window really closes the Studio.
	quitting.Store(true)
	go application.Get().Quit()
	return nil
}

// stagedAppImage is the new AppImage downloaded next to the running one.
const stagedAppImage = ".blup-studio-update.AppImage"

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
	appdir := ""
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "APPDIR="); ok && v != "" {
			appdir = filepath.Clean(v)
		}
	}
	out := env[:0:0]
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		// Its folders in PATH and XDG_DATA_DIRS: gone once it closes.
		if (k == "PATH" || k == "XDG_DATA_DIRS") && appdir != "" {
			var keep []string
			for _, d := range strings.Split(v, ":") {
				if d != appdir && !strings.HasPrefix(d, appdir+"/") {
					keep = append(keep, d)
				}
			}
			out = append(out, k+"="+strings.Join(keep, ":"))
			continue
		}
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
	// It waits as long as the Studio takes to close (the installer can't
	// replace a running exe), keeps the installer when it failed (the next
	// try reuses nothing, but nothing is lost either) and starts the Studio
	// in any case: the new version, or the old one if the install failed.
	script := fmt.Sprintf(`$ErrorActionPreference='SilentlyContinue'; `+
		`while (Get-Process -Id %d) { Start-Sleep -Milliseconds 200 }; `+
		`$p = Start-Process -FilePath %s -ArgumentList %s -Wait -PassThru; `+
		`if ($p -and $p.ExitCode -eq 0) { Remove-Item -LiteralPath %s }; Start-Process -FilePath %s`,
		os.Getpid(), q(setup), q("/S /D="+dir), q(setup), q(exe))
	return exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-Command", script), nil
}

// cleanUpdateLeftovers removes what an update left behind: a new version
// downloaded and never installed (the AppImage next to this one, the
// installer or disk image in the cache) and, on macOS, the old .app the
// swap kept until the Studio closed. Called when the Studio starts.
func cleanUpdateLeftovers() {
	if cache, err := os.UserCacheDir(); err == nil {
		os.RemoveAll(filepath.Join(cache, "blup", "studio-update"))
	}
	kind, target := app.StudioSelfUpdate()
	dir, name := filepath.Dir(target), filepath.Base(target)
	switch kind {
	case update.StudioAppImage:
		parts, _ := filepath.Glob(filepath.Join(dir, ".blup-studio-*.part"))
		for _, f := range append(parts, filepath.Join(dir, stagedAppImage)) {
			if f != target {
				os.Remove(f)
			}
		}
	case update.StudioMac:
		os.RemoveAll(filepath.Join(dir, "."+name+".old"))
		os.RemoveAll(filepath.Join(dir, "."+name+".new"))
	}
}
