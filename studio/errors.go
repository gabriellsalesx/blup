package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/getsentry/sentry-go"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Error reports: the Studio's errors go to Blup Pulse with the official
// Sentry SDK, only when the person said yes (asked once; changed in
// Settings). Errors only: no tracing, no notes, no file names with the
// person's name, no computer name. The DSN comes in the build
// (-ldflags "-X main.studioSentryDSN=..."), or BLUP_SENTRY_DSN for a test;
// without one nothing is sent.
// https://docs.sentry.io/platforms/go/

var studioSentryDSN = ""

var reports struct {
	sync.Mutex
	on bool
}

func sentryDSN() string {
	if v := os.Getenv("BLUP_SENTRY_DSN"); v != "" {
		return v
	}
	return studioSentryDSN
}

func reportsEnvironment() string {
	if app.Version == "dev" {
		return "development"
	}
	return "production"
}

// startReports turns the reports on or off to match the person's answer.
func startReports(on bool) {
	reports.Lock()
	defer reports.Unlock()
	dsn := sentryDSN()
	if !on || dsn == "" {
		if reports.on {
			sentry.Flush(2 * time.Second)
			_ = sentry.Init(sentry.ClientOptions{}) // no DSN: nothing is sent
		}
		reports.on = false
		return
	}
	if reports.on {
		return
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Environment: reportsEnvironment(),
		Release:     "blup-studio@" + app.Version,
		ServerName:  runtime.GOOS, // never the computer's own name
		// No user data from the integrations (SendDefaultPII is deprecated
		// since sentry-go v0.50 in favour of DataCollection).
		DataCollection:   &sentry.DataCollection{UserInfo: sentry.Set(false)},
		EnableTracing:    false,
		AttachStacktrace: true,
		BeforeSend:       scrubReport,
	})
	reports.on = err == nil
}

// The person's home folder, as it shows in paths ("/home/ana/..."), and
// the same with the other slash on Windows.
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
	// CaptureMessage with AttachStacktrace puts the frames in Threads.
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

// onPanic is the Wails panic handler: the panic is reported (when the
// person said yes), then the Studio stops as Wails' own handler does.
func onPanic(details *application.PanicDetails) {
	if details == nil {
		return
	}
	err := details.Error
	if err == nil {
		err = fmt.Errorf("panic")
	}
	reports.Lock()
	on := reports.on
	reports.Unlock()
	if on {
		sentry.CaptureException(err)
		sentry.Flush(3 * time.Second)
	}
	fmt.Fprintf(os.Stderr, "panic error: %v\n%s\n", err, details.StackTrace)
	os.Exit(1)
}

// ErrorReports is what the page needs: the person's answer ("" = not
// asked yet, "on" or "off") and, when on, the DSN and the release for the
// page's own errors.
type ErrorReports struct {
	Answer      string `json:"answer"`
	DSN         string `json:"dsn"`
	Environment string `json:"environment"`
	Release     string `json:"release"`
}

func (s *Studio) ErrorReports() ErrorReports {
	p := s.Prefs()
	r := ErrorReports{Answer: p.ErrorReports, Environment: reportsEnvironment(), Release: "blup-studio@" + app.Version}
	if p.ErrorReports == "on" {
		r.DSN = sentryDSN()
	}
	return r
}

// SetErrorReports saves the person's answer and turns the reports on or
// off at once.
func (s *Studio) SetErrorReports(on bool) (ErrorReports, error) {
	answer := "off"
	if on {
		answer = "on"
	}
	if err := updatePrefs(func(p *Prefs) { p.ErrorReports = answer }); err != nil {
		return ErrorReports{}, err
	}
	startReports(on)
	return s.ErrorReports(), nil
}
