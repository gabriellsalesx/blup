//go:build linux && !android

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/hugo"
	"github.com/gabriellsalesx/blup/internal/pagefind"
	"github.com/gabriellsalesx/blup/internal/sandbox"
	"github.com/godbus/dbus/v5"
)

func probeSandbox(root string) error {
	if !sandbox.Flatpak() || os.Getenv("FLATPAK_ID") != previewAppID || !strings.HasPrefix(filepath.Base(root), "blup-store-preview-") {
		return errors.New("this probe requires the isolated Studio Flatpak")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if app.UpdatesBy() != "flatpak" {
		return errors.New("the application did not detect the Flatpak update channel")
	}
	h, found := hugo.Find(ctx)
	if !found || h.Path != "/app/bin/hugo" {
		return errors.New("Hugo was not selected from the package")
	}
	p, err := pagefind.Ensure(ctx)
	if err != nil || p != "/app/bin/pagefind" {
		return errors.New("Pagefind was not selected from the package")
	}
	if _, err := hugo.Download(ctx, nil); !errors.Is(err, hugo.ErrBundledMissing) {
		return errors.New("the sandbox did not reject executable downloads")
	}
	fmt.Printf("Sandbox detection, Flatpak updates, packaged Hugo %s and Pagefind %s, and refusal of executable downloads passed\n", h.Version, pagefind.Pinned)
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()
	portal := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	for _, name := range []string{"FileChooser", "Background", "GlobalShortcuts", "OpenURI", "Notification"} {
		value, err := portal.GetProperty("org.freedesktop.portal." + name + ".version")
		if err != nil {
			fmt.Println("Portal " + name + ": unavailable in this desktop session")
			if name == "FileChooser" {
				return errors.New("the FileChooser portal is not available")
			}
			continue
		}
		fmt.Printf("Portal %s version %v available from inside the sandbox\n", name, value.Value())
	}
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	options := map[string]dbus.Variant{
		"handle_token":   dbus.MakeVariant("blup_preview_" + hex.EncodeToString(nonce[:])),
		"modal":          dbus.MakeVariant(false),
		"directory":      dbus.MakeVariant(true),
		"current_folder": dbus.MakeVariant(append([]byte(root), 0)),
	}
	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	namespace := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_"))
	if err := conn.AddMatchSignal(dbus.WithMatchSender("org.freedesktop.portal.Desktop"), dbus.WithMatchPathNamespace(namespace), dbus.WithMatchInterface("org.freedesktop.portal.Request"), dbus.WithMatchMember("Response")); err != nil {
		return err
	}
	var handle dbus.ObjectPath
	if err := portal.CallWithContext(ctx, "org.freedesktop.portal.FileChooser.OpenFile", 0, "", "Blup: disposable folder chooser test", options).Store(&handle); err != nil {
		return err
	}
	if !strings.HasPrefix(string(handle), "/org/freedesktop/portal/desktop/request/") {
		return errors.New("invalid FileChooser request handle")
	}
	response := func(wait time.Duration) error {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		for {
			select {
			case signal := <-signals:
				if signal.Path != handle || len(signal.Body) == 0 {
					continue
				}
				if code, ok := signal.Body[0].(uint32); ok && code == 1 {
					fmt.Println("FileChooser returned cancellation without choosing a personal folder")
					return nil
				}
				return fmt.Errorf("FileChooser ended before cancellation with response code %v", signal.Body[0])
			case <-timer.C:
				return errors.New("no FileChooser response was received")
			}
		}
	}
	if err := conn.Object("org.freedesktop.portal.Desktop", handle).CallWithContext(ctx, "org.freedesktop.portal.Request.Close", 0).Err; err != nil {
		return response(time.Second)
	}
	fmt.Println("FileChooser request opened and cancelled without choosing a personal folder")
	return nil
}

func checkPreviewSandbox(session, runtimeDir string) error {
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
	desktop, err := desktopSession(session)
	if err != nil {
		return err
	}
	env := append(previewEnvironment(desktop), "FLATPAK_USER_DIR="+filepath.Join(state.Root, "installation"))
	if runtimeDir != "" {
		runtimeDir, err = filepath.Abs(runtimeDir)
		if err != nil {
			return err
		}
		env = append(env, "FLATPAK_SYSTEM_DIR="+runtimeDir)
	}
	probe, err := filepath.Abs("bin/blup-flatpak-probe")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "flatpak", "--user", "run", "--die-with-parent", "--unshare=network", "--filesystem="+state.Root, "--filesystem="+probe+":ro", "--command="+probe, previewAppID, "-probe", "-probe-dir="+state.Root)
	cmd.Env, cmd.Stdout, cmd.Stderr = env, os.Stdout, os.Stderr
	err = cmd.Run()
	if err != nil {
		// Report only service states, never journals, window titles or user data.
		status := exec.Command("systemctl", "--user", "show", "-p", "Id", "-p", "ActiveState", "-p", "Result", "xdg-desktop-portal-gtk.service", "at-spi-dbus-bus.service", "at-spi2-registryd.service")
		status.Env, status.Stdout = previewEnvironment(desktop), os.Stdout
		_ = status.Run()
	}
	return err
}
