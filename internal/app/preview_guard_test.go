package app

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// guardInFrontOf is a guard whose "hugo" is backend.
func guardInFrontOf(t *testing.T, backend *httptest.Server, appOrigin func(string) bool) *PreviewGuard {
	t.Helper()
	g, err := NewPreviewGuard(appOrigin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Close() })
	_, port, _ := net.SplitHostPort(backend.Listener.Addr().String())
	g.hugo, _ = strconv.Atoi(port)
	g.srv.Handler = g.Handler()
	return g
}

// The preview answers only requests addressed to itself (DNS rebinding:
// Host evil.com) and, with an Origin, only from the preview or the window.
func TestPreviewGuard(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "page "+r.URL.Path)
	}))
	defer backend.Close()
	wails := func(o string) bool { return o == "wails://localhost" || o == "http://wails.localhost" }
	g := guardInFrontOf(t, backend, wails)
	if !strings.HasPrefix(g.URL, "http://127.0.0.1:") || g.port == g.hugo {
		t.Fatalf("URL = %q", g.URL)
	}
	self := strings.TrimSuffix(g.URL, "/")
	local := "http://" + g.host("localhost")

	cases := []struct {
		name, host, origin string
		ok                 bool
	}{
		{"its own address", g.host("127.0.0.1"), "", true},
		{"localhost", g.host("localhost"), "", true},
		{"its own pages", g.host("127.0.0.1"), self, true},
		{"its own pages by localhost", g.host("localhost"), local, true},
		{"the window (Linux, macOS)", g.host("127.0.0.1"), "wails://localhost", true},
		{"the window (Windows)", g.host("127.0.0.1"), "http://wails.localhost", true},
		{"DNS rebinding", "evil.com:" + strconv.Itoa(g.port), "", false},
		{"DNS rebinding without port", "evil.com", "", false},
		{"another port", "127.0.0.1:1", "", false},
		{"hugo's own port", g.host("127.0.0.1")[:len("127.0.0.1:")] + strconv.Itoa(g.hugo), "", false},
		{"a foreign origin", g.host("127.0.0.1"), "https://evil.com", false},
		{"a sandboxed page", g.host("127.0.0.1"), "null", false},
		{"another local port", g.host("127.0.0.1"), "http://127.0.0.1:1", false},
	}
	for _, c := range cases {
		req, _ := http.NewRequest(http.MethodGet, g.URL+"notes/", nil)
		req.Host = c.host
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if got := res.StatusCode == http.StatusOK && string(body) == "page /notes/"; got != c.ok {
			t.Errorf("%s: status %d %q, want ok=%v", c.name, res.StatusCode, body, c.ok)
		}
	}
}

// The address handed out is 127.0.0.1 (on Windows, localhost may resolve
// to ::1 first, where nothing listens), but localhost works too.
func TestPreviewGuardByName(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") }))
	defer backend.Close()
	g := guardInFrontOf(t, backend, nil)
	for _, u := range []string{g.URL, "http://" + g.host("localhost") + "/"} {
		res, err := http.Get(u)
		if err != nil {
			t.Fatalf("%s: %v", u, err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d", u, res.StatusCode)
		}
	}
	// Without an app, the window's origins are foreign.
	req := httptest.NewRequest(http.MethodGet, g.URL, nil)
	req.Header.Set("Origin", "wails://localhost")
	if g.check(req) == nil {
		t.Error("an origin passed without an app")
	}
}

// hugo listens on 127.0.0.1 only, its pages and live reload point to the
// guard.
func TestPreviewGuardHugoArgs(t *testing.T) {
	g := &PreviewGuard{port: 18810, hugo: 18811, URL: "http://127.0.0.1:18810/"}
	got := strings.Join(g.HugoArgs(), " ")
	want := "--bind 127.0.0.1 --port 18811 --baseURL http://127.0.0.1:18810/ --appendPort=false --liveReloadPort 18810"
	if got != want {
		t.Errorf("args = %q, want %q", got, want)
	}
}
