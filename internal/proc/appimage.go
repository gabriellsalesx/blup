package proc

import (
	"os"
	"path/filepath"
	"strings"
)

// Inside an AppImage, AppRun points the Studio at the libraries, GStreamer
// plugins and WebKit files bundled with it (LD_LIBRARY_PATH and friends),
// and every program the Studio starts inherits that. The system's programs
// (the shell of the built-in terminal, the browser a link opens, xdg-open,
// hyprctl, Hugo) then load the AppImage's libraries instead of their own:
// they crash, or fail with a missing symbol. They get the system's
// environment instead.

// Environ is os.Environ() without what an AppImage set for itself. Outside
// an AppImage (no APPDIR) it is os.Environ() unchanged.
func Environ() []string {
	return OutsideAppImage(os.Environ())
}

// OutsideAppImage drops from env what the AppImage named by its APPDIR set
// for itself: its folders in PATH, XDG_DATA_DIRS and LD_LIBRARY_PATH, and
// the variables only its AppRun sets. Without APPDIR, env is returned as
// it is.
func OutsideAppImage(env []string) []string {
	appdir := ""
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "APPDIR="); ok && v != "" {
			appdir = filepath.Clean(v)
		}
	}
	if appdir == "" || appdir == "/" {
		return env
	}
	out := env[:0:0]
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		switch {
		case k == "PATH" || k == "XDG_DATA_DIRS" || k == "LD_LIBRARY_PATH":
			var keep []string
			for _, d := range strings.Split(v, ":") {
				if d != "" && d != appdir && !strings.HasPrefix(d, appdir+"/") {
					keep = append(keep, d)
				}
			}
			if len(keep) == 0 && k == "LD_LIBRARY_PATH" {
				continue
			}
			out = append(out, k+"="+strings.Join(keep, ":"))
		case k == "WEBKIT_INJECTED_BUNDLE_PATH", k == "GSETTINGS_SCHEMA_DIR", k == "GIO_EXTRA_MODULES",
			k == "GDK_PIXBUF_MODULE_FILE", k == "GTK_PATH", k == "GTK_DATA_PREFIX", k == "GTK_EXE_PREFIX",
			k == "APPIMAGE", k == "APPDIR", k == "ARGV0", k == "OWD", strings.HasPrefix(k, "GST_"):
		default:
			out = append(out, kv)
		}
	}
	return out
}
