package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Live sync on the phone, for plans that have it: while the app is on
// screen it keeps Blup Cloud's live room open (blupcloud.Client.Listen, the
// same one the Studio uses) and syncs as soon as another device saves to
// the linked vault, plus every few minutes in case a message was missed.
// In the background the room closes (Android keeps no socket awake for an
// app nobody is looking at, and the battery thanks it); back on screen, or
// with the network back, it opens again at once. Plans without it sync by
// the screen's own triggers.

func init() {
	// What a sync of its own did, so the screen shows it.
	application.RegisterEvent[LiveSync]("cloudSynced")
}

// LiveSync is told to the screen after each sync of the live room.
type LiveSync struct {
	Result SyncResult `json:"result"`
	Error  string     `json:"error"`
	// Stuck: the sync stopped for safety; the Cloud card offers "Compare
	// from scratch".
	Stuck bool `json:"stuck"`
}

// cloudSynced tells the screen. Without the app (the tests), nobody hears.
var cloudSynced = func(r LiveSync) {
	if a := application.Get(); a != nil {
		a.Event.Emit("cloudSynced", r)
	}
}

// The live room's pauses: a dropped connection is tried again after
// liveRetry, doubling up to liveRetryMax; a sync runs liveSettle after the
// last message (one save sends several), and every liveCheck anyway.
var (
	liveRetry    = 5 * time.Second
	liveRetryMax = 2 * time.Minute
	liveSettle   = 700 * time.Millisecond
	liveCheck    = 5 * time.Minute
)

// errLiveOff: the session has nothing to listen to (signed out, a plan
// without live sync): it waits for the account to be read again.
var errLiveOff = errors.New("live sync off")

// liveSync opens the live room only while it is wanted (a live plan with a
// linked, unlocked vault: Account says so) and the app is on screen.
type liveSync struct {
	// session keeps the room open until ctx ends or it drops; sync is
	// asked to run whenever something may have changed. Replaced in tests.
	session func(ctx context.Context, sync func()) error
	syncNow func(ctx context.Context)

	mu         sync.Mutex
	want       bool
	foreground bool
	cancel     context.CancelFunc // of the loop running, nil for none
	done       chan struct{}      // closed when that loop ends
	kick       chan struct{}      // the network came back: retry now
}

var live = newLive()

func newLive() *liveSync {
	l := &liveSync{foreground: true, kick: make(chan struct{}, 1)}
	l.session = listenSession
	l.syncNow = liveSyncNow
	return l
}

// Want turns the live room on or off, as the account allows.
func (l *liveSync) Want(on bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.want = on
	l.apply()
}

// Foreground is told when the app comes on screen or leaves it.
func (l *liveSync) Foreground(on bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.foreground = on
	l.apply()
}

// Online is told when the network comes back: a room waiting to try again
// tries now.
func (l *liveSync) Online() {
	select {
	case l.kick <- struct{}{}:
	default:
	}
}

// Running tells whether the room's loop is on.
func (l *liveSync) Running() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.cancel != nil
}

// Stop ends the loop and waits for it.
func (l *liveSync) Stop() {
	l.mu.Lock()
	l.want = false
	done := l.stopLocked()
	l.mu.Unlock()
	if done != nil {
		<-done
	}
}

// apply starts or stops the loop as want and foreground say. l.mu held.
func (l *liveSync) apply() {
	if !l.want || !l.foreground {
		l.stopLocked()
		return
	}
	if l.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	l.cancel, l.done = cancel, done
	go func() {
		defer close(done)
		l.run(ctx)
		l.mu.Lock()
		if l.done == done { // not already replaced by a newer loop
			l.cancel, l.done = nil, nil
		}
		l.mu.Unlock()
		cancel()
	}()
}

func (l *liveSync) stopLocked() chan struct{} {
	if l.cancel == nil {
		return nil
	}
	done := l.done
	l.cancel()
	l.cancel, l.done = nil, nil
	return done
}

