//go:build linux && !android

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func inspectPreview(session string) error {
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
	if state.Instance != "" {
		if err := validateFlatpakPreview(state); err != nil {
			return err
		}
	}
	status, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(state.PID), "status"))
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(line, "Name:") || strings.HasPrefix(line, "NSpid:") {
			fmt.Println(line)
		}
	}
	cmd := exec.Command("hyprctl", "-j", "clients")
	cmd.Env = previewEnvironment(desktop)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	var windows []desktopWindow
	if err := json.Unmarshal(output, &windows); err != nil {
		return err
	}
	for _, w := range windows {
		if previewOwnsWindow(state, w) {
			// Never log window titles or the contents of other applications.
			fmt.Printf("Studio window: PID=%d class=%s mapped=%t hidden=%t address=%s\n", w.PID, w.Class, w.Mapped, w.Hidden, w.Address)
		}
	}
	return nil
}
