package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

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
func (s *Studio) TranslateNote(ctx context.Context, rel, lang string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	out, err := app.TranslateNote(ctx, site, rel, lang, translateProvider(s.Prefs()), func(done, total int) {
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
	if stale {
		s.restartPreview()
	}
}

// restartPreview starts the running preview again, and the window hears
// its new address ("previewRestarted").
// Restarts come one at a time, and only while the preview runs: one
// stopped (or another site opened) meanwhile is not started again.
func (s *Studio) restartPreview() {
	s.restarting.Lock()
	defer s.restarting.Unlock()
	s.mu.Lock()
	site, before, running := s.site, s.url, s.preview != nil
	s.mu.Unlock()
	if !running {
		return
	}
	s.StopPreview()
	url, err := s.startPreview(site)
	if err != nil {
		return
	}
	// Same address: the window only reloads the page it shows.
	if url == before {
		application.Get().Event.Emit("previewChanged", "")
		return
	}
	application.Get().Event.Emit("previewRestarted", url)
}

// sectionsChanged restarts a running preview after a section folder came
// or went: Hugo's watcher misses the notes of a folder created at once
// (it watches the folder only after its files are written), so the new
// section showed up only on a later change.
func (s *Studio) sectionsChanged() {
	go s.restartPreview()
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

// TranslateSiteJob is the site's translation running in the background:
// one at a time, going on while the person works on other screens (the
// window shows it in the side strip).
type TranslateSiteJob struct {
	Running bool   `json:"running"`
	Lang    string `json:"lang"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Note    string `json:"note"`
}

// TranslateSiteDone is how a background translation ended, heard by the
// window as "translateSiteDone".
type TranslateSiteDone struct {
	Lang    string   `json:"lang"`
	Done    int      `json:"done"`
	Skipped int      `json:"skipped"`
	Failed  []string `json:"failed"`
	Error   string   `json:"error"`
}

var siteJob struct {
	sync.Mutex
	st TranslateSiteJob
}

func init() {
	application.RegisterEvent[TranslateSiteDone]("translateSiteDone")
}

// TranslateSiteState is the background translation now (Running false
// when there is none): a window that opens the screen again picks it up.
func (s *Studio) TranslateSiteState() TranslateSiteJob {
	siteJob.Lock()
	defer siteJob.Unlock()
	return siteJob.st
}

// StartTranslateSite starts translating the open site's notes that miss a
// translation into lang (or have an out of date one) with the AI in the
// cloud chosen, in the background. What would stop every note (no key, no
// AI in the plan, no consent) comes back now; the progress goes out as
// "translateSite" and the end as "translateSiteDone". Starting the same
// language again while it runs only keeps it going: never two batches.
func (s *Studio) StartTranslateSite(ctx context.Context, lang string) error {
	site, err := s.current()
	if err != nil {
		return err
	}
	if running, err := siteJobBusy(lang); running || err != nil {
		return err
	}
	if s.NarrateSiteState().Running {
		return app.ErrBatchNarrating(site)
	}
	provider := translateProvider(s.Prefs())
	if err := app.TranslateReady(ctx, provider); err != nil {
		return err
	}
	// Checked again: another start may have taken the job meanwhile.
	if ok, err := claimSiteJob(lang); !ok {
		return err
	}
	go func() {
		res, err := app.TranslateSite(context.Background(), site, lang, provider, func(done, total int, note string) {
			siteJob.Lock()
			siteJob.st = TranslateSiteJob{Running: true, Lang: lang, Done: done, Total: total, Note: note}
			siteJob.Unlock()
			application.Get().Event.Emit("translateSite", TranslateSiteProgress{lang, done, total, note})
		})
		s.previewFollows(site)
		end := TranslateSiteDone{Lang: lang, Done: res.Done, Skipped: res.Skipped, Failed: res.Failed}
		if err != nil {
			end.Error = err.Error()
		}
		siteJob.Lock()
		siteJob.st = TranslateSiteJob{}
		siteJob.Unlock()
		application.Get().Event.Emit("translateSiteDone", end)
	}()
	return nil
}

var errOtherLang = errors.New("translate: another language is being translated")

// siteJobBusy tells whether the site's translation runs already: with no
// error when in lang (starting again only keeps it going), with
// errOtherLang when in another language.
func siteJobBusy(lang string) (bool, error) {
	siteJob.Lock()
	defer siteJob.Unlock()
	if !siteJob.st.Running {
		return false, nil
	}
	if siteJob.st.Lang != lang {
		return true, errOtherLang
	}
	return true, nil
}

// claimSiteJob takes the job for lang. False when it runs already (taken
// by another start meanwhile), with errOtherLang when in another
// language: never two batches, and never a silent "started" for a
// language that is not the one running.
func claimSiteJob(lang string) (bool, error) {
	siteJob.Lock()
	defer siteJob.Unlock()
	if siteJob.st.Running {
		if siteJob.st.Lang != lang {
			return false, errOtherLang
		}
		return false, nil
	}
	siteJob.st = TranslateSiteJob{Running: true, Lang: lang}
	return true, nil
}

// translateProvider is the AI in the cloud the person chose to translate
// ("openrouter" or "blup"); "" when none was (the translation on the
// computer, or no choice yet). The older "cloud" let the Studio pick by
// itself, so it counts as no choice: the person picks again.
func translateProvider(p Prefs) string {
	switch p.TranslateAI {
	case app.AIOpenRouter, app.AIBlup:
		return p.TranslateAI
	}
	return ""
}

// RemovedTranslation is a language taken off the site: where each of its
// folders went (to undo), and the notes now.
type RemovedTranslation struct {
	Trash []string `json:"trash"`
	Tree  Node     `json:"tree"`
}

// RemoveTranslation takes lang off the open site: its translations go to
// the Blup trash (restorable) and the site stops publishing it. Refused
// while lang is being translated.
func (s *Studio) RemoveTranslation(lang string) (RemovedTranslation, error) {
	site, err := s.current()
	if err != nil {
		return RemovedTranslation{}, err
	}
	siteJob.Lock()
	busy := siteJob.st.Running && siteJob.st.Lang == lang
	siteJob.Unlock()
	if busy {
		return RemovedTranslation{}, errors.New("translate: this language is being translated")
	}
	trash, err := app.RemoveTranslation(site, lang)
	s.previewFollows(site)
	if err != nil {
		return RemovedTranslation{}, err
	}
	tree, err := s.Tree()
	return RemovedTranslation{trash, tree}, err
}

// RestoreTranslation brings back what RemoveTranslation took off.
func (s *Studio) RestoreTranslation(lang string, trash []string) (Node, error) {
	site, err := s.current()
	if err != nil {
		return Node{}, err
	}
	err = app.RestoreTranslation(site, lang, trash)
	s.previewFollows(site)
	if err != nil {
		return Node{}, err
	}
	return s.Tree()
}
