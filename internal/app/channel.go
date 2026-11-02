package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Channel is set by builds made for a store
// (-ldflags "-X github.com/gabriellsalesx/blup/internal/app.Channel=msstore"):
// the store updates them.
var Channel = ""

// UpdatesBy tells who brings new versions of this Blup: "" when Blup tells
// about them itself (the binary, AppImage and installers from GitHub),
// "msstore" (Microsoft Store), "flatpak" or "package" (a system package,
// like the AUR's). In those the Studio doesn't look for new versions.
func UpdatesBy() string {
	exe, _ := os.Executable()
	if exe != "" {
		if p, err := filepath.EvalSymlinks(exe); err == nil {
			exe = p
		}
	}
	return updatesBy(runtime.GOOS, exe, os.Getenv)
}

func updatesBy(goos, exe string, env func(string) string) string {
	if Channel != "" {
		return Channel
	}
	switch goos {
	case "windows":
		// A packaged app (MSIX) runs from Program Files\WindowsApps.
		if strings.Contains(strings.ToLower(exe), `\windowsapps\`) {
			return "msstore"
		}
	case "linux":
		if env("FLATPAK_ID") != "" {
			return "flatpak"
		}
		if _, err := os.Stat("/.flatpak-info"); err == nil {
			return "flatpak"
		}
		// An AppImage runs from a temporary mount under /tmp; APPIMAGE
		// names the file the person runs.
		if env("APPIMAGE") != "" {
			return ""
		}
		// Only the package manager writes to /usr (/usr/local is the
		// person's own).
		if strings.HasPrefix(exe, "/usr/") && !strings.HasPrefix(exe, "/usr/local/") {
			return "package"
		}
	}
	return ""
}
