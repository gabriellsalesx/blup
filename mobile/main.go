package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "Blup Notes",
		Description: "Blup Notes: notas, desenhos e listas",
		Services:    []application.Service{application.NewService(&NotesService{}), application.NewService(&CloudService{}), application.NewService(&CalendarService{}), application.NewService(&SettingsService{}), application.NewService(&SpaceService{}), application.NewService(&BoardService{})},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		// Um pânico vai para o Blup Pulse quando a pessoa deixou (errors.go).
		PanicHandler: onPanic,
		OnShutdown:   func() { startReports(false) },
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Blup Notes",
		Width:            412,
		Height:           880,
		BackgroundColour: application.NewRGB(9, 9, 11),
		URL:              "/",
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
