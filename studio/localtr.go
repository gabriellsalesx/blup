package main

import (
	"context"
	"net/http"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/localtr"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The translation on the computer (Firefox's engine and models, internal/
// localtr): downloaded on demand; the window runs it in a worker
// (public/bergamot) and saves each translation through SaveTranslation.

// LocalTranslateState tells whether the local translation is installed
// and how much it downloads.
type LocalTranslateState struct {
	Installed bool  `json:"installed"`
	Bytes     int64 `json:"bytes"`
}

// TranslateInstallProgress is how much of the local translation came.
type TranslateInstallProgress struct {
	Done  int64 `json:"done"`
	Total int64 `json:"total"`
}

func init() {
	application.RegisterEvent[TranslateInstallProgress]("translateInstall")
}

func localTranslateDir() string {
	d, _ := app.LocalTranslateDir()
	return d
}

// serveLocalTranslate gives the window the engine and models.
func serveLocalTranslate(next http.Handler) http.Handler {
	return localtr.Serve(localTranslateDir, next)
}

// LocalTranslate is the local translation's state.
func (s *Studio) LocalTranslate() LocalTranslateState {
	return LocalTranslateState{Installed: localtr.Installed(localTranslateDir()), Bytes: localtr.DownloadSize()}
}

// InstallLocalTranslate downloads, checks and unpacks the local
// translation; progress goes out as "translateInstall" events.
func (s *Studio) InstallLocalTranslate() error {
	return localtr.Install(context.Background(), localTranslateDir(), func(done, total int64) {
		application.Get().Event.Emit("translateInstall", TranslateInstallProgress{done, total})
	})
}

// RemoveLocalTranslate deletes the local translation.
func (s *Studio) RemoveLocalTranslate() error { return localtr.Remove(localTranslateDir()) }

// SaveTranslation saves a translation made in the window and returns its
// path; props are the translated texts of the note's other properties (as
// TranslationTexts listed them). follow brings the preview up to date
// (false while a whole site is being translated: FollowTranslations at the
// end).
func (s *Studio) SaveTranslation(rel, lang, title, description, body string, props map[string][]string, follow bool) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	tr := map[string]any{}
	for k, v := range props {
		tr[k] = v
	}
	out, err := app.WriteTranslation(site, rel, lang, title, description, body, tr)
	if err == nil && follow {
		s.previewFollows(site)
	}
	return out, err
}

// TranslationTodo lists the open site's notes to translate into lang.
func (s *Studio) TranslationTodo(lang string) ([]string, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.TranslationTodo(site, lang)
}

// TranslationTexts are the texts of a note's properties a translation
// translates too (the menu's labels, the site's description...).
func (s *Studio) TranslationTexts(rel string) (map[string][]string, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	n, err := app.ReadSiteNote(site, rel)
	if err != nil {
		return nil, err
	}
	return app.PropTexts(n), nil
}

// MarkReviewed marks a translation reviewed (the site stops showing it was
// made by AI), or not.
func (s *Studio) MarkReviewed(rel string, reviewed bool) error {
	site, err := s.current()
	if err != nil {
		return err
	}
	return app.MarkReviewed(site, rel, reviewed)
}
