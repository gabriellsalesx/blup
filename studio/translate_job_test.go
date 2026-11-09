package main

import (
	"errors"
	"testing"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
)

// Two starts racing past the first check: the second never reports
// "started" for a language that is not the one running.
func TestClaimSiteJob(t *testing.T) {
	t.Cleanup(func() {
		siteJob.Lock()
		siteJob.st = TranslateSiteJob{}
		siteJob.Unlock()
	})
	if ok, err := claimSiteJob("en"); !ok || err != nil {
		t.Fatalf("first claim: %v %v", ok, err)
	}
	if ok, err := claimSiteJob("en"); ok || err != nil {
		t.Fatalf("same language again: %v %v", ok, err)
	}
	if ok, err := claimSiteJob("es"); ok || !errors.Is(err, errOtherLang) {
		t.Fatalf("other language: %v %v", ok, err)
	}
	if busy, err := siteJobBusy("es"); !busy || !errors.Is(err, errOtherLang) {
		t.Fatalf("busy other: %v %v", busy, err)
	}
	if busy, err := siteJobBusy("en"); !busy || err != nil {
		t.Fatalf("busy same: %v %v", busy, err)
	}
}

// A sync that ends after another site was opened never starts the live
// sync of the site it synced (it would follow the other site's changes).
func TestLiveAfterSyncOnlyForTheOpenSite(t *testing.T) {
	a, b := config.NewSite(t.TempDir()), config.NewSite(t.TempDir())
	a.Sync.Remote = app.RemoteBlup
	s := &Studio{site: a}
	if !s.liveAfterSync(a, nil) {
		t.Fatal("the open site's first sync does not start live sync")
	}
	if s.liveAfterSync(a, errors.New("x")) {
		t.Fatal("a failed sync starts live sync")
	}
	s.site = b
	if s.liveAfterSync(a, nil) {
		t.Fatal("live sync starts for a site no longer open")
	}
	s.site = nil
	if s.liveAfterSync(a, nil) {
		t.Fatal("live sync starts with no site open")
	}
}
