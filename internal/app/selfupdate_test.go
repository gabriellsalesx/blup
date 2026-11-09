package app

import (
	"testing"

	"github.com/gabriellsalesx/blup/internal/update"
)

func TestStudioSelfUpdate(t *testing.T) {
	old := Version
	Version = "1.0.5"
	t.Cleanup(func() { Version = old })
	env := func(vars map[string]string) func(string) string { return func(k string) string { return vars[k] } }
	cases := []struct {
		name, goos, goarch, exe, by string
		vars                        map[string]string
		kind                        update.StudioKind
		target                      string
	}{
		{"appimage", "linux", "amd64", "/tmp/.mount_x/usr/bin/blup-studio", "", map[string]string{"APPIMAGE": "/home/a/bin/blup-studio.AppImage", "APPDIR": "/tmp/.mount_x"}, update.StudioAppImage, "/home/a/bin/blup-studio.AppImage"},
		{"appimage on arm64", "linux", "arm64", "/tmp/.mount_x/usr/bin/blup-studio", "", map[string]string{"APPIMAGE": "/home/a/blup.AppImage"}, "", ""},
		{"linux tarball", "linux", "amd64", "/home/a/blup-studio", "", nil, "", ""},
		{"started from another AppImage", "linux", "amd64", "/home/a/blup-studio", "", map[string]string{"APPIMAGE": "/home/a/Terminal.AppImage", "APPDIR": "/tmp/.mount_term"}, "", ""},
		{"deb", "linux", "amd64", "/usr/bin/blup-studio", "package", nil, "", ""},
		{"flatpak", "linux", "amd64", "/app/bin/blup-studio", "flatpak", map[string]string{"APPIMAGE": "/x.AppImage"}, "", ""},
		{"macos", "darwin", "arm64", "/Applications/Blup Studio.app/Contents/MacOS/blup-studio", "", nil, update.StudioMac, "/Applications/Blup Studio.app"},
		{"macos loose binary", "darwin", "arm64", "/Users/a/bin/blup-studio", "", nil, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, target := studioSelfUpdate(c.goos, c.goarch, c.exe, c.by, env(c.vars))
			if kind != c.kind || target != c.target {
				t.Fatalf("got %q %q, want %q %q", kind, target, c.kind, c.target)
			}
		})
	}
	Version = "dev"
	if kind, _ := studioSelfUpdate("linux", "amd64", "/x", "", env(map[string]string{"APPIMAGE": "/x.AppImage"})); kind != "" {
		t.Fatal("a development build never updates itself")
	}
}
