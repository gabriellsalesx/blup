// Command blup turns an Obsidian vault into a fast Hugo site and
// publishes it, guiding the user step by step.
package main

import (
	"context"
	"os"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/ui"
)

func main() {
	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		if ui.Aborted(err) {
			os.Exit(130)
		}
		ui.Error(err.Error())
		os.Exit(1)
	}
}
