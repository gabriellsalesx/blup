package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gabriellsalesx/blup/internal/sandbox"
)

// launchPath is the file that starts the Studio, for what runs it later:
// Hyprland's show-and-hide binding and the "open with the computer" entry.
//
// It is not always this program. An AppImage runs from a temporary mount
// (/tmp/.mount_blup-XXXXXX) that changes every time it starts and is gone
// once the Studio closes: a binding or an autostart entry pointing there
// runs nothing. The AppImage file itself is in APPIMAGE.
func launchPath() string {
	return launchPathFrom(runtime.GOOS, os.Executable, os.Getenv, fileExists)
}

func launchPathFrom(goos string, executable func() (string, error), env func(string) string, exists func(string) bool) string {
	exe, _ := executable()
	if exe != "" {
		if p, err := filepath.EvalSymlinks(exe); err == nil {
			exe = p
		}
	}
	if goos != "linux" {
		return exe
	}
	// Only when this program really runs from the AppImage's mount: APPIMAGE
	// passes on to every program an AppImage starts, so it alone says
	// nothing about this one.
	appimage, appdir := env("APPIMAGE"), env("APPDIR")
	if appimage == "" || appdir == "" || exe == "" || !filepath.IsAbs(appimage) {
		return exe
	}
	if !strings.HasPrefix(exe, strings.TrimSuffix(appdir, "/")+"/") || !exists(appimage) {
		return exe
	}
	return appimage
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// enableAutostart makes the Studio open (in the tray) at login. The Wails
// entry points at this program, which in an AppImage is the temporary
// mount: there the Studio writes the same .desktop file itself, pointing at
// the AppImage.
func enableAutostart() error {
	exe := launchPath()
	if f := autostartPath(); runtime.GOOS == "linux" && os.Getenv("APPIMAGE") == exe && exe != "" && f != "" {
		return writeAutostart(f, exe)
	}
	if desk.app == nil {
		return errors.New("not available")
	}
	return desk.app.Autostart.EnableWithOptions(autostartOptions())
}

// autostartPath is where the Studio's .desktop autostart entry goes on
// Linux ("" elsewhere), whether or not it exists.
func autostartPath() string {
	if runtime.GOOS != "linux" || sandbox.Flatpak() {
		return ""
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", autostartID+".desktop")
}

func writeAutostart(file, exe string) error {
	entry, err := autostartEntry(exe)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, []byte(entry), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// autostartEntry is the .desktop entry that starts exe hidden in the tray,
// in the same format Wails writes.
func autostartEntry(exe string) (string, error) {
	for _, r := range exe {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("control character %U in the path", r)
		}
	}
	return "[Desktop Entry]\nType=Application\nName=Blup Studio\nExec=" + autostartExec(exe) +
		"\nX-GNOME-Autostart-enabled=true\nHidden=false\nNoDisplay=false\nTerminal=false\n", nil
}

// autostartExec is the Exec value of the entry for exe.
func autostartExec(exe string) string {
	return desktopExecQuote(exe) + " " + hiddenArg
}

// desktopExecQuote quotes one argument of a .desktop Exec line (the
// freedesktop.org rules: inside double quotes, " ` $ and \ take a
// backslash).
func desktopExecQuote(s string) string {
	if !strings.ContainsAny(s, " \t\"`$\\'><~|&;*?#()") {
		return s
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`)
	return `"` + r.Replace(s) + `"`
}
