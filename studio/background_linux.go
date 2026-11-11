//go:build linux && !android

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/godbus/dbus/v5"
)

const (
	backgroundBus       = "org.freedesktop.portal.Desktop"
	backgroundPath      = dbus.ObjectPath("/org/freedesktop/portal/desktop")
	backgroundInterface = "org.freedesktop.portal.Background"
	backgroundRequest   = "org.freedesktop.portal.Request"
)

// portalAutostart requests the desktop's permission instead of writing a
// .desktop file inside the sandbox, where it cannot start the host's app.
func portalAutostart(ctx context.Context, on bool, lang string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("autostart portal: %w", err)
	}
	defer conn.Close()
	reason := "Open Blup Studio when you sign in to your computer"
	switch lang {
	case "pt":
		reason = "Abrir o Blup Studio ao entrar no computador"
	case "es":
		reason = "Abrir Blup Studio al iniciar sesión en la computadora"
	}
	return applyPortalAutostart(ctx, backgroundConnection{conn}, on, reason)
}

func applyPortalAutostart(ctx context.Context, portal backgroundPortal, on bool, reason string) error {
	if err := requestPortalAutostart(ctx, portal, on, reason); err != nil {
		return err
	}
	// The portal has no autostart-status query. Remember only a confirmed
	// response, never a cancelled or denied request.
	f, err := portalAutostartFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f), 0o700); err != nil {
		return err
	}
	value := "off\n"
	if on {
		value = "on\n"
	}
	return writeFileAtomic(f, []byte(value), 0o600)
}

func portalAutostartFile() (string, error) {
	dir, err := config.Dir()
	return filepath.Join(dir, "flatpak-autostart"), err
}

func portalAutostartEnabled() bool {
	f, err := portalAutostartFile()
	if err != nil {
		return false
	}
	b, err := os.ReadFile(f)
	return err == nil && strings.TrimSpace(string(b)) == "on"
}

type backgroundPortal interface {
	name() string
	subscribe(context.Context, dbus.ObjectPath, chan *dbus.Signal) (func(), error)
	request(context.Context, map[string]dbus.Variant) (dbus.ObjectPath, error)
	closeRequest(context.Context, dbus.ObjectPath)
}

type backgroundConnection struct{ *dbus.Conn }

func (c backgroundConnection) name() string { return c.Names()[0] }

func (c backgroundConnection) subscribe(ctx context.Context, namespace dbus.ObjectPath, signals chan *dbus.Signal) (func(), error) {
	c.Signal(signals)
	cleanup := func() { c.RemoveSignal(signals) }
	err := c.AddMatchSignalContext(ctx,
		dbus.WithMatchSender(backgroundBus),
		dbus.WithMatchInterface(backgroundRequest),
		dbus.WithMatchMember("Response"),
		dbus.WithMatchPathNamespace(namespace),
	)
	return cleanup, err
}

func (c backgroundConnection) request(ctx context.Context, options map[string]dbus.Variant) (dbus.ObjectPath, error) {
	var handle dbus.ObjectPath
	err := c.Object(backgroundBus, backgroundPath).
		CallWithContext(ctx, backgroundInterface+".RequestBackground", 0, "", options).Store(&handle)
	return handle, err
}

func (c backgroundConnection) closeRequest(ctx context.Context, handle dbus.ObjectPath) {
	c.Object(backgroundBus, handle).CallWithContext(ctx, backgroundRequest+".Close", 0)
}

func requestPortalAutostart(ctx context.Context, portal backgroundPortal, on bool, reason string) error {
	sender := strings.ReplaceAll(strings.TrimPrefix(portal.name(), ":"), ".", "_")
	prefix := "/org/freedesktop/portal/desktop/request/" + sender + "/"
	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant("blup_autostart"),
		"reason":       dbus.MakeVariant(reason),
		"autostart":    dbus.MakeVariant(on),
		"commandline":  dbus.MakeVariant([]string{"blup-studio", hiddenArg}),
	}
	// Subscribe before making the request: a portal can answer before its
	// method reply reaches us. Match its sender as well as the interface.
	signals := make(chan *dbus.Signal, 8)
	cleanup, err := portal.subscribe(ctx, dbus.ObjectPath(strings.TrimSuffix(prefix, "/")), signals)
	defer cleanup()
	if err != nil {
		return fmt.Errorf("autostart portal: %w", err)
	}
	handle, err := portal.request(ctx, options)
	if err != nil {
		return fmt.Errorf("autostart portal: %w", err)
	}
	if !handle.IsValid() || !strings.HasPrefix(string(handle), prefix) {
		return errors.New("autostart portal returned an invalid request handle")
	}
	for {
		select {
		case sig, ok := <-signals:
			if !ok {
				return errors.New("autostart portal connection closed")
			}
			if sig == nil || sig.Path != handle || sig.Name != backgroundRequest+".Response" {
				continue
			}
			var code uint32
			var results map[string]dbus.Variant
			if err := dbus.Store(sig.Body, &code, &results); err != nil {
				return fmt.Errorf("autostart portal response: %w", err)
			}
			if code != 0 {
				return errors.New("autostart was not authorized by the desktop")
			}
			granted, ok := results["autostart"].Value().(bool)
			if !ok || granted != on {
				return errors.New("the desktop did not apply the requested autostart setting")
			}
			return nil
		case <-ctx.Done():
			// Close the dialog when the operation is cancelled, without
			// trying to reuse the cancelled context for that final call.
			closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			portal.closeRequest(closeCtx, handle)
			cancel()
			return ctx.Err()
		}
	}
}
