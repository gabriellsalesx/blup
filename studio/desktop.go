package main

import (
	_ "embed"
	"errors"
	"slices"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// The Studio on the desktop: an icon in the tray (where it goes when
// minimized, if the user wants), opening with the computer, a global
// shortcut to show and hide it, and one window only (opening it again
// shows the one already running).

//go:embed build/appicon.png
var trayIcon []byte

// hiddenArg starts the Studio in the tray (used when it opens with the
// computer).
const hiddenArg = "--hidden"

type desktop struct {
	mu     sync.Mutex
	app    *application.App
	win    *application.WebviewWindow
	tray   *application.SystemTray
	hotkey string // the global shortcut registered now
}

var desk desktop

// setupDesktop wires the tray, the minimize hook and the saved shortcut.
func setupDesktop(s *Studio, app *application.App, win *application.WebviewWindow, labels [2]string) {
	desk.app, desk.win = app, win
	prefs := s.Prefs()

	menu := application.NewMenu()
	menu.Add(labels[0]).OnClick(func(*application.Context) { showWindow() })
	menu.AddSeparator()
	menu.Add(labels[1]).OnClick(func(*application.Context) { app.Quit() })
	desk.tray = app.SystemTray.New().SetIcon(trayIcon).SetMenu(menu)
	desk.tray.SetTooltip("Blup Studio")
	desk.tray.OnClick(toggleWindow)
	if !prefs.Tray {
		desk.tray.Hide()
	}

	// Minimized, it goes to the tray (when that is on).
	win.OnWindowEvent(events.Common.WindowMinimise, func(*application.WindowEvent) {
		if s.Prefs().Tray {
			win.Hide()
		}
	})
	if prefs.ToggleKey != "" && !onHyprland() { // there, Hyprland's own binding calls --toggle
		_ = setHotkey(prefs.ToggleKey)
	}
	go fixBinding(prefs.ToggleKey)
}

func showWindow() {
	if desk.win == nil {
		return
	}
	desk.win.Show()
	desk.win.UnMinimise()
	desk.win.Focus()
}

// toggleWindow shows the window, or hides it (to the tray, or minimized).
func toggleWindow() {
	w := desk.win
	if w == nil {
		return
	}
	if w.IsVisible() && !w.IsMinimised() {
		if desk.tray != nil && trayOn() {
			w.Hide()
		} else {
			w.Minimise()
		}
		return
	}
	showWindow()
}

var trayOn = func() bool { return false }

func setHotkey(accel string) error {
	desk.mu.Lock()
	defer desk.mu.Unlock()
	if desk.app == nil {
		return errors.New("not ready")
	}
	if desk.hotkey != "" {
		_ = desk.app.GlobalShortcut.Unregister(desk.hotkey)
		desk.hotkey = ""
	}
	if accel == "" {
		return nil
	}
	if err := desk.app.GlobalShortcut.Register(accel, toggleWindow); err != nil {
		return err
	}
	desk.hotkey = accel
	return nil
}

// AutostartEnabled tells whether the Studio opens with the computer.
func (s *Studio) AutostartEnabled() bool {
	if desk.app == nil {
		return false
	}
	on, _ := desk.app.Autostart.IsEnabled()
	return on
}

// SetAutostart makes the Studio open (in the tray) when the computer starts.
func (s *Studio) SetAutostart(on bool) error {
	if desk.app == nil {
		return errors.New("not available")
	}
	if !on {
		return desk.app.Autostart.Disable()
	}
	return desk.app.Autostart.EnableWithOptions(application.AutostartOptions{Identifier: "blup-studio", Arguments: []string{hiddenArg}})
}

// SetTray turns the tray icon on or off (and saves the choice).
func (s *Studio) SetTray(on bool) error {
	p := s.Prefs()
	p.Tray = on
	if err := s.SavePrefs(p); err != nil {
		return err
	}
	if desk.tray != nil {
		if on {
			desk.tray.Show()
		} else {
			desk.tray.Hide()
		}
	}
	return nil
}

// SetToggleHotkey registers the global shortcut that shows and hides the
// Studio ("" removes it) and saves it.
func (s *Studio) SetToggleHotkey(accel string) error {
	if onHyprland() {
		return s.BindHyprland(accel)
	}
	if err := setHotkey(accel); err != nil {
		return err
	}
	p := s.Prefs()
	p.ToggleKey = accel
	return s.SavePrefs(p)
}

func startHidden(args []string) bool { return slices.Contains(args, hiddenArg) }
