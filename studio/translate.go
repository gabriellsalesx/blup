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

func init() { application.RegisterEvent[TranslateProgress]("translate") }

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
