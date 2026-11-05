package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var secret = []byte(strings.Repeat("s", 40))

func signed(t *testing.T, method, path string, body []byte, at time.Time) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	sign(req, secret, body, at)
	return req
}

// Assinatura: o pedido certo passa; corpo trocado, caminho trocado, senha
// errada, fora da janela de cinco minutos ou repetido, não.
func TestSignatureRefusesTampering(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	v := newVerifier(secret, func() time.Time { return now })
	body := []byte(`{"a":1}`)
	if !v.check(signed(t, "POST", "/v1/sessions/abcdefgh/posts", body, now), body) {
		t.Fatal("o pedido certo foi recusado")
	}
	req := signed(t, "POST", "/v1/sessions/abcdefgh/posts", body, now.Add(time.Second))
	if v.check(req, []byte(`{"a":2}`)) {
		t.Fatal("corpo trocado passou")
	}
	req = signed(t, "POST", "/v1/sessions/abcdefgh/posts", body, now.Add(2*time.Second))
	req.URL.Path = "/v1/sessions/outrasess/posts"
	if v.check(req, body) {
		t.Fatal("caminho trocado passou")
	}
	other := httptest.NewRequest("POST", "/v1/sessions/abcdefgh/posts", bytes.NewReader(body))
	sign(other, []byte(strings.Repeat("x", 40)), body, now)
	if v.check(other, body) {
		t.Fatal("senha errada passou")
	}
	if v.check(signed(t, "POST", "/v1/sessions/abcdefgh/posts", body, now.Add(-6*time.Minute)), body) {
		t.Fatal("pedido velho passou")
	}
	again := signed(t, "GET", "/v1/sessions/abcdefgh", nil, now.Add(3*time.Second))
	if !v.check(again, nil) {
		t.Fatal("primeira vez recusada")
	}
	copied := httptest.NewRequest("GET", "/v1/sessions/abcdefgh", nil)
	copied.Header = again.Header.Clone()
	if v.check(copied, nil) {
		t.Fatal("pedido repetido passou")
	}
	if !v.check(signed(t, "GET", "/v1/sessions/abcdefgh", nil, now.Add(3*time.Second)), nil) {
		t.Fatal("o mesmo pedido, de novo e com outro nonce, foi recusado")
	}
}

func TestConfigRefusesWeakSettings(t *testing.T) {
	base := map[string]string{"BRIDGE_SECRET": string(secret)}
	get := func(over map[string]string) func(string) string {
		return func(k string) string {
			if v, ok := over[k]; ok {
				return v
			}
			return base[k]
		}
	}
	c, err := loadConfig(get(nil))
	if err != nil || c.MinGap != 30*time.Minute || c.Host != "127.0.0.1" {
		t.Fatalf("padrão: %+v %v", c, err)
	}
	for name, over := range map[string]map[string]string{
		"senha curta":         {"BRIDGE_SECRET": "curta"},
		"intervalo de 10 min": {"MIN_GAP": "10m"},
		"aviso sem https":     {"CALLBACK_URL": "http://cloud.example/x", "CALLBACK_SECRET": string(secret)},
		"aviso sem senha":     {"CALLBACK_URL": "https://cloud.example/x"},
	} {
		if _, err := loadConfig(get(over)); err == nil {
			t.Errorf("%s: aceitou", name)
		}
	}
	if c, err := loadConfig(get(map[string]string{"MIN_GAP": "1h"})); err != nil || c.MinGap != time.Hour {
		t.Fatalf("1h: %v", err)
	}
	if c.SessionGap != 5*time.Minute || c.DevicePlatform != "desktop" {
		t.Fatalf("padrão do número: %v %s", c.SessionGap, c.DevicePlatform)
	}
	for name, over := range map[string]map[string]string{
		"número a cada minuto": {"SESSION_GAP": "1m"},
		"ícone que não existe": {"DEVICE_PLATFORM": "blup"},
	} {
		if _, err := loadConfig(get(over)); err == nil {
			t.Errorf("%s: aceitou", name)
		}
	}
}

