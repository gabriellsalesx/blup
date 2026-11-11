// Package sandbox identifies the restrictions of the installed application.
package sandbox

import (
	"os"
	"runtime"
)

// Flatpak reports whether this process is running inside a Flatpak sandbox.
func Flatpak() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if os.Getenv("FLATPAK_ID") != "" {
		return true
	}
	_, err := os.Stat("/.flatpak-info")
	return err == nil
}
