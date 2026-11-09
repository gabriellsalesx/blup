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
	// Dir is the site's folder: a window open on another site ignores it.
	Dir string `json:"dir"`
	// State: "syncing", "synced", "error", "gone" (deleted on another
	// device), "off" (stopped, or a plan without live sync: by button) or
	// "signedOut" (no Blup Cloud session).
	State string    `json:"state"`
	At    time.Time `json:"at"`
	Error string    `json:"error,omitempty"`
	// Stuck: the error is a sync stopped for safety (ErrSyncStuck); the
	// Sync screen offers "Compare from scratch".
	Stuck bool     `json:"stuck,omitempty"`
	Info  SyncInfo `json:"info"`
}

// LiveHooks are who hears a live sync: Report its states, Account (may be
// nil) each time the account's plan changes (what syncs live) or its
// session ends, as found by the periodic check.
type LiveHooks struct {
	Report  func(LiveState)
	Account func()
}

// Live is the live sync of one open site.
type Live struct {
	report  func(LiveState)
	account func()
	ctx     context.Context // ends with Stop
	cancel  context.CancelFunc
	// wake cuts short the wait of a plan without live sync (Recheck).
	wake chan struct{}

	mu      sync.Mutex
	site    *config.Site // swapped by SetSite (settings from another computer)
	timer   *time.Timer
	syncing bool
	again   bool      // a change arrived during a sync
	quiet   time.Time // our own downloads end before this
	gone    bool      // the vault was deleted elsewhere: stopped for good
	active  bool      // the plan syncs live now: changes on disk are sent
	idle    bool      // waiting on a plan without live sync
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

// Dir is the folder of the site it syncs ("" for nil).
func (l *Live) Dir() string {
	if l == nil {
		return ""
	}
	return l.siteNow().Dir()
}

// SetSite gives it the site as it is now (the same folder, its settings
// read again): the next syncs share these settings.
func (l *Live) SetSite(site *config.Site) {
	if l == nil || site == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.site.Dir() == site.Dir() {
		l.site = site
	}
}

func (l *Live) siteNow() *config.Site {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.site
}

// Recheck asks the account again now when it waits on a plan without
// live sync (the window came back into view, the plan may have changed in
// the browser).
func (l *Live) Recheck() {
	if l == nil {
		return
	}
	l.mu.Lock()
	idle := l.idle
	l.mu.Unlock()
	if !idle {
		return
	}
	select {
	case l.wake <- struct{}{}:
	default:
	}
}

// liveDelay is how long after the last saved change the notes are sent.
const liveDelay = 2 * time.Second

// livePlanCheck is how often a plan without live sync is asked again (a
// plan changed in the browser starts it).
var livePlanCheck = 3 * time.Minute

// StartLive starts the live sync of site when it syncs through Blup Cloud
// on a live plan; otherwise it returns nil. Stop it with Stop.
func StartLive(site *config.Site, report func(LiveState)) *Live {
	return StartLiveHooks(site, LiveHooks{Report: report})
}

// StartLiveHooks is StartLive, also telling h.Account of the plan's
// changes.
func StartLiveHooks(site *config.Site, h LiveHooks) *Live {
	if site.Sync.Remote != RemoteBlup || site.Sync.Vault == "" {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	l := &Live{site: site, report: h.Report, account: h.Account, ctx: ctx, cancel: cancel, wake: make(chan struct{}, 1)}
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
	l.active = false
	l.mu.Unlock()
}

// emit reports st with the site's folder; nothing once stopped.
func (l *Live) emit(ctx context.Context, st LiveState) {
	if ctx.Err() != nil || l.report == nil {
		return
	}
	st.Dir = l.siteNow().Dir()
	l.report(st)
}

func (l *Live) tellAccount() {
	if l.account != nil && l.ctx.Err() == nil {
		l.account()
	}
}

// Changed is told when notes change on disk: they are sent after liveDelay
// without other changes (plans with live sync only). Changes written by
// the sync itself are ignored.
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
	// Stop, which ends l.ctx). A plan without live sync sends by button.
	if time.Now().Before(l.quiet) || l.ctx.Err() != nil || !l.active {
		return
	}
	if l.timer != nil {
		l.timer.Stop()
	}
	l.timer = time.AfterFunc(liveDelay, func() { l.sync(l.ctx) })
}

// planOf is what of the account decides the live sync (and what the
// window shows of the plan).
type planOf struct {
	name    string
	live    bool
	history int
	ai      bool
}

func planKey(me blupcloud.Account) planOf {
	return planOf{me.Plan.Name, me.Plan.Limits.Live, me.Plan.Limits.HistoryDays, me.Plan.Limits.AIUsage > 0}
}

func (l *Live) run(ctx context.Context) {
	c, _, err := blupClient()
	if err != nil {
		if errors.Is(err, blupcloud.ErrSignedOut) {
			l.emit(ctx, LiveState{State: "signedOut", At: time.Now()})
		}
		return
	}
	var last *planOf
	off := false // "off" was said (not again every check)
	for ctx.Err() == nil {
		me, err := l.me(ctx, c)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, blupcloud.ErrSignedOut) {
			l.emit(ctx, LiveState{State: "signedOut", At: time.Now()})
			l.tellAccount()
			return
		}
		k := planKey(me)
		if last != nil && *last != k {
			l.tellAccount() // changed in the browser
		}
		last = &k
		if !k.live {
			// By button; asked again every few minutes and when the
			// window comes back into view (Recheck).
			if !off {
				l.emit(ctx, LiveState{State: "off", At: time.Now()})
				off = true
			}
			l.setIdle(true)
			select {
			case <-ctx.Done():
				return
			case <-time.After(livePlanCheck):
			case <-l.wake:
			}
			l.setIdle(false)
			continue
		}
		off = false
		l.liveLoop(ctx, c)
		// The live room said the plan lost it or the session ended:
		// the account is asked again at once.
	}
}

