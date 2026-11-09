package ui

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// The Studio marks the process headless: no prompt may open the terminal
// (huh fails with "could not open TTY", or, started from a terminal,
// draws there and waits for an answer that never comes).
func TestHeadlessPromptsFailClearly(t *testing.T) {
	SetHeadless()
	if Interactive() {
		t.Fatal("Interactive is true while headless")
	}
	var s string
	if err := Ask("q", "", "", &s, nil); !errors.Is(err, ErrNoTerminal) {
		t.Fatalf("Ask: %v", err)
	}
	if err := AskSecret("q", "h", &s); !errors.Is(err, ErrNoTerminal) {
		t.Fatalf("AskSecret: %v", err)
	}
	if ok, err := Confirm("q", "", true); !errors.Is(err, ErrNoTerminal) || !ok {
		t.Fatalf("Confirm: %v %v", ok, err)
	}
	if err := Explain("h", "b"); !errors.Is(err, ErrNoTerminal) {
		t.Fatalf("Explain: %v", err)
	}
	if ErrNoTerminal.Error() == "" || ErrNoTerminal.Error() == "ui.no_terminal" {
		t.Fatalf("no message for ErrNoTerminal: %q", ErrNoTerminal.Error())
	}
}

// SetReport is set and cleared by the Studio's publish while other calls
// print lines (go test -race).
func TestReportConcurrent(t *testing.T) {
	var mu sync.Mutex
	got := 0
	SetReport(func(kind, text string) {
		mu.Lock()
		got++
		mu.Unlock()
	})
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 50 {
				Warn("w")
				_ = Spin(context.Background(), "s", func(context.Context) error { return nil })
			}
		})
	}
	wg.Go(func() {
		for range 50 {
			SetReport(func(string, string) {})
		}
	})
	wg.Wait()
	SetReport(nil)
}
