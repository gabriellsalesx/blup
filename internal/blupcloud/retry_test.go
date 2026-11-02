package blupcloud

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSleep records the waits instead of sleeping.
func fakeSleep(t *testing.T) *[]time.Duration {
	var waits []time.Duration
	old := retrySleep
	retrySleep = func(ctx context.Context, d time.Duration) error {
		waits = append(waits, d)
		return ctx.Err()
	}
	t.Cleanup(func() { retrySleep = old })
	return &waits
}

// busyThen answers 429 (with Retry-After when given) n times, then 200.
func busyThen(n int32, retryAfter string, calls *atomic.Int32, bodies *[]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if bodies != nil {
			*bodies = append(*bodies, string(b))
		}
		if calls.Add(1) <= n {
			if retryAfter != "" {
				w.Header().Set("Retry-After", retryAfter)
			}
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"code":"TOO_MANY_REQUESTS"}`))
			return
		}
		w.Write([]byte(`{"missing":[]}`))
	}))
}

func TestRetryOn429(t *testing.T) {
	waits := fakeSleep(t)
	var calls atomic.Int32
	var bodies []string
	srv := busyThen(2, "3", &calls, &bodies)
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client(), Token: "t"}
	// A block goes up under its hash: safe to send again, with the same body.
	if err := c.do(context.Background(), http.MethodPut, "/sync/blocks/x", []byte("sealed"), "application/octet-stream", nil); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
	for _, b := range bodies {
		if b != "sealed" {
			t.Fatalf("body sent again = %q", b)
		}
	}
	if len(*waits) != 2 || (*waits)[0] != 3*time.Second {
		t.Fatalf("waits = %v, want Retry-After 3s twice", *waits)
	}
}

func TestRetryGivesUpAfterFourAndCapsWait(t *testing.T) {
	waits := fakeSleep(t)
	var calls atomic.Int32
	srv := busyThen(100, "3600", &calls, nil)
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	err := c.do(context.Background(), http.MethodGet, "/me", nil, "", nil)
	var e *apiError
	if !errors.As(err, &e) || e.Status != http.StatusTooManyRequests || e.Code != "TOO_MANY_REQUESTS" {
		t.Fatalf("err = %v, want the 429", err)
	}
	if calls.Load() != maxAttempts {
		t.Fatalf("calls = %d, want %d", calls.Load(), maxAttempts)
	}
	for _, w := range *waits {
		if w != maxRetryWait {
			t.Fatalf("wait %v, want capped at %v", w, maxRetryWait)
		}
	}
}

func TestRetryWithoutHeaderBacksOff(t *testing.T) {
	waits := fakeSleep(t)
	var calls atomic.Int32
	srv := busyThen(3, "", &calls, nil)
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	// POST .../missing only asks: it may be sent again.
	if err := c.do(context.Background(), http.MethodPost, "/sync/blocks/missing", map[string]any{"ids": []string{}}, "", nil); err != nil {
		t.Fatal(err)
	}
	if len(*waits) != 3 {
		t.Fatalf("waits = %v", *waits)
	}
	for i, w := range *waits {
		base := 500 * time.Millisecond << i
		if w < base || w > base+base/2 {
			t.Fatalf("wait %d = %v, want %v plus up to half", i, w, base)
		}
	}
}

func TestNoRetryForPlainPost(t *testing.T) {
	fakeSleep(t)
	for _, path := range []string{"/sites", "/sync/vaults", "/sites/s1/deploys", "/sync/vaults/v/versions", "/auth/sign-out"} {
		var calls atomic.Int32
		srv := busyThen(1, "1", &calls, nil)
		c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
		err := c.do(context.Background(), http.MethodPost, path, map[string]any{}, "", nil)
		srv.Close()
		if err == nil || calls.Load() != 1 {
			t.Fatalf("%s: err = %v, calls = %d: a POST must not be sent twice", path, err, calls.Load())
		}
	}
}

func TestRetryStopsWithContext(t *testing.T) {
	var calls atomic.Int32
	srv := busyThen(100, "30", &calls, nil)
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := c.do(ctx, http.MethodGet, "/me", nil, "", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want the context's", err)
	}
	if time.Since(start) > 5*time.Second || calls.Load() != 1 {
		t.Fatalf("waited %v with %d calls after the context ended", time.Since(start), calls.Load())
	}
}

func TestDeployBusy(t *testing.T) {
	fakeSleep(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"code":"DEPLOY_BUSY"}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	err := c.do(context.Background(), http.MethodPost, "/sites/s1/deploys", map[string]any{"files": map[string]string{}}, "", nil)
	if !errors.Is(err, ErrDeployBusy) || calls.Load() != 1 {
		t.Fatalf("err = %v, calls = %d: want ErrDeployBusy, no retry", err, calls.Load())
	}
}

func TestRetryWait(t *testing.T) {
	if d := retryWait("7", 0); d != 7*time.Second {
		t.Errorf("seconds: %v", d)
	}
	if d := retryWait("-1", 0); d < 500*time.Millisecond {
		t.Errorf("negative falls back to backoff: %v", d)
	}
	if d := retryWait(time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat), 0); d != 0 {
		t.Errorf("past date: %v", d)
	}
	if d := retryWait("x", 20); d > maxRetryWait {
		t.Errorf("backoff over the cap: %v", d)
	}
}
