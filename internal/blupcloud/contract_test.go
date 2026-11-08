package blupcloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// The device sign in against a fake cloud that answers as Better Auth's
// device authorization plugin does: the code, then "pending", a "slow_down",
// a hiccup of the cloud (503) and at last the token.
func TestDeviceLogin(t *testing.T) {
	var polls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "" || body["client_id"] != ClientID {
			t.Errorf("%s %s auth=%q body=%v", r.Method, r.URL.Path, r.Header.Get("Authorization"), body)
		}
		switch r.URL.Path {
		case "/api/auth/device/code":
			json.NewEncoder(w).Encode(map[string]any{
				"device_code": "dev-123", "user_code": "ABCDEFGH",
				"verification_uri": "https://cloud.example/device", "verification_uri_complete": "https://cloud.example/device?user_code=ABCDEFGH",
				"interval": 5, "expires_in": 900,
			})
		case "/api/auth/device/token":
			if body["grant_type"] != "urn:ietf:params:oauth:grant-type:device_code" || body["device_code"] != "dev-123" {
				t.Errorf("token body %v", body)
			}
			switch polls.Add(1) {
			case 1:
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"authorization_pending","error_description":"..."}`))
			case 2:
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"slow_down"}`))
			case 3:
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"code":"INTERNAL"}`))
			default:
				w.Write([]byte(`{"access_token":"session-token","token_type":"Bearer"}`))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	l, err := StartLogin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if l.Code != "ABCD-EFGH" || !strings.HasSuffix(l.URL, "user_code=ABCDEFGH") || l.interval != 5*time.Second {
		t.Fatalf("login = %+v", l)
	}
	l.interval = time.Millisecond // the test doesn't wait 5 s per poll
	old := slowDown
	slowDown = time.Millisecond
	defer func() { slowDown = old }()
	tok, err := l.Wait(ctx)
	if err != nil || tok != "session-token" || polls.Load() != 4 {
		t.Fatalf("Wait = %q, %v after %d polls", tok, err, polls.Load())
	}
}

// A refusal or an expired code ends the wait; a bad request is not retried.
func TestDeviceLoginEnds(t *testing.T) {
	for _, c := range []struct {
		status int
		body   string
		want   error
	}{
		{400, `{"error":"access_denied"}`, ErrDenied},
		{400, `{"error":"expired_token"}`, ErrExpired},
		{400, `{"error":"invalid_grant"}`, nil},
	} {
		var polls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			polls.Add(1)
			w.WriteHeader(c.status)
			w.Write([]byte(c.body))
		}))
		l := &Login{deviceCode: "d", interval: time.Millisecond, expires: time.Now().Add(time.Minute), c: &Client{BaseURL: srv.URL, HTTP: srv.Client()}}
		_, err := l.Wait(context.Background())
		srv.Close()
		if c.want != nil && !errors.Is(err, c.want) || c.want == nil && err == nil || polls.Load() != 1 {
			t.Errorf("%s: err = %v after %d polls", c.body, err, polls.Load())
		}
	}
}

// Offline the whole time: the wait goes on until the code expires, then
// says so (not the network error).
func TestDeviceLoginOfflineUntilExpired(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	base := srv.URL
	srv.Close() // nothing listens there now
	l := &Login{deviceCode: "d", interval: time.Millisecond, expires: time.Now().Add(150 * time.Millisecond), c: &Client{BaseURL: base, HTTP: &http.Client{Timeout: time.Second}}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := l.Wait(ctx); !errors.Is(err, ErrExpired) {
		t.Fatalf("err = %v", err)
	}
}

// The live room: joined with the session as a Bearer token on
// /api/sync/live, and its refusals told apart.
func TestListenContract(t *testing.T) {
	vault := "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sync/live" {
			http.NotFound(w, r)
			return
		}
		switch r.Header.Get("Authorization") {
		case "Bearer gone":
			w.WriteHeader(http.StatusUnauthorized)
			return
		case "Bearer free":
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"code":"NOT_LIVE"}`))
			return
		case "Bearer tok":
		default:
			t.Errorf("auth %q", r.Header.Get("Authorization"))
			return
		}
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "Blup Studio (") {
			t.Errorf("user agent %q", r.Header.Get("User-Agent"))
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		conn.Write(r.Context(), websocket.MessageText, []byte(`{"vault":"`+vault+`","n":2}`))
		conn.Close(websocket.StatusNormalClosure, "")
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for tok, want := range map[string]error{"gone": ErrSignedOut, "free": ErrNotLive} {
		c := &Client{BaseURL: srv.URL, Token: tok, HTTP: srv.Client()}
		if err := c.Listen(ctx, func(Moved) { t.Error("message on a refused room") }); !errors.Is(err, want) {
			t.Errorf("%s: err = %v", tok, err)
		}
	}
	var got []Moved
	c := &Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}
	c.Listen(ctx, func(m Moved) { got = append(got, m) })
	if len(got) != 1 || got[0] != (Moved{Vault: vault, N: 2}) {
		t.Fatalf("got %+v", got)
	}
}

// A room that goes quiet (a network that dropped without closing) is left,
// so the caller reconnects; the pings ("ping", answered "pong") keep a
// healthy one open.
func TestListenQuietRoom(t *testing.T) {
	oldPing, oldIdle := livePing, liveIdle
	livePing, liveIdle = 20*time.Millisecond, 200*time.Millisecond
	defer func() { livePing, liveIdle = oldPing, oldIdle }()

	answer := atomic.Bool{}
	answer.Store(true)
	var pings atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			if string(data) == "ping" {
				pings.Add(1)
				if answer.Load() {
					conn.Write(r.Context(), websocket.MessageText, []byte("pong"))
				}
			}
		}
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}

	// Answered: still open after several idle windows, until ctx ends.
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	err := c.Listen(ctx, func(Moved) {})
	ended := ctx.Err()
	cancel()
	if ended == nil || pings.Load() < 5 {
		t.Fatalf("healthy room left early: err = %v, %d pings", err, pings.Load())
	}

	// Not answered: left after liveIdle, with ctx still running.
	answer.Store(false)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	err = c.Listen(ctx, func(Moved) {})
	if err == nil || ctx.Err() != nil || time.Since(start) > 2*time.Second {
		t.Fatalf("quiet room: err = %v after %s", err, time.Since(start))
	}
}

// A repeated reminder id is sent once: the cloud would refuse the list.
func TestSetRemindersDedup(t *testing.T) {
	var sent struct {
		Reminders []Reminder `json:"reminders"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/reminders" || r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&sent)
		w.Write([]byte(`{"limit":50,"used":1,"saved":2}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}
	a, b := strings.Repeat("a", 32), strings.Repeat("b", 32)
	u, err := c.SetReminders(context.Background(), []Reminder{{a, 1}, {b, 2}, {a, 3}})
	if err != nil || u.Limit != 50 || u.Used != 1 {
		t.Fatalf("use = %+v, %v", u, err)
	}
	if len(sent.Reminders) != 2 || sent.Reminders[0] != (Reminder{a, 1}) || sent.Reminders[1] != (Reminder{b, 2}) {
		t.Fatalf("sent %+v", sent.Reminders)
	}
	// None: an empty list (never null), which cancels them all.
	if _, err := c.SetReminders(context.Background(), nil); err != nil || sent.Reminders == nil || len(sent.Reminders) != 0 {
		t.Fatalf("empty list: %+v, %v", sent.Reminders, err)
	}
}
