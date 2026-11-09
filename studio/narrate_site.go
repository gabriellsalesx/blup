package main

import (
	"context"
	"errors"
	"sync"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Narrating every post of the open blog at once (the Podcast screen), in
// the background like the site's translation: one batch at a time, going
// on while the person works on other screens (the side strip shows it),
// each language with the voice chosen for it. The batch belongs to its
// site: the window shows it only while that site is open.

// NarrateSiteJob is the batch now (Running false when none). Preparing:
// it was claimed and checks its notes, voices and cost before the first
// note (nothing to stop yet).
type NarrateSiteJob struct {
	Running   bool   `json:"running"`
	Preparing bool   `json:"preparing"`
	Site      string `json:"site"` // the folder of the site being narrated
	Done      int    `json:"done"`
	Total     int    `json:"total"`
	Note      string `json:"note"`
}

// NarrateSiteProgress is how far the batch went, heard as "narrateSite".
type NarrateSiteProgress struct {
	Site  string `json:"site"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Note  string `json:"note"`
}

// NarrateSiteDone is how the batch ended, heard as "narrateSiteDone".
type NarrateSiteDone struct {
	Site      string                 `json:"site"`
	Done      int                    `json:"done"`
	Skipped   int                    `json:"skipped"`
	Failed    []app.NarrationFailure `json:"failed"`
	Error     string                 `json:"error"`
	Cancelled bool                   `json:"cancelled"`
}

var narrJob struct {
	sync.Mutex
	st     NarrateSiteJob
	cancel context.CancelFunc
}

// jobsMu is taken to claim either background job of the notes (the
// site's translation, the narration of its posts): one claims while the
// other can't, before any slow check, so the two never run together.
var jobsMu sync.Mutex

// errNarratingNow / errTranslatingNow: the other background job holds the
// notes (the screen gets the message in the site's language).
var (
	errNarratingNow   = errors.New("narrate: the posts are being narrated")
	errTranslatingNow = errors.New("translate: the site is being translated")
)

func init() {
	application.RegisterEvent[NarrateSiteProgress]("narrateSite")
	application.RegisterEvent[NarrateSiteDone]("narrateSiteDone")
}

func narrateRunning() bool {
	narrJob.Lock()
	defer narrJob.Unlock()
	return narrJob.st.Running
}

// claimNarrJob takes the background for the narration of site's posts:
// false when a narration runs already (taken meanwhile), with
// errTranslatingNow when the translation holds it.
func claimNarrJob(site string, cancel context.CancelFunc) (bool, error) {
	jobsMu.Lock()
	defer jobsMu.Unlock()
	siteJob.Lock()
	translating := siteJob.st.Running
	siteJob.Unlock()
	if translating {
		return false, errTranslatingNow
	}
	narrJob.Lock()
	defer narrJob.Unlock()
	if narrJob.st.Running {
		return false, nil
	}
	narrJob.st = NarrateSiteJob{Running: true, Preparing: true, Site: site}
	narrJob.cancel = cancel
	return true, nil
}

func releaseNarrJob() {
	narrJob.Lock()
	defer narrJob.Unlock()
	narrJob.st = NarrateSiteJob{}
	narrJob.cancel = nil
}

// NarrationPlan is what narrating the open blog's posts would do, with
// what it would use of the month's AI allowance when some voice goes
// through Blup Cloud, and of the person's OpenRouter key when it is theirs.
// voices are the narration's voices as the window has them now (it saves
// them a moment later); nil: the saved ones.
func (s *Studio) NarrationPlan(ctx context.Context, voices map[string]string) (app.NarrationPlan, error) {
	site, err := s.current()
	if err != nil {
		return app.NarrationPlan{}, err
	}
	if voices == nil {
		voices = s.Prefs().Voices
	}
	plan, err := app.NarrationTodo(site, voices)
	if err != nil {
		return plan, err
	}
	cost, err := app.NarrationPlanCost(ctx, site, plan, voices)
	if err != nil {
		plan.CostError = err.Error()
	}
	plan.Cost = cost
	app.NarrationOpenRouterCost(ctx, &plan, voices)
	app.NarrationVoiceNames(ctx, &plan)
	plan.Made, plan.MadeLangs, _ = app.NarrationsMade(site)
	return plan, nil
}

// jobsRunning tells whether a background job (the translation, the narration) holds
// the notes.
func jobsRunning() bool {
	siteJob.Lock()
	translating := siteJob.st.Running
	siteJob.Unlock()
	return translating || narrateRunning()
}

// RemoveNarrations takes the narrations the Studio made off the open
// blog's posts (translations included; an audio of the person's own
// stays): the MP3s go to the Blup trash, together. Refused while the
// site's translation or a narration runs. RestoreNarrations undoes it.
func (s *Studio) RemoveNarrations() (app.RemovedNarrations, error) {
	site, err := s.current()
	if err != nil {
		return app.RemovedNarrations{}, err
	}
	jobsMu.Lock()
	defer jobsMu.Unlock()
	if jobsRunning() {
		return app.RemovedNarrations{}, app.ErrJobsRunning(site)
	}
	return app.RemoveNarrations(site)
}

// RestoreNarrations puts back what RemoveNarrations took off (trash: the
// folder it gave).
func (s *Studio) RestoreNarrations(trash string) error {
	site, err := s.current()
	if err != nil {
		return err
	}
	jobsMu.Lock()
	defer jobsMu.Unlock()
	if jobsRunning() {
		return app.ErrJobsRunning(site)
	}
	return app.RestoreNarrations(site, trash)
}

// NarrateSiteState is the batch now: a window that opens again picks it up.
func (s *Studio) NarrateSiteState() NarrateSiteJob {
	narrJob.Lock()
	defer narrJob.Unlock()
	return narrJob.st
}

// CancelNarrateSite stops the batch now: the note being read is left as
// it was, the notes done keep their narration.
func (s *Studio) CancelNarrateSite() {
	narrJob.Lock()
	defer narrJob.Unlock()
	if narrJob.cancel != nil && !narrJob.st.Preparing {
		narrJob.cancel()
	}
}

// StartNarrateSite narrates the open blog's posts that have no narration
// or an out of date one, in the background. The background is claimed
// first (never with the site's translation); what would stop every note
// (no voice, no key, the consent, an allowance too small) comes back now
// and frees it. The progress goes out as "narrateSite" and the end as
// "narrateSiteDone". Starting again while it runs only keeps it going.
func (s *Studio) StartNarrateSite(ctx context.Context) error {
	site, err := s.current()
	if err != nil {
		return err
	}
	jctx, cancel := context.WithCancel(jobContext())
	ok, err := claimNarrJob(site.Dir(), cancel)
	if !ok {
		cancel()
		if errors.Is(err, errTranslatingNow) {
			return app.ErrBatchTranslating(site)
		}
		return err
	}
	fail := func(err error) error {
		releaseNarrJob()
		cancel()
		return err
	}
	voices := s.Prefs().Voices
	plan, err := app.NarrationTodo(site, voices)
	if err != nil {
		return fail(err)
	}
	if len(plan.Items) == 0 {
		return fail(nil)
	}
	if err := app.NarrateReady(site, plan, voices); err != nil {
		return fail(err)
	}
	cost, err := app.NarrationPlanCost(ctx, site, plan, voices)
	if err != nil {
		return fail(err)
	}
	if cost != nil && !cost.Enough {
		return fail(app.ErrBatchAllowance(site, *cost))
	}
	narrJob.Lock()
	narrJob.st = NarrateSiteJob{Running: true, Site: site.Dir(), Total: len(plan.Items)}
	narrJob.Unlock()
	goJob(func() {
		defer cancel()
		res, err := app.NarrateSite(jctx, site, plan, voices, func(done, total int, note string) {
			narrJob.Lock()
			narrJob.st = NarrateSiteJob{Running: true, Site: site.Dir(), Done: done, Total: total, Note: note}
			narrJob.Unlock()
			application.Get().Event.Emit("narrateSite", NarrateSiteProgress{site.Dir(), done, total, note})
		})
		end := NarrateSiteDone{Site: site.Dir(), Done: res.Done, Skipped: res.Skipped, Failed: res.Failed}
		switch {
		case errors.Is(err, context.Canceled):
			end.Cancelled = true
		case err != nil:
			end.Error = err.Error()
		}
		releaseNarrJob()
		application.Get().Event.Emit("narrateSiteDone", end)
	})
	return nil
}
