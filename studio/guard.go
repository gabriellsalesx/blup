package main

import (
	"net/http"
	"strings"
)

// guardRuntime keeps the Go methods the window calls (the Wails bindings,
// reached at /wails/runtime) to the window's own code.
//
// Outside its server mode, Wails v3 checks nothing on that address, and it
// also takes a call as a GET with the call in the query string (a fallback
// for WebKitGTK). Any page the webview loads could then call the Studio
// with a plain <img src=".../wails/runtime?object=...">: the site's preview
// (a note's image, a theme's HTML, a page reached by a link), or an image
// in a note. The window's runtime always calls with fetch: a POST with the
// x-wails-client-id header, which a page of another origin can't send
// without a CORS preflight that the asset server never answers.
func guardRuntime(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/wails/") && crossSite(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.URL.Path == "/wails/runtime" && (r.Method != http.MethodPost || r.Header.Get("x-wails-client-id") == "") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// crossSite tells whether the browser says the request came from another
// origin (Sec-Fetch-Site); engines that don't send it pass.
func crossSite(r *http.Request) bool {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "", "same-origin", "none":
		return false
	}
	return true
}

// assetMiddleware is the asset server's middleware: the guard first, then
// the local translation's files.
func assetMiddleware(next http.Handler) http.Handler {
	return guardRuntime(serveLocalTranslate(next))
}
