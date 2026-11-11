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
	"strconv"
	"strings"
	"time"
)

type desktopWindow struct {
	Address string `json:"address"`
	Class   string `json:"class"`
	PID     int    `json:"pid"`
	At      []int  `json:"at"`
	Size    []int  `json:"size"`
	Mapped  bool   `json:"mapped"`
	Hidden  bool   `json:"hidden"`
}

func captureWindow(session, output string) error {
	desktop, err := desktopSession(session)
	if err != nil {
		return err
	}
	b, err := os.ReadFile("bin/store-preview.json")
	if err != nil {
		return err
	}
	var state previewState
	if err := json.Unmarshal(b, &state); err != nil {
		return err
	}
	if state.PID <= 0 || !strings.HasPrefix(filepath.Base(state.Root), "blup-store-preview-") {
		return errors.New("invalid preview state")
	}
	if state.Instance != "" {
		if err := validateFlatpakPreview(state); err != nil {
			return err
		}
	} else {
		exe, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(state.PID), "exe"))
		expected, pathErr := filepath.Abs("bin/blup-studio-store-preview")
		if err != nil || pathErr != nil || strings.TrimSuffix(exe, " (deleted)") != expected {
			return errors.New("the isolated preview is not running")
		}
	}
	if filepath.Ext(output) != ".png" {
		return errors.New("capture destination must be a PNG")
	}
	env := []string{"PATH=" + os.Getenv("PATH"), "LANG=en_US.UTF-8"}
	for _, key := range sessionKeys {
		if value := desktop[key]; value != "" {
			env = append(env, key+"="+value)
		}
	}
	command := func(name string, args ...string) ([]byte, error) {
		cmd := exec.Command(name, args...)
		cmd.Env = env
		return cmd.Output()
	}
	window := func() (desktopWindow, error) {
		data, err := command("hyprctl", "-j", "clients")
		if err != nil {
			return desktopWindow{}, errors.New("could not inspect the desktop's windows")
		}
		var windows []desktopWindow
		if err := json.Unmarshal(data, &windows); err != nil {
			return desktopWindow{}, err
		}
		for _, w := range windows {
			if previewOwnsWindow(state, w) && w.Mapped && !w.Hidden && regexp.MustCompile(`^0x[0-9a-fA-F]+$`).MatchString(w.Address) {
				return w, nil
			}
		}
		return desktopWindow{}, errors.New("isolated preview window not found")
	}
	w, err := window()
	if err != nil {
		return err
	}
	var previous desktopWindow
	if data, err := command("hyprctl", "-j", "activewindow"); err == nil {
		_ = json.Unmarshal(data, &previous)
	}
	if previous.Address != w.Address && regexp.MustCompile(`^0x[0-9a-fA-F]+$`).MatchString(previous.Address) {
		defer func() {
			_, _ = command("hyprctl", "eval", fmt.Sprintf("hl.dispatch(hl.dsp.focus({window=%q}))", "address:"+previous.Address))
		}()
	}
	// Target only this process's window. No desktop configuration is edited,
	// and other windows' titles or content are never logged or captured.
	selector := "address:" + w.Address
	script := fmt.Sprintf("hl.dispatch(hl.dsp.window.float({window=%q,action='on'})); hl.dispatch(hl.dsp.window.resize({window=%q,x=1360,y=860,relative=false})); hl.dispatch(hl.dsp.window.center({window=%q})); hl.dispatch(hl.dsp.focus({window=%q}))", selector, selector, selector, selector)
	if _, err := command("hyprctl", "eval", script); err != nil {
		return errors.New("could not focus the preview window")
	}
	time.Sleep(time.Second)
	w, err = window()
	if err != nil {
		return err
	}
	activeData, err := command("hyprctl", "-j", "activewindow")
	if err != nil {
		return err
	}
	var active desktopWindow
	if err := json.Unmarshal(activeData, &active); err != nil {
		return err
	}
	if active.PID != w.PID || active.Address != w.Address {
		return errors.New("capture cancelled: another window has focus")
	}
	if len(w.At) != 2 || len(w.Size) != 2 || w.Size[0] < 900 || w.Size[1] < 560 {
		return errors.New("preview window geometry is invalid")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		return err
	}
	geometry := fmt.Sprintf("%d,%d %dx%d", w.At[0], w.At[1], w.Size[0], w.Size[1])
	// The pointer goes just outside the window's top-left corner, so no
	// cursor (or hover highlight) lands in the picture: Hyprland may draw
	// the cursor into what grim copies.
	park := fmt.Sprintf("hl.dispatch(hl.dsp.cursor.move({x=%d,y=%d}))", max(0, w.At[0]-20), max(0, w.At[1]-20))
	if _, err := command("hyprctl", "eval", park); err != nil {
		return errors.New("could not move the pointer off the preview window")
	}
	time.Sleep(400 * time.Millisecond)
	if _, err := command("grim", "-g", geometry, output); err != nil {
		return errors.New("native window capture failed")
	}
	fmt.Printf("Captured only the isolated Studio window: %s (%dx%d)\n", output, w.Size[0], w.Size[1])
	return nil
}
