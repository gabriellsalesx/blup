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
// each language with the voice chosen for it.

// NarrateSiteJob is the batch running now (Running false when none).
type NarrateSiteJob struct {
	Running bool   `json:"running"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Note    string `json:"note"`
}

// NarrateSiteProgress is how far the batch went, heard as "narrateSite".
type NarrateSiteProgress struct {
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Note  string `json:"note"`
}

// NarrateSiteDone is how the batch ended, heard as "narrateSiteDone".
type NarrateSiteDone struct {
	Done      int      `json:"done"`
	Failed    []string `json:"failed"`
	Error     string   `json:"error"`
	Cancelled bool     `json:"cancelled"`
}

var narrJob struct {
	sync.Mutex
	st     NarrateSiteJob
	cancel context.CancelFunc
}

func init() {
	application.RegisterEvent[NarrateSiteProgress]("narrateSite")
	application.RegisterEvent[NarrateSiteDone]("narrateSiteDone")
}

// NarrationPlan is what narrating the open blog's posts would do, with
// what it would use of the month's AI allowance when some voice goes
// through Blup Cloud.
func (s *Studio) NarrationPlan(ctx context.Context) (app.NarrationPlan, error) {
	site, err := s.current()
	if err != nil {
		return app.NarrationPlan{}, err
	}
	voices := s.Prefs().Voices
	plan, err := app.NarrationTodo(site, voices)
	if err != nil {
		return plan, err
	}
	cost, err := app.NarrationPlanCost(ctx, site, plan, voices)
	if err != nil {
		plan.CostError = err.Error()
	}
	plan.Cost = cost
	return plan, nil
}

// NarrateSiteState is the batch now: a window that opens again picks it up.
func (s *Studio) NarrateSiteState() NarrateSiteJob {
	narrJob.Lock()
	defer narrJob.Unlock()
	return narrJob.st
}

// CancelNarrateSite stops the batch after the note being read (the notes
// done keep their narration).
func (s *Studio) CancelNarrateSite() {
	narrJob.Lock()
	defer narrJob.Unlock()
	if narrJob.cancel != nil {
		narrJob.cancel()
	}
}

// StartNarrateSite narrates the open blog's posts that have no narration
// or an out of date one, in the background. What would stop every note
// (no voice, no key, the consent, an allowance too small) comes back now;
// the progress goes out as "narrateSite" and the end as
// "narrateSiteDone". Starting again while it runs only keeps it going.
func (s *Studio) StartNarrateSite(ctx context.Context) error {
	site, err := s.current()
	if err != nil {
		return err
	}
	if s.NarrateSiteState().Running {
		return nil
	}
	if s.TranslateSiteState().Running {
		return app.ErrBatchTranslating(site)
	}
	voices := s.Prefs().Voices
	plan, err := app.NarrationTodo(site, voices)
	if err != nil {
		return err
	}
	if len(plan.Items) == 0 {
		return nil
	}
	if err := app.NarrateReady(site, plan, voices); err != nil {
		return err
	}
	cost, err := app.NarrationPlanCost(ctx, site, plan, voices)
	if err != nil {
		return err
	}
	if cost != nil && !cost.Enough {
		return app.ErrBatchAllowance(site, *cost)
	}
	jctx, cancel := context.WithCancel(context.Background())
	narrJob.Lock()
	if narrJob.st.Running { // another start took it meanwhile
		narrJob.Unlock()
		cancel()
		return nil
	}
	narrJob.st = NarrateSiteJob{Running: true, Total: len(plan.Items)}
	narrJob.cancel = cancel
	narrJob.Unlock()
	go func() {
		defer cancel()
		res, err := app.NarrateSite(jctx, site, plan, voices, func(done, total int, note string) {
			narrJob.Lock()
			narrJob.st = NarrateSiteJob{Running: true, Done: done, Total: total, Note: note}
			narrJob.Unlock()
			application.Get().Event.Emit("narrateSite", NarrateSiteProgress{done, total, note})
		})
		end := NarrateSiteDone{Done: res.Done, Failed: res.Failed}
		switch {
		case errors.Is(err, context.Canceled):
			end.Cancelled = true
		case err != nil:
			end.Error = err.Error()
		}
		narrJob.Lock()
		narrJob.st = NarrateSiteJob{}
		narrJob.cancel = nil
		narrJob.Unlock()
		application.Get().Event.Emit("narrateSiteDone", end)
	}()
	return nil
}
