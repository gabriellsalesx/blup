package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
)

// Live sync of a site through Blup Cloud, for plans that have it: the notes
// are sent a moment after they are saved, received as soon as another
// computer saves (the account's live room), and checked every few minutes
// in case a message was missed. Plans without it sync by button.

// LiveState is reported to the window.
type LiveState struct {
	State string    `json:"state"` // "syncing", "synced", "error" or "gone" (deleted on another device)
	At    time.Time `json:"at"`
	Error string    `json:"error,omitempty"`
	// Stuck: the error is a sync stopped for safety (ErrSyncStuck); the
	// Sync screen offers "Compare from scratch".
	Stuck bool     `json:"stuck,omitempty"`
	Info  SyncInfo `json:"info"`
}

// Live is the live sync of one open site.
type Live struct {
	site   *config.Site
	report func(LiveState)
	ctx    context.Context // ends with Stop
	cancel context.CancelFunc

	mu      sync.Mutex
	timer   *time.Timer
	syncing bool
	again   bool      // a change arrived during a sync
	quiet   time.Time // our own downloads end before this
	gone    bool      // the vault was deleted elsewhere: stopped for good
}

// Gone tells whether it stopped because the vault was deleted elsewhere.
func (l *Live) Gone() bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.gone
}

// liveDelay is how long after the last saved change the notes are sent.
const liveDelay = 2 * time.Second

// StartLive starts the live sync of site when it syncs through Blup Cloud
// on a live plan; otherwise it returns nil. Stop it with Stop.
func StartLive(site *config.Site, report func(LiveState)) *Live {
	if site.Sync.Remote != RemoteBlup || site.Sync.Vault == "" {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	l := &Live{site: site, report: report, ctx: ctx, cancel: cancel}
	go l.run(ctx)
	return l
}

// Stop ends the live sync.
func (l *Live) Stop() {
	if l == nil {
		return
	}
	l.cancel()
	l.mu.Lock()
	if l.timer != nil {
		l.timer.Stop()
	}
	l.mu.Unlock()
}

// Changed is told when notes change on disk: they are sent after liveDelay
// without other changes. Changes written by the sync itself are ignored.
func (l *Live) Changed() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.syncing {
		l.again = true
		return
	}
	// Stopped: a change no longer starts a sync (nor one cut short by
	// Stop, which ends l.ctx).
	if time.Now().Before(l.quiet) || l.ctx.Err() != nil {
		return
	}
	if l.timer != nil {
		l.timer.Stop()
	}
	l.timer = time.AfterFunc(liveDelay, func() { l.sync(l.ctx) })
}

func (l *Live) run(ctx context.Context) {
	c, _, err := blupClient()
	if err != nil {
		return
	}
	// Offline at the start: asked again with growing pauses, so the live
	// sync begins once the network is back. Signed out or a plan without
	// it: by button.
	var me blupcloud.Account
	for wait := 10 * time.Second; ; wait = min(wait*2, 5*time.Minute) {
		if me, err = c.Me(ctx); err == nil || errors.Is(err, blupcloud.ErrSignedOut) {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
	if err != nil || !me.Plan.Limits.Live {
		return // by button
	}
	l.sync(ctx) // what changed while the Studio was closed
	go l.listen(ctx, c)
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			l.sync(ctx)
		}
	}
}

// listen keeps the live room open, reconnecting with growing pauses.
func (l *Live) listen(ctx context.Context, c *blupcloud.Client) {
	wait := 5 * time.Second
	for ctx.Err() == nil {
		start := time.Now()
		err := c.Listen(ctx, func(m blupcloud.Moved) {
			if m.Vault == l.site.Sync.Vault {
				l.sync(ctx)
			}
		})
		if errors.Is(err, blupcloud.ErrSignedOut) || errors.Is(err, blupcloud.ErrNotLive) {
			return
		}
		if time.Since(start) > time.Minute {
			wait = 5 * time.Second // it had been working
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = min(wait*2, 2*time.Minute)
	}
}

// sync runs one sync at a time; a change during it means one more after.
func (l *Live) sync(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	l.mu.Lock()
	if l.syncing {
		l.again = true
		l.mu.Unlock()
		return
	}
	l.syncing = true
	l.mu.Unlock()
	for {
		l.report(LiveState{State: "syncing", At: time.Now()})
		info, err := syncSite(ctx, l.site, true)
		st := LiveState{State: "synced", At: time.Now(), Info: info}
		if err != nil {
			st = LiveState{State: "error", At: time.Now(), Error: err.Error(), Stuck: errors.Is(err, ErrSyncStuck)}
		}
		if errors.Is(err, ErrVaultGone) {
			st.State = "gone"
			l.mu.Lock()
			l.gone, l.syncing = true, false
			l.mu.Unlock()
			l.report(st)
			l.Stop()
			return
		}
		l.report(st)
		l.mu.Lock()
		// The notes it wrote show up on disk a moment later.
		l.quiet = time.Now().Add(1500 * time.Millisecond)
		if !l.again || ctx.Err() != nil {
			l.syncing = false
			l.mu.Unlock()
			return
		}
		l.again = false
		l.mu.Unlock()
	}
}
