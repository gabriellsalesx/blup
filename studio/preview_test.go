package main

import (
	"os/exec"
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
)

// sleeper is a process standing in for hugo server.
func sleeper(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skip("no sleep command:", err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	return cmd
}

// stopped tells whether the process ended (it was killed).
func stopped(cmd *exec.Cmd) bool {
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

// Two starts of the preview can overlap (a restart after sections changed
// while the window asks for the preview): the second hugo must be stopped,
// not left running unseen; and a start for a site no longer open is too.
func TestAdoptPreview(t *testing.T) {
	a, b := config.NewSite(t.TempDir()), config.NewSite(t.TempDir())
	s := &Studio{site: a}

	first := sleeper(t)
	if url, err := s.adopt(a, first, "http://localhost:1313/", nil); err != nil || url != "http://localhost:1313/" {
		t.Fatalf("first = %q, %v", url, err)
	}
	second := sleeper(t)
	if url, err := s.adopt(a, second, "http://localhost:1314/", nil); err != nil || url != "http://localhost:1313/" {
		t.Errorf("second = %q, %v", url, err)
	}
	if !stopped(second) {
		t.Error("the second hugo kept running")
	}
	if s.preview != first {
		t.Error("the running preview changed")
	}

	s.StopPreview()
	s.site = b
	late := sleeper(t)
	if _, err := s.adopt(a, late, "http://localhost:1315/", nil); err == nil {
		t.Error("a preview of a closed site was kept")
	}
	if !stopped(late) || s.preview != nil {
		t.Error("the preview of a closed site kept running")
	}
}
