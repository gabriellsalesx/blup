package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gabriellsalesx/blup/internal/update"
)

// StudioSelfUpdate tells how this Blup Studio can update itself (download
// the new version and put it in place) and what it replaces: the AppImage
// file, the per-user install folder on Windows, or the .app bundle on
// macOS. Kind "" means it can't: a store or the system's packages update
// it (UpdatesBy), or it runs from somewhere the published files don't
// cover (an arm64 Linux, a tar.gz copy, a build of one's own). Then the
// Studio sends the person to the site's download page.
func StudioSelfUpdate() (update.StudioKind, string) {
	exe, _ := os.Executable()
	if exe != "" {
		if p, err := filepath.EvalSymlinks(exe); err == nil {
			exe = p
		}
	}
	return studioSelfUpdate(runtime.GOOS, runtime.GOARCH, exe, UpdatesBy(), os.Getenv)
}

func studioSelfUpdate(goos, goarch, exe, updatesBy string, env func(string) string) (update.StudioKind, string) {
	if updatesBy != "" || Version == "dev" || exe == "" {
		return "", ""
	}
	switch goos {
	case "linux":
		// Only the x86_64 AppImage is published; APPIMAGE names its file.
		if file := env("APPIMAGE"); file != "" && goarch == "amd64" && filepath.IsAbs(file) {
			return update.StudioAppImage, file
		}
	case "windows":
		// The installer is per user and x64: it lives in
		// %LOCALAPPDATA%\Programs\<name>, and runs there again.
		local := env("LOCALAPPDATA")
		dir := filepath.Dir(exe)
		if goarch == "amd64" && local != "" && strings.EqualFold(filepath.Dir(dir), filepath.Join(local, "Programs")) {
			return update.StudioWindows, dir
		}
	case "darwin":
		// <bundle>.app/Contents/MacOS/<exe>: the bundle is what changes.
		macos := filepath.Dir(exe)
		contents := filepath.Dir(macos)
		bundle := filepath.Dir(contents)
		if filepath.Base(macos) == "MacOS" && filepath.Base(contents) == "Contents" && strings.HasSuffix(bundle, ".app") {
			return update.StudioMac, bundle
		}
	}
	return "", ""
}
