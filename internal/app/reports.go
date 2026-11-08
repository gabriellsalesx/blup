package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
)

// Error reports from the CLI to Blup Pulse, with the official Sentry SDK,
// as the Studio does (studio/errors.go): only when the person said yes
// (asked once, saved in config.toml, changed with "blup reports on|off"),
// only unexpected errors (not the ones the CLI already explains, nor a
// cancel), no notes, no home folder, no computer name. The DSN comes in the
// release build (-ldflags "-X .../internal/app.SentryDSN=..."), or
// BLUP_SENTRY_DSN for a test; without one nothing is asked nor sent.
// https://docs.sentry.io/platforms/go/

// SentryDSN is the Blup Pulse project of the CLI, set at build time.
var SentryDSN = ""

var reportsOn bool

func reportsDSN() string {
	if v := os.Getenv("BLUP_SENTRY_DSN"); v != "" {
		return v
	}
	return SentryDSN
}

// askReports asks once, in an interactive run, whether errors may go to
// Blup. Nothing is asked in a build without a DSN.
func askReports(g *config.Global) error {
	if g.ErrorReports != "" || reportsDSN() == "" || !ui.Interactive() {
		return nil
	}
	ok, err := ui.Confirm(i18n.T("reports.q"), i18n.T("reports.help"), false)
	if err != nil {
		return err
	}
	g.ErrorReports = "off"
	if ok {
		g.ErrorReports = "on"
	}
	return g.Save()
}

// startReports turns the reports on when the person said yes.
func startReports(g config.Global) {
	dsn := reportsDSN()
	if g.ErrorReports != "on" || dsn == "" || reportsOn {
		return
	}
	env := "production"
	if Version == "dev" {
		env = "development"
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		Release:          "blup-cli@" + Version,
		ServerName:       runtime.GOOS, // never the computer's own name
		DataCollection:   &sentry.DataCollection{UserInfo: sentry.Set(false)},
		EnableTracing:    false,
		AttachStacktrace: true,
		BeforeSend:       scrubReport,
	})
	reportsOn = err == nil
}

// setReports is "blup reports on|off".
func setReports(g *config.Global, args []string) error {
	if len(args) == 0 || (args[0] != "on" && args[0] != "off") {
		fmt.Println(i18n.T("reports.usage", cmpOr(g.ErrorReports, "off")))
		return nil
	}
	g.ErrorReports = args[0]
	if err := g.Save(); err != nil {
		return err
	}
	ui.OK(i18n.T("reports.saved_" + args[0]))
	return nil
}

func cmpOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// expected tells the errors the CLI already explains to the person: they
// are not bugs, so they are never reported.
func expected(err error) bool {
	if errors.Is(err, context.Canceled) || ui.Aborted(err) {
		return true
	}
	for _, d := range knownErrors {
		if errors.Is(err, d.err) {
			return true
		}
	}
	return false
}

// Report sends an unexpected error that ended the command.
func Report(err error) {
	if !reportsOn || err == nil || expected(err) {
		return
	}
	sentry.CaptureException(err)
	sentry.Flush(3 * time.Second)
}

// ReportPanic sends a panic before the CLI stops.
func ReportPanic(v any) {
	if !reportsOn || v == nil {
		return
	}
	sentry.CurrentHub().Recover(v)
	sentry.Flush(3 * time.Second)
}

// The person's home folder, as it shows in paths ("/home/ana/...").
var homes = func() []string {
	home, err := os.UserHomeDir()
	if err != nil || len(home) < 3 {
		return nil
	}
	return []string{home, filepath.ToSlash(home)}
}()

// scrub replaces the home folder with "~": it carries the person's name.
func scrub(s string) string {
	for _, h := range homes {
		s = strings.ReplaceAll(s, h, "~")
	}
	return s
}

func scrubReport(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	event.ServerName = runtime.GOOS
	event.User = sentry.User{}
	event.Request = nil
	event.Message = scrub(event.Message)
	scrubFrames := func(st *sentry.Stacktrace) {
		if st == nil {
			return
		}
		for j := range st.Frames {
			f := &st.Frames[j]
			f.AbsPath, f.Filename = scrub(f.AbsPath), scrub(f.Filename)
		}
	}
	for i := range event.Exception {
		ex := &event.Exception[i]
		ex.Value = scrub(ex.Value)
		scrubFrames(ex.Stacktrace)
	}
	for i := range event.Threads {
		scrubFrames(event.Threads[i].Stacktrace)
	}
	for i := range event.Breadcrumbs {
		event.Breadcrumbs[i].Message = scrub(event.Breadcrumbs[i].Message)
		event.Breadcrumbs[i].Data = nil
	}
	if dev, ok := event.Contexts["device"]; ok {
		delete(dev, "name")
	}
	return event
}
