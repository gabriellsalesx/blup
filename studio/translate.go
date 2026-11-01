package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/scaffold"
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
	application.RegisterEvent[string]("previewRestarted")
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
	out, err := app.TranslateNote(context.Background(), site, rel, lang, func(done, total int) {
		application.Get().Event.Emit("translate", TranslateProgress{rel, lang, done, total})
	})
	if err == nil {
		s.previewFollows(site)
	}
	return out, err
}

// previewConfig is the preview's generated config now.
func (s *Studio) previewConfig(site *config.Site) []byte {
	b, _ := os.ReadFile(filepath.Join(site.Dir(), scaffold.PreviewConfig))
	return b
}

// previewFollows writes the site's config again after translations; when
// it is not the one the running preview started with (a new language, a
// new language folder, which Hugo only reads at start), the preview starts
// again, and the window hears its new address ("previewRestarted").
func (s *Studio) previewFollows(site *config.Site) {
	if scaffold.Write(site) != nil {
		return
	}
	now := s.previewConfig(site)
	s.mu.Lock()
	stale := s.preview != nil && !bytes.Equal(s.previewCfg, now)
	s.mu.Unlock()
	if !stale {
		return
	}
	s.StopPreview()
	if url, err := s.StartPreview(); err == nil {
		application.Get().Event.Emit("previewRestarted", url)
	}
}

// FollowTranslations brings the preview up to date after translations
// saved one by one (the local translation of a whole site).
func (s *Studio) FollowTranslations() {
	if site, err := s.current(); err == nil {
		s.previewFollows(site)
	}
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
	n, err := app.TranslateSite(context.Background(), site, lang, func(done, total int, note string) {
		application.Get().Event.Emit("translateSite", TranslateSiteProgress{lang, done, total, note})
	})
	s.previewFollows(site)
	return n, err
}
