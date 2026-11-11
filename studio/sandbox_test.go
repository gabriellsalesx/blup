package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFlatpakDoesNotWriteHostDesktopConfiguration(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Flatpak is a Linux installation")
	}
	t.Setenv("FLATPAK_ID", "io.getblup.Studio")
	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "fixture")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if h := (&Studio{}).HyprlandSetup("Alt+Space"); h.On || h.Auto || h.File != "" || h.Line != "" {
		t.Fatalf("sandbox exposed host setup: %+v", h)
	}
	if err := writeBinding("Alt+Space"); err == nil {
		t.Fatal("sandbox attempted to write the compositor's configuration")
	}
	if got := autostartPath(); got != "" {
		t.Fatalf("sandbox would use a nonfunctional autostart file: %s", got)
	}
	fixAutostart()
	if _, err := os.Stat(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "autostart")); !os.IsNotExist(err) {
		t.Fatalf("native autostart directory created in Flatpak: %v", err)
	}
}

func TestFlatpakShellDoesNotUseHostStartup(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Flatpak is a Linux installation")
	}
	t.Setenv("FLATPAK_ID", "io.getblup.Studio")
	t.Setenv("SHELL", "/host-only/fish")
	name, args := shell()
	if name != "/bin/bash" || len(args) != 2 || args[0] != "--noprofile" || args[1] != "--norc" {
		t.Fatalf("sandbox terminal uses host startup: %s %v", name, args)
	}
}