func testLedger(t *testing.T) *ledger {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	l := &ledger{db: db}
	if err := l.migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return l
}

// Modo segurança: um post por Canal a cada 30 minutos; o post que falha
// devolve a vez; Canais diferentes não se atrapalham.
func TestGapPerChannel(t *testing.T) {
	ctx, l, gap := context.Background(), testLedger(t), 30*time.Minute
	t0 := time.Unix(1_800_000_000, 0)
	if ok, _, _, err := l.claim(ctx, "s1", "111111", t0, gap); !ok || err != nil {
		t.Fatalf("primeiro: %v %v", ok, err)
	}
	ok, wait, _, _ := l.claim(ctx, "s1", "111111", t0.Add(10*time.Minute), gap)
	if ok || wait != 20*time.Minute {
		t.Fatalf("10 min depois: ok=%v espera=%v", ok, wait)
	}
	if ok, _, _, _ := l.claim(ctx, "s1", "222222", t0.Add(time.Minute), gap); !ok {
		t.Fatal("outro Canal ficou preso")
	}
	t1 := t0.Add(31 * time.Minute)
	ok, _, prev, _ := l.claim(ctx, "s1", "111111", t1, gap)
	if !ok || prev != t0.UnixMilli() {
		t.Fatalf("31 min depois: %v %d", ok, prev)
	}
	// O envio falhou: a vez volta, e logo depois pode tentar de novo.
	if err := l.release(ctx, "s1", "111111", t1, prev); err != nil {
		t.Fatal(err)
	}
	if ok, _, _, _ := l.claim(ctx, "s1", "111111", t1.Add(time.Second), gap); !ok {
		t.Fatal("a vez não voltou")
	}
}

func TestSentOnceAndForget(t *testing.T) {
	ctx, l := context.Background(), testLedger(t)
	at := time.UnixMilli(1_800_000_000_000)
	if _, ok, _ := l.sent(ctx, "post:1"); ok {
		t.Fatal("achou post que não existe")
	}
	_ = l.markSent(ctx, "post:1", "s1", Sent{ID: "ABC", At: at})
	got, ok, err := l.sent(ctx, "post:1")
	if !ok || err != nil || got.ID != "ABC" || !got.At.Equal(at) {
		t.Fatalf("%+v %v %v", got, ok, err)
	}
	_ = l.saveSession(ctx, "s1", "5585@s.whatsapp.net", at)
	if err := l.forget(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if list, _ := l.sessions(ctx); len(list) != 0 {
		t.Fatal("sessão não foi esquecida")
	}
	if _, ok, _ := l.sent(ctx, "post:1"); ok {
		t.Fatal("posts da sessão ficaram")
	}
}

// A miniatura respeita a proporção e cabe em 100 pixels.
func TestThumbnail(t *testing.T) {
	pic := image.NewRGBA(image.Rect(0, 0, 640, 360))
	small, err := jpeg.Decode(bytes.NewReader(thumbnail(pic)))
	if err != nil {
		t.Fatal(err)
	}
	if b := small.Bounds(); b.Dx() != 100 || b.Dy() != 56 {
		t.Fatalf("tamanho %v", b)
	}
	tall, _ := jpeg.Decode(bytes.NewReader(thumbnail(image.NewRGBA(image.Rect(0, 0, 300, 600)))))
	if b := tall.Bounds(); b.Dx() != 50 || b.Dy() != 100 {
		t.Fatalf("em pé %v", b)
	}
}

func TestPostgresPlaceholders(t *testing.T) {
	l := &ledger{postgres: true}
	if got := l.q(`SELECT a FROM b WHERE c = ? AND d = ?`); got != `SELECT a FROM b WHERE c = $1 AND d = $2` {
		t.Fatal(got)
	}
}

// fake faz o papel do WhatsApp nas rotas.
type fake struct {
	posts []Post
	err   error
}

func (f *fake) Start(context.Context, string) (View, error) { return View{State: statePairing}, f.err }
func (f *fake) View(string) (View, error)                   { return View{State: stateConnected}, f.err }
func (f *fake) Channels(context.Context, string) ([]Channel, error) {
	return []Channel{{ID: "120363422259812011", Name: "BuildFeed", Role: "owner"}}, f.err
}
func (f *fake) Publish(_ context.Context, _ string, p Post) (Sent, error) {
	f.posts = append(f.posts, p)
	return Sent{ID: "MSG"}, f.err
}
func (f *fake) Remove(context.Context, string) error { return f.err }

func call(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, signed(t, method, path, raw, time.Now()))
	return w
}

