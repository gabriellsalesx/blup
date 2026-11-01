package main

import (
	"context"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// TranslateProgress is how far the translation of a note went (parts done
// of total), heard by the window as "translate".
type TranslateProgress struct {
	Note  string `json:"note"`
	Lang  string `json:"lang"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
}

func init() {
	application.RegisterEvent[TranslateProgress]("translate")
	application.RegisterEvent[TranslateSiteProgress]("translateSite")
}

// NoteLangs lists the note's languages: the original, then each other
// site language, translated (and up to date) or not.
func (s *Studio) NoteLangs(rel string) ([]app.NoteLang, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.NoteLangs(site, rel)
}

// TranslateNote translates the note (or the original of a translation)
// into lang with AI and returns the translation's path.
func (s *Studio) TranslateNote(rel, lang string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	return app.TranslateNote(context.Background(), site, rel, lang, func(done, total int) {
		application.Get().Event.Emit("translate", TranslateProgress{rel, lang, done, total})
	})
}

// TranslateSiteProgress is how far the translation of the whole site went.
type TranslateSiteProgress struct {
	Lang  string `json:"lang"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Note  string `json:"note"`
}

// SiteLangs tells how much of the open site is in each other language.
func (s *Studio) SiteLangs() ([]app.SiteLangState, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.SiteLangs(site)
}

// TranslateSite translates the open site's notes that miss a translation
// into lang (or have an out of date one) and returns how many it did.
func (s *Studio) TranslateSite(lang string) (int, error) {
	site, err := s.current()
	if err != nil {
		return 0, err
	}
	return app.TranslateSite(context.Background(), site, lang, func(done, total int, note string) {
		application.Get().Event.Emit("translateSite", TranslateSiteProgress{lang, done, total, note})
	})
}
