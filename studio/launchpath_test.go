package main

import (
	"errors"
	"strings"
	"testing"
)

func TestLaunchPath(t *testing.T) {
	exe := func(p string) func() (string, error) { return func() (string, error) { return p, nil } }
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	yes := func(string) bool { return true }
	no := func(string) bool { return false }
	app := map[string]string{"APPIMAGE": "/home/u/Apps/blup-studio.AppImage", "APPDIR": "/tmp/.mount_blup-abc"}
	inMount := "/tmp/.mount_blup-abc/usr/bin/blup-studio"

	cases := []struct {
		name, goos, exe string
		env             map[string]string
		exists          func(string) bool
		want            string
	}{
		{"appimage", "linux", inMount, app, yes, app["APPIMAGE"]},
		{"appimage file gone", "linux", inMount, app, no, inMount},
		{"inherited APPIMAGE, other program", "linux", "/usr/bin/blup-studio", app, yes, "/usr/bin/blup-studio"},
		{"mount prefix only", "linux", "/tmp/.mount_blup-abcd/usr/bin/blup-studio", app, yes, "/tmp/.mount_blup-abcd/usr/bin/blup-studio"},
		{"relative APPIMAGE", "linux", inMount, map[string]string{"APPIMAGE": "blup.AppImage", "APPDIR": "/tmp/.mount_blup-abc"}, yes, inMount},
		{"no AppImage", "linux", "/usr/bin/blup-studio", nil, yes, "/usr/bin/blup-studio"},
		{"not linux", "darwin", "/Applications/Blup Studio.app/Contents/MacOS/blup-studio", app, yes, "/Applications/Blup Studio.app/Contents/MacOS/blup-studio"},
	}
	for _, c := range cases {
		if got := launchPathFrom(c.goos, exe(c.exe), env(c.env), c.exists); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
	if got := launchPathFrom("linux", func() (string, error) { return "", errors.New("x") }, env(app), yes); got != "" {
		t.Errorf("no executable: got %q", got)
	}
}

func TestAutostartEntry(t *testing.T) {
	e, err := autostartEntry("/home/u/My Apps/blup$1.AppImage")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e, "\nExec=\"/home/u/My Apps/blup\\$1.AppImage\" --hidden\n") {
		t.Fatalf("entry: %s", e)
	}
	if !strings.Contains(e, "\nExec="+autostartExec("/home/u/My Apps/blup$1.AppImage")+"\n") {
		t.Fatal("autostartExec differs from the entry")
	}
	if e, _ := autostartEntry("/usr/bin/blup-studio"); !strings.Contains(e, "\nExec=/usr/bin/blup-studio --hidden\n") {
		t.Fatalf("plain: %s", e)
	}
	if _, err := autostartEntry("/x\n[Desktop Entry]\nExec=evil"); err == nil {
		t.Fatal("a line break in the path was accepted")
	}
}
