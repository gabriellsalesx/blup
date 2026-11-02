package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Forged calls to the bindings, as a page in the webview (the preview, an
// image in a note) could make them, never reach Wails; the window's own
// calls do.
func TestGuardRuntime(t *testing.T) {
	reached := false
	h := guardRuntime(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	call := `/wails/runtime?object=0&method=0&args=` + `%7B%22methodName%22%3A%22main.Studio.OpenTerminal%22%7D`

	cases := []struct {
		name    string
		method  string
		path    string
		headers map[string]string
		body    string
		ok      bool
	}{
		{"img GET with the call in the query", http.MethodGet, call, nil, "", false},
		{"img GET from the preview", http.MethodGet, call, map[string]string{"Sec-Fetch-Site": "cross-site"}, "", false},
		{"no-cors POST without the client id", http.MethodPost, "/wails/runtime", map[string]string{"Content-Type": "text/plain"}, `{"object":0,"method":0}`, false},
		{"POST with the client id from another site", http.MethodPost, "/wails/runtime", map[string]string{"x-wails-client-id": "x", "Sec-Fetch-Site": "cross-site", "Origin": "http://localhost:1313"}, `{}`, false},
		{"POST with the client id from a sibling site", http.MethodPost, "/wails/runtime", map[string]string{"x-wails-client-id": "x", "Sec-Fetch-Site": "same-site"}, `{}`, false},
		{"stream poll from another site", http.MethodGet, "/wails/stream/poll", map[string]string{"Sec-Fetch-Site": "cross-site"}, "", false},
		{"the window's own call", http.MethodPost, "/wails/runtime", map[string]string{"x-wails-client-id": "abc", "Content-Type": "application/json", "Sec-Fetch-Site": "same-origin"}, `{"object":0,"method":0}`, true},
		{"the window's call on an engine without Sec-Fetch-Site", http.MethodPost, "/wails/runtime", map[string]string{"x-wails-client-id": "abc"}, `{}`, true},
		{"the window's page", http.MethodGet, "/index.html", nil, "", true},
		{"the runtime script", http.MethodGet, "/wails/runtime.js", map[string]string{"Sec-Fetch-Site": "same-origin"}, "", true},
		{"a local translation file", http.MethodGet, "/local-translate/x.bin", nil, "", true},
	}
	for _, c := range cases {
		reached = false
		r := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		for k, v := range c.headers {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if reached != c.ok {
			t.Errorf("%s: reached=%v, want %v (status %d)", c.name, reached, c.ok, w.Code)
		}
		if !c.ok && w.Code != http.StatusForbidden {
			t.Errorf("%s: status %d, want 403", c.name, w.Code)
		}
	}
}

// The preview's guard lets the window's origins through, on every system,
// and nothing that only looks like them.
func TestWindowOrigin(t *testing.T) {
	for origin, want := range map[string]bool{
		"wails://localhost":               true,
		"http://wails.localhost":          true,
		"http://wails.localhost:5173":     true,
		"wails://localhost:5173":          true,
		"https://wails.localhost":         true,
		"http://wails.localhost.evil.com": false,
		"http://evil.com":                 false,
		"http://localhost:1313":           false,
		"null":                            false,
		"":                                false,
		"http://user@wails.localhost":     false,
		"http://wails.localhost/path":     false,
	} {
		if windowOrigin(origin) != want {
			t.Errorf("windowOrigin(%q) = %v", origin, !want)
		}
	}
}
