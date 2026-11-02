// Blup Studio: the Blup CLI in a window. Notes and folders on the left, a
// Markdown editor in the middle and the site's live preview on the right;
// everything is plain .md files on disk, so it works offline and the notes
// still open in Obsidian, VS Code or any editor.
package main

import (
	"embed"
	"log"
	"os"
	"slices"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// The Studio speaks the language chosen in Blup (blup language), else
	// the system's: Portuguese on a computer in Portuguese, English on the
	// others.
	lang := string(i18n.System())
	if g, err := config.LoadGlobal(); err == nil && g.Lang != "" {
		lang = g.Lang
	}
	i18n.Set(i18n.Lang(lang))
	studio := &Studio{lang: lang}
	app := application.New(application.Options{
		Name:        "Blup Studio",
		Description: "Write in Markdown, preview and publish your site",
		Services:    []application.Service{application.NewService(studio)},
		// The local translation's engine and models are served from where
		// they were downloaded (/local-translate/...).
		Assets:     application.AssetOptions{Handler: application.AssetFileServerFS(assets), Middleware: serveLocalTranslate},
		Mac:        application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		OnShutdown: studio.shutdown,
		// Opening it again shows the window already running.
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "dev.blup.studio",
			OnSecondInstanceLaunch: func(d application.SecondInstanceData) {
				if slices.Contains(d.Args, toggleArg) {
					toggleWindow() // a shortcut of the desktop (Hyprland) called it
				} else {
					showWindow()
				}
			},
		},
	})
	trayOn = func() bool { return studio.Prefs().Tray }
	hidden := startHidden(os.Args) && studio.Prefs().Tray
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Blup Studio",
		Width:            1360,
		Height:           860,
		MinWidth:         900,
		MinHeight:        560,
		BackgroundColour: application.NewRGB(250, 250, 250),
		URL:              "/",
		Hidden:           hidden, // opened with the computer: straight to the tray
	})
	setupDesktop(studio, app, win)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
