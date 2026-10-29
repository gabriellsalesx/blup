package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Hyprland (Omarchy and others) doesn't let an app take a global shortcut
// by itself: shortcuts are bindings in its own config. There, the Studio's
// "show and hide" shortcut is a binding that runs "blup-studio --toggle";
// the running Studio gets that call (one instance only) and toggles its
// window.

const toggleArg = "--toggle"

func onHyprland() bool {
	return runtime.GOOS == "linux" && os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != ""
}

// Hyprland describes the Hyprland setup for the Shortcuts screen.
type Hyprland struct {
	On    bool   `json:"on"`
	File  string `json:"file"`  // the config file the binding goes in
	Line  string `json:"line"`  // the binding for the shortcut chosen
	Bound bool   `json:"bound"` // the file already has the Studio's binding
	Keys  string `json:"keys"`  // the keys of that binding
}

func hyprFile() (string, bool) {
	home, _ := os.UserHomeDir()
	lua := filepath.Join(home, ".config", "hypr", "bindings.lua")
	if _, err := os.Stat(filepath.Join(home, ".config", "hypr", "hyprland.lua")); err == nil {
		return lua, true
	}
	return filepath.Join(home, ".config", "hypr", "bindings.conf"), false
}

// hyprLine is the binding for accel ("Alt+Space"), in the config's language.
func hyprLine(accel string, lua bool) string {
	exe, _ := os.Executable()
	cmd := "'" + strings.ReplaceAll(exe, "'", `'\''`) + "' " + toggleArg
	parts := strings.Split(accel, "+")
	key := strings.ToUpper(parts[len(parts)-1])
	if key == "`" {
		key = "GRAVE"
	}
	mods := []string{}
	for _, m := range parts[:len(parts)-1] {
		mods = append(mods, strings.ToUpper(strings.Replace(m, "Ctrl", "CTRL", 1)))
	}
	if lua {
		return `o.bind("` + strings.Join(append(mods, key), " + ") + `", "Blup Studio", "` + strings.ReplaceAll(cmd, `"`, `\"`) + `") -- Blup Studio`
	}
	return "bindd = " + strings.Join(mods, " ") + ", " + key + ", Blup Studio, exec, " + cmd + " # Blup Studio"
}

var blupLine = regexp.MustCompile(`(?m)^.*(-- Blup Studio|# Blup Studio)\s*$\n?`)

// HyprlandSetup tells whether the Studio runs on Hyprland and the binding
// for accel.
func (s *Studio) HyprlandSetup(accel string) Hyprland {
	if !onHyprland() {
		return Hyprland{}
	}
	file, lua := hyprFile()
	h := Hyprland{On: true, File: file}
	if accel != "" {
		h.Line = hyprLine(accel, lua)
	}
	if b, err := os.ReadFile(file); err == nil {
		if m := blupLine.Find(b); m != nil {
			h.Bound = true
			h.Keys = string(m)
		}
	}
	return h
}

// BindHyprland writes the Studio's binding for accel in Hyprland's config
// (replacing an older one; "" removes it) and reloads Hyprland.
func (s *Studio) BindHyprland(accel string) error {
	if !onHyprland() {
		return errors.New("not on Hyprland")
	}
	file, lua := hyprFile()
	b, err := os.ReadFile(file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	text := blupLine.ReplaceAllString(string(b), "")
	if accel != "" {
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += hyprLine(accel, lua) + "\n"
	}
	if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
		return err
	}
	p := s.Prefs()
	p.ToggleKey = accel
	if err := s.SavePrefs(p); err != nil {
		return err
	}
	return exec.Command("hyprctl", "reload").Run()
}
