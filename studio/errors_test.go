package main

import (
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
)

// What leaves the computer in a report: no home folder (it has the
// person's name), no computer name, no user.
func TestScrubReport(t *testing.T) {
	if len(homes) == 0 {
		t.Skip("no home folder")
	}
	home := homes[0]
	e := scrubReport(&sentry.Event{
		ServerName: "notebook-da-ana",
		Message:    "abrir " + home + "/Notas/Diário.md",
		User:       sentry.User{Email: "ana@x.com"},
		Exception:  []sentry.Exception{{Value: "open " + home + "/Notas/x.md: no such file", Stacktrace: &sentry.Stacktrace{Frames: []sentry.Frame{{AbsPath: home + "/go/x.go"}}}}},
		Threads:    []sentry.Thread{{Stacktrace: &sentry.Stacktrace{Frames: []sentry.Frame{{AbsPath: home + "/go/y.go"}}}}},
		Contexts:   map[string]sentry.Context{"device": {"name": "notebook-da-ana", "arch": "amd64"}},
	}, nil)
	all := e.Message + e.Exception[0].Value + e.Exception[0].Stacktrace.Frames[0].AbsPath + e.Threads[0].Stacktrace.Frames[0].AbsPath
	if strings.Contains(all, home) || !strings.Contains(e.Message, "~/Notas") {
		t.Fatal("pasta pessoal", all)
	}
	if e.ServerName == "notebook-da-ana" || e.User.Email != "" || e.Contexts["device"]["name"] != nil {
		t.Fatal("dados pessoais", e.ServerName, e.User, e.Contexts)
	}
}

// Nothing is sent before the person answers, nor after "no".
func TestReportsNeedAYes(t *testing.T) {
	t.Setenv("BLUP_SENTRY_DSN", "https://abc@ingest.invalid/1")
	startReports(false)
	if reports.on {
		t.Fatal("ligado sem sim")
	}
	startReports(true)
	if !reports.on {
		t.Fatal("desligado com sim")
	}
	startReports(false)
	if reports.on {
		t.Fatal("ligado depois do não")
	}
}
