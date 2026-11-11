//go:build linux && !android

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const previewAppID = "io.getblup.Studio"

func previewEnvironment(desktop map[string]string) []string {
	env := []string{"PATH=" + os.Getenv("PATH"), "LANG=en_US.UTF-8"}
	for _, key := range sessionKeys {
		if value := desktop[key]; value != "" {
			env = append(env, key+"="+value)
		}
	}
	return env
}

func runFlatpak(bundle, runtimeDir, view, theme, session string) error {
	if (view != "notes" && view != "site" && view != "library") || (theme != "light" && theme != "dark") {
		return errors.New("invalid preview view or theme")
	}
	desktop, err := desktopSession(session)
	if err != nil {
		return err
	}
	bundle, err = filepath.Abs(bundle)
	if err != nil {
		return err
	}
	if filepath.Ext(bundle) != ".flatpak" {
		return errors.New("the preview requires a .flatpak bundle")
	}
	env := previewEnvironment(desktop)
	// The application has the real installed ID. Refuse to activate an
	// already running instance that might contain the person's real notes.
	ps := exec.Command("flatpak", "ps", "--columns=application")
	ps.Env = env
	output, err := ps.Output()
	if err != nil {
		return err
	}
	for _, line := range strings.Fields(string(output)) {
		if line == previewAppID {
			return errors.New("close the running Blup Flatpak before starting its disposable preview")
		}
	}
	root, err := os.MkdirTemp("", "blup-store-preview-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	env = append(env, "FLATPAK_USER_DIR="+filepath.Join(root, "installation"))
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME"} {
		value := filepath.Join(root, strings.ToLower(key))
		if err := os.Setenv(key, value); err != nil {
			return err
		}
		env = append(env, key+"="+value)
	}
	if runtimeDir != "" {
		runtimeDir, err = filepath.Abs(runtimeDir)
		if err != nil {
			return err
		}
		env = append(env, "FLATPAK_SYSTEM_DIR="+runtimeDir)
	}
	if err := prepare(root, view, theme); err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	// This is a bounded test session, even if its launcher loses contact.
	ctx, timeout := context.WithTimeout(ctx, 8*time.Minute)
	defer timeout()
	install := exec.CommandContext(ctx, "flatpak", "--user", "install", "--bundle", "--no-deps", "--no-related", "--noninteractive", "-y", bundle)
	install.Env, install.Stdout, install.Stderr = env, os.Stdout, os.Stderr
	if err := install.Run(); err != nil {
		return err
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	defer reader.Close()
	// Flatpak sets the XDG directories itself. Set our disposable directories
	// inside the sandbox, after that setup, before starting the real binary.
	args := []string{"--user", "run", "--die-with-parent", "--instance-id-fd=3", "--filesystem=" + root, "--env=LANG=en_US.UTF-8", "--command=sh", previewAppID, "-c", `export XDG_CONFIG_HOME="$1" XDG_CACHE_HOME="$2" XDG_DATA_HOME="$3" XDG_STATE_HOME="$4" BLUP_NO_UPDATE_CHECK=1; exec /app/bin/blup-studio`, "blup-preview"}
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME"} {
		args = append(args, os.Getenv(key))
	}
	cmd := exec.CommandContext(ctx, "flatpak", args...)
	cmd.Env, cmd.Dir = env, root
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.ExtraFiles = []*os.File{writer}
	if err := cmd.Start(); err != nil {
		writer.Close()
		return err
	}
	writer.Close()
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		_ = cmd.Wait()
		return errors.New("the Flatpak did not report its instance ID")
	}
	instance := strings.TrimSpace(scanner.Text())
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(instance) {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return errors.New("invalid Flatpak instance ID")
	}
	defer func() {
		stop := exec.Command("flatpak", "kill", instance)
		stop.Env = env
		_ = stop.Run()
	}()
	pid := 0
	for i := 0; i < 50; i++ {
		ps := exec.Command("flatpak", "ps", "--columns=instance,child-pid,application")
		ps.Env = env
		if output, err := ps.Output(); err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				f := strings.Fields(line)
				if len(f) == 3 && f[0] == instance && f[2] == previewAppID {
					pid, _ = strconv.Atoi(f[1])
				}
			}
		}
		if pid > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	state := previewState{PID: pid, Root: root, View: view, Instance: instance}
	if err := validateFlatpakPreview(state); err != nil {
		cancel()
		_ = cmd.Wait()
		return err
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err == nil {
		err = os.WriteFile("bin/store-preview.json", b, 0o600)
	}
	if err != nil {
		cancel()
		_ = cmd.Wait()
		return err
	}
	fmt.Printf("Isolated Flatpak preview started; PID %d, instance %s\n", pid, instance)
	err = cmd.Wait()
	if ctx.Err() != nil {
		return nil
	}
	if exit, ok := err.(*exec.ExitError); ok && (exit.ExitCode() == 137 || exit.ExitCode() == 143) {
		return nil
	}
	return err
}

func validateFlatpakPreview(state previewState) error {
	if state.PID <= 0 || !strings.HasPrefix(filepath.Base(state.Root), "blup-store-preview-") || !regexp.MustCompile(`^[0-9]+$`).MatchString(state.Instance) {
		return errors.New("invalid isolated Flatpak preview state")
	}
	info, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(state.PID), "root", ".flatpak-info"))
	prefix := "\napp-path=" + filepath.Join(state.Root, "installation", "app", previewAppID) + "/"
	if err != nil || !strings.Contains(string(info), prefix) || !strings.Contains(string(info), "\ninstance-id="+state.Instance+"\n") {
		return errors.New("the process is not the isolated Flatpak preview")
	}
	return nil
}

func previewOwnsWindow(state previewState, window desktopWindow) bool {
	if state.Instance == "" {
		return window.PID == state.PID
	}
	if window.Class != previewAppID {
		return false
	}
	state.PID = window.PID
	return validateFlatpakPreview(state) == nil
}

func stopFlatpakPreview(state previewState, session string) error {
	if _, err := os.Stat(state.Root); os.IsNotExist(err) && strings.HasPrefix(filepath.Base(state.Root), "blup-store-preview-") && regexp.MustCompile(`^[0-9]+$`).MatchString(state.Instance) {
		fmt.Println("The isolated Flatpak preview has already stopped")
		return nil
	}
	if err := validateFlatpakPreview(state); err != nil {
		return err
	}
	desktop, err := desktopSession(session)
	if err != nil {
		return err
	}
	cmd := exec.Command("flatpak", "kill", state.Instance)
	cmd.Env = previewEnvironment(desktop)
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Println("Stopped only the isolated Flatpak preview instance")
	return nil
}
