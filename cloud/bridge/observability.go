package main

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

// Os erros da Ponte vão para o Blup Pulse pelo SDK oficial do Sentry (só
// erros: o Pulse não guarda desempenho). Sem SENTRY_DSN, nada é enviado.
// Nada pessoal sai daqui: nem usuário, nem cabeçalhos, cookies, query ou
// corpo dos pedidos (lá vão assinaturas e números de WhatsApp).
// https://docs.sentry.io/platforms/go/guides/http/

// version é a versão da Ponte, posta no build (-ldflags "-X main.version=...").
var version = "dev"

func initSentry(dsn, environment string) (bool, error) {
	if dsn == "" {
		return false, nil
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      or(environment, "production"),
		Release:          "blup-bridge@" + version,
		SendDefaultPII:   false,
		EnableTracing:    false,
		AttachStacktrace: true,
		BeforeSend:       scrubEvent,
	})
	return err == nil, err
}

// scrubEvent deixa do pedido só o método e o caminho, e tira o usuário.
func scrubEvent(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	if event.Request != nil {
		url, _, _ := strings.Cut(event.Request.URL, "?")
		event.Request = &sentry.Request{Method: event.Request.Method, URL: url}
	}
	event.User = sentry.User{}
	return event
}

// Saúde, para o monitor do Blup Pulse (sem assinatura, sem detalhes):
//   - /health e /_blup/health: o processo responde;
//   - /_blup/ready: consegue trabalhar: o banco responde (o servidor só
//     sobe depois de religar as sessões). 503 quando não.
type readiness struct {
	db *sql.DB
}

func (rd readiness) check(ctx context.Context) bool {
	if rd.db == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return rd.db.PingContext(ctx) == nil
}

func healthPath(path string) bool {
	return path == "/health" || path == "/_blup/health" || path == "/_blup/ready"
}

func serveHealth(w http.ResponseWriter, r *http.Request, rd *readiness) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	status := http.StatusOK
	if r.URL.Path == "/_blup/ready" && (rd == nil || !rd.check(r.Context())) {
		status = http.StatusServiceUnavailable
	}
	w.Header().Set("Cache-Control", "no-store")
	body := map[string]any{"status": "ok"}
	if status != http.StatusOK {
		body = map[string]any{"status": "unavailable"}
	}
	// /health continua respondendo {"ok":true}, como antes.
	if r.URL.Path == "/health" {
		body = map[string]any{"ok": true}
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(status)
		return
	}
	reply(w, status, body)
}
