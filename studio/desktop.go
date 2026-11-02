package main

import (
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

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
	menu   *application.Menu
	items  [2]*application.MenuItem // the tray menu's Open and Quit
	hotkey string                   // the global shortcut registered now
}

var desk desktop

// trayLabels are the tray menu's Open and Quit in a language.
func trayLabels(lang string) [2]string {
	switch lang {
	case "pt":
		return [2]string{"Abrir o Blup Studio", "Sair"}
	case "es":
		return [2]string{"Abrir Blup Studio", "Salir"}
	}
	return [2]string{"Open Blup Studio", "Quit"}
}

// trayLanguage puts the tray's menu in a new language (chosen in the
// settings, without a restart).
func trayLanguage(lang string) {
	if desk.menu == nil || desk.items[0] == nil {
		return
	}
	labels := trayLabels(lang)
	desk.items[0].SetLabel(labels[0])
	desk.items[1].SetLabel(labels[1])
	desk.menu.Update()
}

// quitting is set by "Quit" in the tray's menu: then closing really closes.
var quitting atomic.Bool

// setupDesktop wires the tray, the minimize hook and the saved shortcut.
func setupDesktop(s *Studio, app *application.App, win *application.WebviewWindow) {
	desk.app, desk.win = app, win
	prefs := s.Prefs()

	labels := trayLabels(s.lang)
	menu := application.NewMenu()
	desk.items[0] = menu.Add(labels[0]).OnClick(func(*application.Context) { showWindow() })
	menu.AddSeparator()
	desk.items[1] = menu.Add(labels[1]).OnClick(func(*application.Context) { quitting.Store(true); app.Quit() })
	desk.menu = menu
	desk.tray = app.SystemTray.New().SetIcon(trayIcon).SetMenu(menu)
	if runtime.GOOS == "linux" {
		// The tray item's name (Wails says "Wails" otherwise). On macOS a
		// label would show as text next to the icon, so only here.
		desk.tray.SetLabel("Blup Studio")
	}
	desk.tray.SetTooltip("Blup Studio")
	desk.tray.OnClick(toggleWindow)
	if !prefs.Tray {
		desk.tray.Hide()
	}

	// With the tray on, closing the window only hides it (like minimizing):
	// the terminals and AIs keep running. "Quit" in the tray's menu quits.
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if !quitting.Load() && s.Prefs().Tray {
			e.Cancel()
			win.Hide()
		}
	})

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
	go fixAutostart(app)
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

// autostartID names the Studio's entry (on Linux, the .desktop file).
const autostartID = "blup-studio"

// AutostartEnabled tells whether the Studio opens with the computer.
func (s *Studio) AutostartEnabled() bool {
	if desk.app == nil {
		return false
	}
	on, _ := desk.app.Autostart.IsEnabled()
	return on || linuxAutostartFile() != ""
}

// linuxAutostartFile is the Studio's .desktop file in the autostart folder
// ("" when there is none, or not on Linux).
func linuxAutostartFile() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	f := filepath.Join(dir, "autostart", autostartID+".desktop")
	if _, err := os.Stat(f); err != nil {
		return ""
	}
	return f
}

// fixAutostart points the "open with the computer" entry at this program
// again when the Studio moved (an update, another install place): the old
// one would run nothing at login.
func fixAutostart(app *application.App) {
	f := linuxAutostartFile()
	if f == "" {
		return
	}
	exe, err := os.Executable()
	if err != nil || strings.Contains(filepath.Base(exe), "server") {
		return
	}
	if b, err := os.ReadFile(f); err == nil && !strings.Contains(string(b), exe) {
		_ = app.Autostart.EnableWithOptions(application.AutostartOptions{Identifier: autostartID, Arguments: []string{hiddenArg}})
	}
}

// SetAutostart makes the Studio open (in the tray) when the computer starts.
func (s *Studio) SetAutostart(on bool) error {
	if desk.app == nil {
		return errors.New("not available")
	}
	if !on {
		if f := linuxAutostartFile(); f != "" {
			_ = os.Remove(f) // also an entry left by an older install place
		}
		return desk.app.Autostart.Disable()
	}
	return desk.app.Autostart.EnableWithOptions(application.AutostartOptions{Identifier: autostartID, Arguments: []string{hiddenArg}})
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
	if accel != "" && !validAccel(accel) {
		return errors.New("invalid shortcut")
	}
	if onHyprland() {
		return s.bindHyprland(accel)
	}
	if err := setHotkey(accel); err != nil {
		return err
	}
	p := s.Prefs()
	p.ToggleKey = accel
	return s.SavePrefs(p)
}

func startHidden(args []string) bool { return slices.Contains(args, hiddenArg) }