// run keeps the room open, reconnecting with growing pauses, until ctx
// ends or there is nothing to listen to.
func (l *liveSync) run(ctx context.Context) {
	wake := make(chan struct{}, 1)
	ask := func() {
		select {
		case wake <- struct{}{}:
		default:
		}
	}
	// The helpers below end with the loop, whatever ends it.
	ctx, cancel := context.WithCancel(ctx)
	var syncing sync.WaitGroup
	defer func() { cancel(); syncing.Wait() }()
	// One sync at a time, a moment after the last message.
	syncing.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-wake:
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(liveSettle):
			}
			// What came during the pause is in this sync.
			select {
			case <-wake:
			default:
			}
			l.syncNow(ctx)
		}
	})
	syncing.Go(func() {
		t := time.NewTicker(liveCheck)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				ask()
			}
		}
	})

	wait := liveRetry
	for ctx.Err() == nil {
		start := time.Now()
		err := l.session(ctx, ask)
		if errors.Is(err, errLiveOff) || errors.Is(err, blupcloud.ErrSignedOut) || errors.Is(err, blupcloud.ErrNotLive) {
			return
		}
		if time.Since(start) > time.Minute {
			wait = liveRetry // it had been working
		}
		select {
		case <-ctx.Done():
			return
		case <-l.kick:
			wait = liveRetry
			continue
		case <-time.After(wait):
		}
		wait = min(wait*2, liveRetryMax)
	}
}

// listenSession checks the plan, syncs what changed while the room was
// closed and listens until the connection drops.
func listenSession(ctx context.Context, changed func()) error {
	c, _, err := client()
	if err != nil {
		if errors.Is(err, blupcloud.ErrSignedOut) {
			return err
		}
		return errLiveOff
	}
	me, err := c.Me(ctx)
	if err != nil {
		return err // offline: tried again
	}
	if !me.Plan.Limits.Live {
		return errLiveOff
	}
	changed() // what changed while it was closed
	return c.Listen(ctx, func(m blupcloud.Moved) {
		if l, err := loadLink(); err == nil && l.ID != "" && m.Vault == l.ID {
			changed()
		}
	})
}

// liveSyncNow syncs the open vault and tells the screen. Not linked (another
// vault opened) or locked, there is nothing to say.
func liveSyncNow(ctx context.Context) {
	if l, err := loadLink(); err != nil || l.ID == "" {
		return
	}
	r, err := (&CloudService{}).sync(ctx, false)
	if ctx.Err() != nil {
		return // left the screen: the next one will tell
	}
	out := LiveSync{Result: r}
	if err != nil {
		var st stuckError
		out.Error, out.Stuck = err.Error(), errors.As(err, &st)
	}
	cloudSynced(out)
}

// ServiceStartup follows the app on and off the screen (Android) and the
// network coming back, for the live room.
func (s *CloudService) ServiceStartup(context.Context, application.ServiceOptions) error {
	a := application.Get()
	if a == nil {
		return nil
	}
	on := func(*application.ApplicationEvent) {
		// Connections Android cut while the app was away are not reused.
		dropStaleConns()
		live.Foreground(true)
	}
	off := func(*application.ApplicationEvent) { live.Foreground(false) }
	a.Event.OnApplicationEvent(events.Android.ActivityResumed, on)
	a.Event.OnApplicationEvent(events.Android.ActivityPaused, off)
	a.Event.OnApplicationEvent(events.Android.ActivityStopped, off)
	a.Event.OnApplicationEvent(events.Common.NetworkChanged, func(e *application.ApplicationEvent) {
		if connected, ok := e.Context().Data()["connected"].(bool); ok && connected {
			live.Online()
		}
	})
	return nil
}

// ServiceShutdown closes the room.
func (s *CloudService) ServiceShutdown() error {
	live.Stop()
	return nil
}
