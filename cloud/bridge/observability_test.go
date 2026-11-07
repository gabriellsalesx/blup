package main

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

// Um erro que a Ponte responde com 502 chega ao Blup Pulse (aqui, um
// servidor de teste no lugar do ingest), sem a query nem os cabeçalhos.
func TestErrorsReachPulse(t *testing.T) {
	var mu sync.Mutex
	var got []string
	ingest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, string(body))
		mu.Unlock()
		w.WriteHeader(200)
	}))
	defer ingest.Close()
	dsn := strings.Replace(ingest.URL, "http://", "http://abc@", 1) + "/7"
	if on, err := initSentry(dsn, "development"); !on || err != nil {
		t.Fatal("init", on, err)
	}
	defer sentry.Init(sentry.ClientOptions{})
	w := httptest.NewRecorder()
	answer(w, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, errors.New("falha do WhatsApp de teste"))
	if w.Code != 502 {
		t.Fatal("status", w.Code)
	}
	if !sentry.Flush(5 * time.Second) {
		t.Fatal("flush")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 || !strings.Contains(got[0], "falha do WhatsApp de teste") || !strings.Contains(got[0], `"environment":"development"`) {
		t.Fatal("envelope", got)
	}
	event := scrubEvent(&sentry.Event{Request: &sentry.Request{Method: "POST", URL: "https://ponte/v1/sessions/x?token=1", Headers: map[string]string{"X-Blup-Signature": "s"}}, User: sentry.User{Email: "a@b.c"}}, nil)
	if event.Request.URL != "https://ponte/v1/sessions/x" || event.Request.Headers != nil || event.User.Email != "" {
		t.Fatal("dados pessoais", event.Request, event.User)
	}
}