func (l *Live) setIdle(idle bool) {
	l.mu.Lock()
	l.idle = idle
	l.mu.Unlock()
}

// me asks for the account; offline, again with growing pauses, so the
// live sync begins once the network is back. It returns nil or
// ErrSignedOut (or ctx's end).
func (l *Live) me(ctx context.Context, c *blupcloud.Client) (blupcloud.Account, error) {
	for wait := 10 * time.Second; ; wait = min(wait*2, 5*time.Minute) {
		me, err := c.Me(ctx)
		if err == nil || errors.Is(err, blupcloud.ErrSignedOut) {
			return me, err
		}
		select {
		case <-ctx.Done():
			return me, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// liveLoop syncs live until ctx ends or the live room ends (the plan lost
// it, or the session ended).
func (l *Live) liveLoop(ctx context.Context, c *blupcloud.Client) {
	lctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		l.mu.Lock()
		l.active = false
		l.mu.Unlock()
	}()
	l.mu.Lock()
	l.active = true
	l.mu.Unlock()
	l.sync(lctx) // what changed while the Studio was closed
	ended := make(chan error, 1)
	go func() { ended <- l.listen(lctx, c) }()
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ended:
			return
		case <-t.C:
			l.sync(lctx)
		}
	}
}

// listen keeps the live room open, reconnecting with growing pauses. It
// returns when the session ended or the plan has no live room
// (ErrSignedOut, ErrNotLive), or ctx ended.
func (l *Live) listen(ctx context.Context, c *blupcloud.Client) error {
	wait := 5 * time.Second
	for ctx.Err() == nil {
		start := time.Now()
		err := c.Listen(ctx, func(m blupcloud.Moved) {
			if m.Vault == l.siteNow().Sync.Vault {
				l.sync(ctx)
			}
		})
		if errors.Is(err, blupcloud.ErrSignedOut) || errors.Is(err, blupcloud.ErrNotLive) {
			return err
		}
		if time.Since(start) > time.Minute {
			wait = 5 * time.Second // it had been working
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		wait = min(wait*2, 2*time.Minute)
	}
	return ctx.Err()
}

// sync runs one sync at a time; a change during it means one more after.
// A sync cut short by Stop (ctx ended) reports nothing: "context
// canceled" is no error of the person's.
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
		l.emit(ctx, LiveState{State: "syncing", At: time.Now()})
		info, err := syncSite(ctx, l.siteNow(), true)
		if ctx.Err() != nil {
			l.mu.Lock()
			l.syncing, l.again = false, false
			l.mu.Unlock()
			return
		}
		st := LiveState{State: "synced", At: time.Now(), Info: info}
		if err != nil {
			st = LiveState{State: "error", At: time.Now(), Error: err.Error(), Stuck: errors.Is(err, ErrSyncStuck)}
		}
		if errors.Is(err, ErrVaultGone) {
			st.State = "gone"
			l.mu.Lock()
			l.gone, l.syncing = true, false
			l.mu.Unlock()
			l.emit(ctx, st)
			l.Stop()
			return
		}
		l.emit(ctx, st)
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
