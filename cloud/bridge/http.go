package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// bridge é o que as rotas usam: o manager de verdade, ou um falso nos
// testes.
type bridge interface {
	Start(ctx context.Context, id string) (View, error)
	View(id string) (View, error)
	Channels(ctx context.Context, id string) ([]Channel, error)
	Publish(ctx context.Context, id string, p Post) (Sent, error)
	Remove(ctx context.Context, id string) error
}

const (
	maxBody  = 8 << 20 // a imagem vem em base64
	maxImage = 5 << 20
	maxText  = 4096
)

var (
	sessionID = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
	channelID = regexp.MustCompile(`^[0-9]{6,32}$`)
	postKey   = regexp.MustCompile(`^[A-Za-z0-9_:-]{8,128}$`)
)

// routes: só /health fica aberta; o resto exige a assinatura do Blup Cloud.
func routes(b bridge, v *verifier, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { reply(w, 200, map[string]bool{"ok": true}) })
	mux.HandleFunc("POST /v1/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		view, err := b.Start(r.Context(), r.PathValue("id"))
		answer(w, log, view, err)
	})
	mux.HandleFunc("GET /v1/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		view, err := b.View(r.PathValue("id"))
		answer(w, log, view, err)
	})
	mux.HandleFunc("GET /v1/sessions/{id}/channels", func(w http.ResponseWriter, r *http.Request) {
		list, err := b.Channels(r.Context(), r.PathValue("id"))
		answer(w, log, map[string][]Channel{"channels": list}, err)
	})
	mux.HandleFunc("POST /v1/sessions/{id}/posts", func(w http.ResponseWriter, r *http.Request) {
		p, ok := readPost(r)
		if !ok {
			reply(w, 400, code("BAD_REQUEST"))
			return
		}
		sent, err := b.Publish(r.Context(), r.PathValue("id"), p)
		answer(w, log, sent, err)
	})
	mux.HandleFunc("DELETE /v1/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := b.Remove(r.Context(), r.PathValue("id")); err != nil {
			answer(w, log, nil, err)
			return
		}
		w.WriteHeader(204)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			mux.ServeHTTP(w, r)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
		if err != nil {
			reply(w, 413, code("TOO_LARGE"))
			return
		}
		if !v.check(r, body) {
			reply(w, 401, code("UNAUTHORIZED"))
			return
		}
		// O formato da sessão é conferido antes de qualquer rota.
		rest, found := strings.CutPrefix(r.URL.Path, "/v1/sessions/")
		if id, _, _ := strings.Cut(rest, "/"); !found || !sessionID.MatchString(id) {
			reply(w, 404, code("NOT_FOUND"))
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		mux.ServeHTTP(w, r)
	})
}

type postBody struct {
	Channel string `json:"channel"`
	Key     string `json:"key"`
	Text    string `json:"text"`
	Image   string `json:"image"` // base64, JPEG ou PNG
}

func readPost(r *http.Request) (Post, bool) {
	var in postBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return Post{}, false
	}
	in.Channel = strings.TrimSuffix(in.Channel, "@newsletter")
	if !channelID.MatchString(in.Channel) || !postKey.MatchString(in.Key) {
		return Post{}, false
	}
	if !utf8.ValidString(in.Text) || utf8.RuneCountInString(in.Text) > maxText || (strings.TrimSpace(in.Text) == "" && in.Image == "") {
		return Post{}, false
	}
	p := Post{Channel: in.Channel, Key: in.Key, Text: in.Text}
	if in.Image != "" {
		img, err := base64.StdEncoding.DecodeString(in.Image)
		if err != nil || len(img) == 0 || len(img) > maxImage {
			return Post{}, false
		}
		if t := http.DetectContentType(img); t != "image/jpeg" && t != "image/png" {
			return Post{}, false
		}
		p.Image = img
	}
	return p, true
}

func answer(w http.ResponseWriter, log *slog.Logger, v any, err error) {
	var soon TooSoon
	switch {
	case err == nil:
		reply(w, 200, v)
	case errors.As(err, &soon):
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(soon.Wait.Seconds()))))
		reply(w, 429, map[string]any{"code": "TOO_SOON", "retryAfter": int(math.Ceil(soon.Wait.Seconds()))})
	case errors.Is(err, errUnknown):
		reply(w, 404, code("NOT_FOUND"))
	case errors.Is(err, errNotConnected):
		reply(w, 409, code("NOT_CONNECTED"))
	case errors.Is(err, errNotAdmin):
		reply(w, 403, code("NOT_ADMIN"))
	case errors.Is(err, errBusy):
		reply(w, 503, code("BUSY"))
	default:
		log.Error("pedido falhou", "err", err)
		reply(w, 502, code("UPSTREAM"))
	}
}

func code(c string) map[string]string { return map[string]string{"code": c} }

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
