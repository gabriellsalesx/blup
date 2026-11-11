//go:build linux && !android

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// exerciseFlatpak sends keys only while the verified disposable window has
// focus. It checks the resulting files under its temporary example directory.
func exerciseFlatpak(session string) error {
	b, err := os.ReadFile("bin/store-preview.json")
	if err != nil {
		return err
	}
	var state previewState
	if err := json.Unmarshal(b, &state); err != nil {
		return err
	}
	if err := validateFlatpakPreview(state); err != nil {
		return err
	}
	if state.View != "site" {
		return errors.New("the GUI exercise requires VIEW=site")
	}
	desktop, err := desktopSession(session)
	if err != nil {
		return err
	}
	env := previewEnvironment(desktop)
	command := func(name string, args ...string) ([]byte, error) {
		cmd := exec.Command(name, args...)
		cmd.Env = env
		return cmd.Output()
	}
	data, err := command("hyprctl", "-j", "clients")
	if err != nil {
		return err
	}
	var windows []desktopWindow
	if err := json.Unmarshal(data, &windows); err != nil {
		return err
	}
	var target desktopWindow
	for _, w := range windows {
		if previewOwnsWindow(state, w) && w.Mapped && !w.Hidden {
			if target.Address != "" {
				return errors.New("more than one isolated preview window is open")
			}
			target = w
		}
	}
	if !regexp.MustCompile(`^0x[0-9a-fA-F]+$`).MatchString(target.Address) {
		return errors.New("isolated preview window not found")
	}
	focused := func() bool {
		data, err := command("hyprctl", "-j", "activewindow")
		var w desktopWindow
		return err == nil && json.Unmarshal(data, &w) == nil && w.PID == target.PID && w.Address == target.Address
	}
	var previous desktopWindow
	if data, err := command("hyprctl", "-j", "activewindow"); err == nil {
		_ = json.Unmarshal(data, &previous)
	}
	if _, err := command("hyprctl", "eval", fmt.Sprintf("hl.dispatch(hl.dsp.focus({window=%q}))", "address:"+target.Address)); err != nil {
		return err
	}
	defer func() {
		if focused() && previous.Address != target.Address && regexp.MustCompile(`^0x[0-9a-fA-F]+$`).MatchString(previous.Address) {
			_, _ = command("hyprctl", "eval", fmt.Sprintf("hl.dispatch(hl.dsp.focus({window=%q}))", "address:"+previous.Address))
		}
	}()
	typeKeys := func(args ...string) error {
		if !focused() {
			return errors.New("keyboard test cancelled: another window has focus")
		}
		_, err := command("wtype", args...)
		return err
	}
	waitFile := func(path string, accept func([]byte) bool) error {
		for i := 0; i < 60; i++ {
			data, err := os.ReadFile(path)
			if err == nil && accept(data) {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
		return errors.New("the expected example file was not saved")
	}
	time.Sleep(300 * time.Millisecond)
	if err := typeKeys("-M", "ctrl", "-k", "End", "-m", "ctrl"); err != nil {
		return err
	}
	const marker = "Flatpak editor smoke test."
	if err := typeKeys("\n\n" + marker + "\n"); err != nil {
		return err
	}
	if err := typeKeys("-M", "ctrl", "-k", "s", "-m", "ctrl"); err != nil {
		return err
	}
	if err := waitFile(filepath.Join(state.Root, "journal", "A place for small ideas.md"), func(b []byte) bool { return strings.Contains(string(b), marker) }); err != nil {
		return fmt.Errorf("editing and saving: %w", err)
	}
	fmt.Println("GUI editing and saving passed")
	if err := typeKeys("-M", "ctrl", "-k", "n", "-m", "ctrl"); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	if err := typeKeys("Sandbox check"); err != nil {
		return err
	}
	if err := typeKeys("-k", "Return"); err != nil {
		return err
	}
	if err := waitFile(filepath.Join(state.Root, "journal", "Sandbox check.md"), func([]byte) bool { return true }); err != nil {
		return fmt.Errorf("creating a note: %w", err)
	}
	fmt.Println("GUI note creation passed")
	if err := typeKeys("-M", "ctrl", "-k", "grave", "-m", "ctrl"); err != nil {
		return err
	}
	time.Sleep(1500 * time.Millisecond)
	proof := filepath.Join(state.Root, "terminal-proof.txt")
	quoted := "'" + strings.ReplaceAll(proof, "'", "'\"'\"'") + "'"
	if err := typeKeys("printf '%s\\n' \"$FLATPAK_ID\" \"$SHELL\" > " + quoted + "\n"); err != nil {
		return err
	}
	if err := waitFile(proof, func(b []byte) bool { return string(b) == previewAppID+"\n/bin/bash\n" }); err != nil {
		return fmt.Errorf("sandbox terminal: %w", err)
	}
	fmt.Println("GUI terminal passed: the app ID and Bash belong to the sandbox")
	return typeKeys("-M", "ctrl", "-k", "grave", "-m", "ctrl")
}
