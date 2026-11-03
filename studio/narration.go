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

// VoiceProgress is how much of the local voice was downloaded.
type VoiceProgress struct {
	Done  int64  `json:"done"`
	Total int64  `json:"total"`
	Step  string `json:"step"` // runtime, tools, model, voices
}

func init() {
	application.RegisterEvent[NarrateProgress]("narrate")
	application.RegisterEvent[VoiceProgress]("voiceInstall")
}

// LocalVoice tells whether the local voice (Kokoro, on the computer) can
// be installed here, whether it is, and its download size.
func (s *Studio) LocalVoice() app.LocalVoice { return app.LocalVoiceState() }

// InstallLocalVoice downloads the local voice (engine, tools, model and
// voices), checks each file and sets it up; progress goes out as
// "voiceInstall" events.
func (s *Studio) InstallLocalVoice(ctx context.Context) error {
	return app.InstallLocalVoice(ctx, func(done, total int64, step string) {
		application.Get().Event.Emit("voiceInstall", VoiceProgress{done, total, step})
	})
}

// RemoveLocalVoice deletes the local voice.
func (s *Studio) RemoveLocalVoice() error { return app.RemoveLocalVoice() }

// AIConnected tells whether an OpenRouter key is saved.
func (s *Studio) AIConnected() (bool, error) { return app.AIConnected() }

// ConnectAI checks an OpenRouter key and saves it for every site.
func (s *Studio) ConnectAI(ctx context.Context, key string) error {
	return app.ConnectOpenRouter(ctx, key)
}

// OpenAIKeyPage opens OpenRouter's page of keys in the browser.
func (s *Studio) OpenAIKeyPage() { s.OpenURL(narrate.KeyPage) }

// Voices lists the narration voices for sites in lang ("pt", "en" or "es"): the
// local ones, then those of the OpenRouter model ("" for the recommended).
func (s *Studio) Voices(ctx context.Context, lang, model, blupModel string) ([]app.Voice, error) {
	return app.NarrationVoices(ctx, lang, model, blupModel)
}

// VoiceModels lists OpenRouter's voice models that can read sites in lang.
func (s *Studio) VoiceModels(ctx context.Context, lang string) ([]app.ModelOption, error) {
	return app.NarrationModels(ctx, lang)
}

// PreviewVoice is a sample of an OpenRouter voice without a recorded one
// (made once, then kept). Base64 MP3 in the JSON answer.
func (s *Studio) PreviewVoice(ctx context.Context, lang, choice string) ([]byte, error) {
	return app.PreviewVoice(ctx, lang, choice)
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

// Narrate reads the note aloud, with the voice chosen for its language
// (the site's, or its translation's), and saves the MP3 in the attachments.
func (s *Studio) Narrate(ctx context.Context, rel string) (app.Narration, error) {
	site, err := s.current()
	if err != nil {
		return app.Narration{}, err
	}
	_, lang := app.NoteOrigin(site, rel)
	voice := s.Prefs().Voices[lang]
	return app.Narrate(ctx, site, rel, voice, func(done, total int) {
		application.Get().Event.Emit("narrate", NarrateProgress{rel, done, total})
	})
}

// NarrationEstimate is what narrating the note would use of the month's AI
// allowance, when the site's voice is read through Blup Cloud (nil
// otherwise).
func (s *Studio) NarrationEstimate(ctx context.Context, rel string) (*app.NarrationCost, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	_, lang := app.NoteOrigin(site, rel)
	if !app.NarratesThroughBlup(lang, s.Prefs().Voices[lang]) {
		return nil, nil
	}
	c, err := app.EstimateNarration(ctx, site, rel, s.Prefs().Voices[lang])
	if err != nil {
		return nil, err
	}
	return &c, nil
}
