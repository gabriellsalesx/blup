package main

import (
	"context"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/narrate"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The AI of the Studio: the OpenRouter key (one for every site) and the
// narration of posts, whose progress goes out as "narrate" events.

// NarrateProgress is how many parts of the note were read aloud.
type NarrateProgress struct {
	Note  string `json:"note"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
}

func init() { application.RegisterEvent[NarrateProgress]("narrate") }

// AIConnected tells whether an OpenRouter key is saved.
func (s *Studio) AIConnected() (bool, error) { return app.AIConnected() }

// ConnectAI checks an OpenRouter key and saves it for every site.
func (s *Studio) ConnectAI(key string) error {
	return app.ConnectOpenRouter(context.Background(), key)
}

// OpenAIKeyPage opens OpenRouter's page of keys in the browser.
func (s *Studio) OpenAIKeyPage() { s.OpenURL(narrate.KeyPage) }

// Voices lists the narration voices for sites in lang ("pt" or "en"), the
// default first.
func (s *Studio) Voices(lang string) ([]app.Voice, error) {
	return app.NarrationVoices(context.Background(), lang)
}

// NarrationHash is the fingerprint of what the note reads now (compared
// with its audio_hash to tell an out of date narration).
func (s *Studio) NarrationHash(rel string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	return app.NarrationHash(site, rel)
}

// Narrate reads the note aloud, with the voice chosen for the site's
// language, and saves the MP3 in the attachments.
func (s *Studio) Narrate(rel string) (app.Narration, error) {
	site, err := s.current()
	if err != nil {
		return app.Narration{}, err
	}
	voice := s.Prefs().Voices[site.Lang]
	return app.Narrate(context.Background(), site, rel, voice, func(done, total int) {
		application.Get().Event.Emit("narrate", NarrateProgress{rel, done, total})
	})
}