var png1 = func() string {
	b, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==")
	return base64.StdEncoding.EncodeToString(b)
}()

func TestRoutes(t *testing.T) {
	f := &fake{}
	h := routes(f, newVerifier(secret, time.Now), slog.New(slog.NewTextHandler(io.Discard, nil)))
	// /health não pede assinatura; o resto pede.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
	if w.Code != 200 {
		t.Fatal("health", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/v1/sessions/abcdefgh", nil))
	if w.Code != 401 {
		t.Fatal("sem assinatura", w.Code)
	}
	if w := call(t, h, "GET", "/v1/sessions/curto", nil); w.Code != 404 {
		t.Fatal("sessão fora do formato", w.Code)
	}
	if w := call(t, h, "GET", "/v1/sessions/abcdefgh/channels", nil); w.Code != 200 || !strings.Contains(w.Body.String(), "BuildFeed") {
		t.Fatal("canais", w.Code, w.Body)
	}
	ok := map[string]string{"channel": "120363422259812011@newsletter", "key": "post:abc123", "text": "*Oferta*", "image": png1}
	if w := call(t, h, "POST", "/v1/sessions/abcdefgh/posts", ok); w.Code != 200 {
		t.Fatal("post certo", w.Code, w.Body)
	}
	if len(f.posts) != 1 || f.posts[0].Channel != "120363422259812011" || len(f.posts[0].Image) == 0 {
		t.Fatalf("post: %+v", f.posts)
	}
	for name, body := range map[string]map[string]string{
		"grupo":            {"channel": "120363425423775876@g.us", "key": "post:abc123", "text": "x"},
		"sem chave":        {"channel": "120363422259812011", "text": "x"},
		"vazio":            {"channel": "120363422259812011", "key": "post:abc123", "text": " "},
		"imagem que não é": {"channel": "120363422259812011", "key": "post:abc123", "text": "x", "image": base64.StdEncoding.EncodeToString([]byte("<svg/>"))},
		"campo a mais":     {"channel": "120363422259812011", "key": "post:abc123", "text": "x", "to": "1"},
		"texto enorme":     {"channel": "120363422259812011", "key": "post:abc123", "text": strings.Repeat("a", maxText+1)},
	} {
		if w := call(t, h, "POST", "/v1/sessions/abcdefgh/posts", body); w.Code != 400 {
			t.Errorf("%s: %d", name, w.Code)
		}
	}
	f.err = TooSoon{Wait: 90 * time.Second}
	w = call(t, h, "POST", "/v1/sessions/abcdefgh/posts", ok)
	if w.Code != 429 || w.Header().Get("Retry-After") != "90" {
		t.Fatal("cedo demais", w.Code, w.Header())
	}
	for err, status := range map[error]int{errUnknown: 404, errNotConnected: 409, errNotAdmin: 403, errBusy: 503, io.ErrUnexpectedEOF: 502} {
		f.err = err
		if w := call(t, h, "GET", "/v1/sessions/abcdefgh/channels", nil); w.Code != status {
			t.Errorf("%v: %d, esperava %d", err, w.Code, status)
		}
	}
	f.err = nil
	if w := call(t, h, "DELETE", "/v1/sessions/abcdefgh", nil); w.Code != 204 {
		t.Fatal("remover", w.Code)
	}
}
