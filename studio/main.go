// Blup Studio: the Blup CLI in a window. Notes and folders on the left, a
// Markdown editor in the middle and the site's live preview on the right;
// everything is plain .md files on disk, so it works offline and the notes
// still open in Obsidian, VS Code or any editor.
package main

import (
	"embed"
	"log"
	"os"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// The Studio speaks the language chosen in Blup (blup language), else
	// the system's.
	lang := "en"
	if g, err := config.LoadGlobal(); err == nil && g.Lang != "" {
		lang = g.Lang
	} else if strings.HasPrefix(strings.ToLower(os.Getenv("LANG")), "pt") {
		lang = "pt"
	}
	i18n.Set(i18n.Lang(lang))
	studio := &Studio{lang: lang}
	app := application.New(application.Options{
		Name:        "Blup Studio",
		Description: "Write in Markdown, preview and publish your site",
		Services:    []application.Service{application.NewService(studio)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		OnShutdown:  studio.shutdown,
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Blup Studio",
		Width:            1360,
		Height:           860,
		MinWidth:         900,
		MinHeight:        560,
		BackgroundColour: application.NewRGB(250, 250, 250),
		URL:              "/",
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
