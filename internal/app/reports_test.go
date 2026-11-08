package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/getsentry/sentry-go"
)

func TestReportsScrubHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || len(home) < 3 {
		t.Skip("no home folder")
	}
	ev := &sentry.Event{
		Message:    "open " + home + "/notas/a.md",
		ServerName: "maquina-da-ana",
		User:       sentry.User{Email: "ana@example.com"},
		Exception: []sentry.Exception{{Value: home + "/x", Stacktrace: &sentry.Stacktrace{
			Frames: []sentry.Frame{{AbsPath: home + "/go/src/a.go", Filename: home + "/a.go"}}}}},
		Contexts: map[string]sentry.Context{"device": {"name": "maquina-da-ana"}},
	}
	got := scrubReport(ev, nil)
	if got.Message != "open ~/notas/a.md" || got.Exception[0].Value != "~/x" ||
		got.Exception[0].Stacktrace.Frames[0].AbsPath != "~/go/src/a.go" {
		t.Fatalf("home not scrubbed: %+v", got)
	}
	if got.User.Email != "" || got.ServerName == "maquina-da-ana" {
		t.Fatalf("person or computer left in the report: %+v", got)
	}
	if _, ok := got.Contexts["device"]["name"]; ok {
		t.Fatal("device name left in the report")
	}
}

func TestReportsSkipExpected(t *testing.T) {
	known := errors.New("known")
	registerErrors(knownError{known, "cancelled"})
	if !expected(fmt.Errorf("wrap: %w", known)) || !expected(context.Canceled) {
		t.Fatal("an error the CLI explains must not be reported")
	}
	if expected(errors.New("bug")) {
		t.Fatal("an unexpected error must be reported")
	}
}

func TestReportsOffByDefault(t *testing.T) {
	if reportsOn {
		t.Fatal("reports on without an answer")
	}
	Report(errors.New("x")) // must not panic nor send anything
}
