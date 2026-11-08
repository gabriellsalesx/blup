package app

import (
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

// After Stop, a change on disk no longer starts a sync: the timer that
// would send it is never made.
func TestLiveChangedAfterStop(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // signed out: run returns at once
	t.Setenv("HOME", t.TempDir())
	site := &config.Site{Sync: config.NoteSync{Remote: RemoteBlup, Vault: "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"}}
	l := StartLive(site, func(LiveState) { t.Error("a sync ran after Stop") })
	if l == nil {
		t.Fatal("no live sync for a Blup Cloud site")
	}
	l.Stop()
	l.Changed()
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.timer != nil {
		t.Fatal("a change after Stop scheduled a sync")
	}
}
