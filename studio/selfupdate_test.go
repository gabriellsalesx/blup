package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestInstallAppImageSwapsTheFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "blup-studio.AppImage")
	file := filepath.Join(dir, "blup-studio_1.0.5_x86_64.AppImage")
	os.WriteFile(target, []byte("old"), 0o755)
	os.WriteFile(file, []byte("new"), 0o644)
	cmd, err := installAppImage(file, target)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target); string(b) != "new" {
		t.Fatal("the AppImage was not replaced")
	}
	if fi, _ := os.Stat(target); fi.Mode().Perm()&0o100 == 0 {
		t.Fatal("the new AppImage must be executable")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("the downloaded file moves, it isn't copied")
	}
	if cmd.Args[len(cmd.Args)-1] != target || !strings.Contains(cmd.Args[2], "kill -0") {
		t.Fatalf("restart command %v", cmd.Args)
	}
}

func TestOutsideAppImage(t *testing.T) {
	env := outsideAppImage([]string{"HOME=/home/a", "APPDIR=/tmp/.mount_x", "LD_LIBRARY_PATH=/tmp/.mount_x/usr/lib", "GST_PLUGIN_PATH_1_0=/x", "APPIMAGE=/a.AppImage", "WEBKIT_INJECTED_BUNDLE_PATH=/y", "PATH=/tmp/.mount_x/usr/bin:/usr/bin", "XDG_DATA_DIRS=/tmp/.mount_x/usr/share:/usr/share"})
	if !slices.Equal(env, []string{"HOME=/home/a", "PATH=/usr/bin", "XDG_DATA_DIRS=/usr/share"}) {
		t.Fatalf("got %v", env)
	}
}

func TestInstallWindowsScript(t *testing.T) {
	cmd, err := installWindows(`C:\Users\a\AppData\Local\blup\studio-update\setup's.exe`, `C:\Users\a\AppData\Local\Programs\Blup Studio`)
	if err != nil {
		t.Fatal(err)
	}
	script := cmd.Args[len(cmd.Args)-1]
	for _, want := range []string{"while (Get-Process -Id", "-PassThru", "ExitCode -eq 0", `'/S /D=C:\Users\a\AppData\Local\Programs\Blup Studio'`, `'C:\Users\a\AppData\Local\blup\studio-update\setup''s.exe'`} {
		if !strings.Contains(script, want) {
			t.Fatalf("script lacks %q:\n%s", want, script)
		}
	}
}
