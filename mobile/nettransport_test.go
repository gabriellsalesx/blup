package main

import (
	"errors"
	"net/http"
	"strings"
	"syscall"
	"testing"
)

type fakeTrip struct {
	errs   []error
	calls  int
	closed int
}

func (f *fakeTrip) RoundTrip(*http.Request) (*http.Response, error) {
	f.calls++
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		if err != nil {
			return nil, err
		}
	}
	return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
}
func (f *fakeTrip) CloseIdleConnections() { f.closed++ }

func TestReviveRetriesReadsOnce(t *testing.T) {
	aborted := &errorsWrap{syscall.ECONNABORTED}
	get, _ := http.NewRequest(http.MethodGet, "https://cloud.getblup.io/api/me", nil)

	f := &fakeTrip{errs: []error{aborted}}
	if _, err := (&revive{base: f}).RoundTrip(get); err != nil || f.calls != 2 || f.closed != 1 {
		t.Fatalf("a dead connection on a GET: err %v, calls %d, closed %d", err, f.calls, f.closed)
	}
	// Only once: a second failure is the answer.
	f = &fakeTrip{errs: []error{aborted, aborted}}
	if _, err := (&revive{base: f}).RoundTrip(get); err == nil || f.calls != 2 {
		t.Fatalf("retried more than once: calls %d", f.calls)
	}
	// A request that changes something is never sent twice.
	post, _ := http.NewRequest(http.MethodPost, "https://cloud.getblup.io/api/auth/device/token", strings.NewReader("{}"))
	f = &fakeTrip{errs: []error{aborted}}
	if _, err := (&revive{base: f}).RoundTrip(post); err == nil || f.calls != 1 {
		t.Fatalf("POST sent again: calls %d", f.calls)
	}
	// Other errors (no network at all, a refused name) are not retried.
	f = &fakeTrip{errs: []error{errors.New("no such host")}}
	if _, err := (&revive{base: f}).RoundTrip(get); err == nil || f.calls != 1 {
		t.Fatalf("other error retried: calls %d", f.calls)
	}
}

// errorsWrap wraps an errno as net and url errors do.
type errorsWrap struct{ err error }

func (e *errorsWrap) Error() string { return "read tcp: " + e.err.Error() }
func (e *errorsWrap) Unwrap() error { return e.err }
