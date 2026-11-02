package main

import (
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
)

// The quiet republish runs at most once in a while per site, and a run
// that couldn't start (another publish going) doesn't count.
func TestRepublishThrottle(t *testing.T) {
	s := &Studio{}
	site := config.NewSite(t.TempDir()) // no destination: the publish fails at once
	busy.Lock()
	s.republish(site)
	busy.Unlock()
	if _, ok := s.republished[site.Dir()]; ok {
		t.Fatal("a publish that never ran counted")
	}
	s.republish(site)
	first, ok := s.republished[site.Dir()]
	if !ok {
		t.Fatal("the publish was not noted")
	}
	time.Sleep(time.Millisecond)
	s.republish(site)
	if s.republished[site.Dir()] != first {
		t.Fatal("published again within the wait")
	}
	s.republished[site.Dir()] = time.Now().Add(-republishWait - time.Second)
	s.republish(site)
	if !s.republished[site.Dir()].After(first) {
		t.Fatal("not published again after the wait")
	}
}
