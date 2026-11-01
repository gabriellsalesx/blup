package main

import (
	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
)

// What the settings window shows about Blup itself.

// Version is the Studio's version (set at build time, like the CLI's).
func (s *Studio) Version() string { return app.Version }

// SetLanguage changes the language of Blup and the Studio ("pt", "en" or "es").
// The window reloads to show it.
func (s *Studio) SetLanguage(lang string) error {
	if lang != "pt" && lang != "en" && lang != "es" {
		lang = "en"
	}
	g, err := config.LoadGlobal()
	if err != nil {
		return err
	}
	g.Lang = lang
	if err := g.Save(); err != nil {
		return err
	}
	i18n.Set(i18n.Lang(lang))
	s.lang = lang
	return nil
}
