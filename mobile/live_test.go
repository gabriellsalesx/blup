package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
)

func fastLive(t *testing.T) {
	t.Helper()
	r, m, s, c := liveRetry, liveRetryMax, liveSettle, liveCheck
	liveRetry, liveRetryMax, liveSettle, liveCheck = 20*time.Millisecond, 40*time.Millisecond, 5*time.Millisecond, time.Hour
	t.Cleanup(func() { liveRetry, liveRetryMax, liveSettle, liveCheck = r, m, s, c })
}

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for range 200 {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(what)
}

// The room is open only while wanted and on screen; leaving the screen
// closes it (the session's ctx ends) and coming back opens it again.
func TestLiveFollowsTheScreen(t *testing.T) {
	fastLive(t)
	var sessions, open, syncs atomic.Int32
	l := newLive()
	l.session = func(ctx context.Context, changed func()) error {
		sessions.Add(1)
		open.Add(1)
		defer open.Add(-1)
		changed()
		<-ctx.Done()
		return ctx.Err()
	}
	l.syncNow = func(context.Context) { syncs.Add(1) }
	defer l.Stop()

	l.Foreground(true)
	if l.Running() {
		t.Fatal("not wanted yet, but running")
	}
	l.Want(true)
	eventually(t, "the room never opened", func() bool { return open.Load() == 1 })
	eventually(t, "no sync on opening", func() bool { return syncs.Load() == 1 })

	l.Foreground(false)
	eventually(t, "still open in the background", func() bool { return open.Load() == 0 && !l.Running() })

	l.Foreground(true)
	eventually(t, "not open again on screen", func() bool { return open.Load() == 1 && sessions.Load() == 2 })

	l.Want(false)
	eventually(t, "still open when no longer wanted", func() bool { return open.Load() == 0 })
}

// A dropped connection is tried again; signed out or a plan without live
// sync, the loop ends until the account is read again.
func TestLiveRetriesAndStops(t *testing.T) {
	fastLive(t)
	var sessions atomic.Int32
	l := newLive()
	l.syncNow = func(context.Context) {}
	l.session = func(ctx context.Context, _ func()) error {
		if sessions.Add(1) < 3 {
			return errors.New("offline")
		}
		return blupcloud.ErrNotLive
	}
	defer l.Stop()
	l.Want(true)
	eventually(t, "did not stop on ErrNotLive", func() bool { return sessions.Load() == 3 && !l.Running() })
	time.Sleep(60 * time.Millisecond)
	if n := sessions.Load(); n != 3 {
		t.Fatalf("sessions after it stopped = %d, want 3", n)
	}
	l.Want(true) // the account read again
	eventually(t, "not started again", func() bool { return sessions.Load() == 4 })
}

// The network back, the room tries at once instead of after its pause.
func TestLiveOnlineRetriesNow(t *testing.T) {
	fastLive(t)
	liveRetry, liveRetryMax = time.Hour, time.Hour
	var sessions atomic.Int32
	l := newLive()
	l.syncNow = func(context.Context) {}
	l.session = func(context.Context, func()) error {
		sessions.Add(1)
		return errors.New("offline")
	}
	defer l.Stop()
	l.Want(true)
	eventually(t, "never tried", func() bool { return sessions.Load() == 1 })
	l.Online()
	eventually(t, "did not try again with the network back", func() bool { return sessions.Load() == 2 })
}

// Many messages in a row make one sync, not one each.
func TestLiveSettles(t *testing.T) {
	fastLive(t)
	liveSettle = 50 * time.Millisecond
	var syncs atomic.Int32
	l := newLive()
	l.syncNow = func(context.Context) { syncs.Add(1) }
	l.session = func(ctx context.Context, changed func()) error {
		for range 5 {
			changed()
		}
		<-ctx.Done()
		return ctx.Err()
	}
	defer l.Stop()
	l.Want(true)
	eventually(t, "no sync", func() bool { return syncs.Load() >= 1 })
	time.Sleep(150 * time.Millisecond)
	if n := syncs.Load(); n != 1 {
		t.Fatalf("syncs = %d, want 1", n)
	}
}
