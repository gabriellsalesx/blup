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
	// Auto: the Studio writes the binding itself (Omarchy, whose config it
	// knows). Elsewhere the user pastes Line in their config.
	Auto bool `json:"auto"`
}

// onOmarchy tells whether Hyprland is Omarchy's, whose bindings file (and
// its o.bind) the Studio knows how to write.
func onOmarchy() bool {
	home, _ := os.UserHomeDir()
	for _, f := range []string{"hyprland.lua", "hyprland.conf"} {
		if b, err := os.ReadFile(filepath.Join(home, ".config", "hypr", f)); err == nil && strings.Contains(string(b), "omarchy") {
			return true
		}
	}
	return false
}

func hyprFile() (string, bool) {
	home, _ := os.UserHomeDir()
	lua := filepath.Join(home, ".config", "hypr", "bindings.lua")
	if _, err := os.Stat(filepath.Join(home, ".config", "hypr", "hyprland.lua")); err == nil {
		return lua, true
	}
	return filepath.Join(home, ".config", "hypr", "bindings.conf"), false
}

// validAccel tells whether accel is a shortcut as the window records it
// ("Ctrl+Shift+K", "Alt+Space", "Super+`", "F5"): parts joined by "+", each
// a key name of letters and digits or one printable sign. Anything else
// (a quote, a backslash, a comma, a line break) would go into Hyprland's
// config as code: a binding that runs any command at the next reload.
func validAccel(accel string) bool {
	if accel == "" || len(accel) > 64 {
		return false
	}
	for _, part := range strings.Split(accel, "+") {
		if !accelPart.MatchString(part) {
			return false
		}
	}
	return true
}

var accelPart = regexp.MustCompile("^(?:[\\p{L}\\p{N}]+|[`~!@$%&*()\\-_=\\[\\]{};:<>./?|^])$")

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
	if accel != "" && !validAccel(accel) {
		accel = ""
	}
	file, lua := hyprFile()
	h := Hyprland{On: true, File: file, Auto: onOmarchy()}
	if accel != "" {
		// Outside Omarchy, plain Hyprland syntax to paste in hyprland.conf.
		h.Line = hyprLine(accel, lua && h.Auto)
		if !h.Auto {
			h.Line = strings.TrimSuffix(strings.Replace(h.Line, "bindd = ", "bind = ", 1), " # Blup Studio")
			h.Line = strings.Replace(h.Line, ", Blup Studio, exec,", ", exec,", 1)
		}
	}
	if b, err := os.ReadFile(file); err == nil {
		if m := blupLine.Find(b); m != nil {
			h.Bound = true
			h.Keys = string(m)
		}
	}
	return h
}

// bindHyprland saves the shortcut and, on Omarchy, writes it in Hyprland's
// config (elsewhere the user pastes the line the Shortcuts screen shows).
func (s *Studio) bindHyprland(accel string) error {
	if !onHyprland() {
		return errors.New("not on Hyprland")
	}
	if err := updatePrefs(func(p *Prefs) { p.ToggleKey = accel }); err != nil {
		return err
	}
	if !onOmarchy() {
		return nil // the Shortcuts screen shows the line to paste
	}
	return writeBinding(accel)
}

// writeBinding puts the Studio's binding for accel in Omarchy's bindings
// file (replacing an older one; "" removes it) and reloads Hyprland.
func writeBinding(accel string) error {
	if accel != "" && !validAccel(accel) {
		return errors.New("invalid shortcut")
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
	return exec.Command("hyprctl", "reload").Run()
}

// fixBinding writes the Studio's binding when it is missing (the default
// shortcut on a first run) or when the Studio moved (an update, another
// install place): the line must run this program.
func fixBinding(accel string) {
	if accel == "" || !validAccel(accel) || !onHyprland() || !onOmarchy() {
		return
	}
	// Only the app itself (not a test build, like blup-studio-server).
	if exe, _ := os.Executable(); !strings.HasPrefix(filepath.Base(exe), "blup-studio") || strings.Contains(filepath.Base(exe), "server") {
		return
	}
	file, lua := hyprFile()
	b, err := os.ReadFile(file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}
	// Missing (the default shortcut, first run) or pointing elsewhere.
	if m := blupLine.Find(b); m == nil || strings.TrimSpace(string(m)) != hyprLine(accel, lua) {
		_ = writeBinding(accel)
	}
}
