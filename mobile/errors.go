package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Relatórios de erro do celular: vão para o Blup Pulse pelo SDK oficial do
// Sentry só depois do "sim" da pessoa (perguntado uma vez; muda nos
// Ajustes). Só erros, sem texto de nota, sem caminho com o nome da pessoa,
// sem nome do aparelho. A DSN vem no build (-ldflags "-X
// main.studioSentryDSN=..."); sem ela, nada sai.
// https://docs.sentry.io/platforms/go/

var (
	studioSentryDSN = ""
	appVersion      = "dev"
)

var reports struct {
	sync.Mutex
	on bool
}

func reportsEnvironment() string {
	if appVersion == "dev" {
		return "development"
	}
	return "production"
}

func startReports(on bool) {
	reports.Lock()
	defer reports.Unlock()
	if !on || studioSentryDSN == "" {
		if reports.on {
			sentry.Flush(2 * time.Second)
			_ = sentry.Init(sentry.ClientOptions{})
		}
		reports.on = false
		return
	}
	if reports.on {
		return
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:            studioSentryDSN,
		Environment:    reportsEnvironment(),
		Release:        "blup-notes@" + appVersion,
		ServerName:     runtime.GOOS,
		SendDefaultPII: false,
		EnableTracing:  false,
		BeforeSend:     scrubReport,
	})
	reports.on = err == nil
}

// A pasta do app e a pessoal viram "~": os caminhos levam o nome da pessoa.
func scrub(s string) string {
	for _, dir := range privateDirs() {
		if len(dir) > 2 {
			s = strings.ReplaceAll(s, dir, "~")
		}
	}
	return s
}

func privateDirs() []string {
	var dirs []string
	if d, err := dataDir(); err == nil {
		dirs = append(dirs, d)
	}
	if h, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, h)
	}
	return dirs
}

func scrubReport(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	event.ServerName = runtime.GOOS
	event.User = sentry.User{}
	event.Request = nil
	event.Message = scrub(event.Message)
	for i := range event.Exception {
		ex := &event.Exception[i]
		ex.Value = scrub(ex.Value)
		if ex.Stacktrace != nil {
			for j := range ex.Stacktrace.Frames {
				f := &ex.Stacktrace.Frames[j]
				f.AbsPath, f.Filename = scrub(f.AbsPath), scrub(f.Filename)
			}
		}
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

// onPanic: o pânico vai para o Blup Pulse (com o "sim") e o app para, como
// no tratamento padrão do Wails.
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

// ErrorReports é o que a tela precisa: a resposta ("" = ainda não
// perguntado, "on" ou "off") e, com "on", a DSN para os erros da própria
// tela.
type ErrorReports struct {
	Answer      string `json:"answer"`
	DSN         string `json:"dsn"`
	Environment string `json:"environment"`
	Release     string `json:"release"`
}

func reportsView(answer string) ErrorReports {
	r := ErrorReports{Answer: answer, Environment: reportsEnvironment(), Release: "blup-notes@" + appVersion}
	if answer == "on" {
		r.DSN = studioSentryDSN
	}
	return r
}

// ErrorReports é pedido pela tela ao abrir: aí os relatórios ligam, se a
// pessoa já disse sim (antes disso a pasta do app pode não estar pronta).
func (c *SettingsService) ErrorReports() (ErrorReports, error) {
	s, err := readSettings()
	if err == nil {
		startReports(s.Reports == "on")
	}
	return reportsView(s.Reports), err
}

func (c *SettingsService) SetErrorReports(on bool) (ErrorReports, error) {
	settingsLock.Lock()
	s, err := readSettings()
	if err == nil {
		s.Reports = "off"
		if on {
			s.Reports = "on"
		}
		err = writeSettings(s)
	}
	settingsLock.Unlock()
	if err != nil {
		return ErrorReports{}, err
	}
	startReports(on)
	return reportsView(s.Reports), nil
}
