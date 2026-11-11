package main

import (
	"errors"
	"io"
	"net/http"
	"syscall"
)

// Android cuts the app's open connections while it is in the background
// (the browser in front to approve the sign in, another app, the screen
// off). Back on screen, a request that took one of those connections from
// the pool failed with "software caused connection abort" and showed as an
// error, though the network was fine. Two things keep that from happening:
// the pool is emptied when the app comes back on screen (dropStaleConns),
// and a request that only reads (GET or HEAD, no body) and found its
// connection dead is sent once more on a new one. A request that changes
// something is never sent twice: it may have been done already.

func init() {
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		http.DefaultTransport = &revive{base: base}
	}
}

// revive is http.DefaultTransport, which every client of the app uses
// (Blup Cloud's included, whose http.Client has no Transport of its own).
type revive struct {
	base interface {
		http.RoundTripper
		CloseIdleConnections()
	}
}

func (r *revive) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := r.base.RoundTrip(req)
	if err == nil || !readOnly(req) || !connDropped(err) || req.Context().Err() != nil {
		return res, err
	}
	r.base.CloseIdleConnections()
	return r.base.RoundTrip(req)
}

func (r *revive) CloseIdleConnections() { r.base.CloseIdleConnections() }

func readOnly(req *http.Request) bool {
	return (req.Method == http.MethodGet || req.Method == http.MethodHead) && (req.Body == nil || req.Body == http.NoBody)
}

// connDropped tells whether err is a connection the system or the other
// side closed under the request.
func connDropped(err error) bool {
	return errors.Is(err, syscall.ECONNABORTED) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}

// dropStaleConns empties the connection pool: called when the app comes
// back on screen, so nothing reuses a connection Android cut meanwhile.
func dropStaleConns() {
	if t, ok := http.DefaultTransport.(interface{ CloseIdleConnections() }); ok {
		t.CloseIdleConnections()
	}
}
