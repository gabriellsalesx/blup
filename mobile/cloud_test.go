package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

// The token only travels over HTTPS; plain HTTP only to the phone itself.
func TestSafeURL(t *testing.T) {
	for raw, want := range map[string]bool{
		"https://cloud.getblup.io":           true,
		"https://getblup.io/device?code=AB":  true,
		"http://localhost:18820":             true,
		"http://127.0.0.1:18820":             true,
		"http://[::1]:18820":                 true,
		"http://cloud.getblup.io":            false,
		"http://10.0.2.2:18820":              false,
		"https://user:pass@cloud.getblup.io": false,
		"intent://x#Intent;scheme=https;end": false,
		"file:///data/data/io.getblup.notes": false,
		"javascript:alert(1)":                false,
		"cloud.getblup.io":                   false,
		"":                                   false,
	} {
		if got := safeURL(raw); got != want {
			t.Errorf("safeURL(%q) = %v, want %v", raw, got, want)
		}
	}
}

// A build given an http:// Cloud refuses it before sending anything.
func TestPrepareRefusesPlainHTTP(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BLUP_CLOUD_URL", "")
	old := cloudURL
	defer func() { cloudURL = old }()
	cloudURL = "http://cloud.getblup.io"
	if err := prepare(); err == nil {
		t.Fatal("prepare accepted an http:// Blup Cloud")
	}
	cloudURL = ""
	if err := prepare(); err != nil {
		t.Fatalf("prepare with the default Cloud: %v", err)
	}
}

// A login page that is not a web page never opens outside the app.
func TestLoginRefusesOtherSchemes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BLUP_CLOUD_URL", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/device/code" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"device_code": "dc", "user_code": "ABCDEFGH", "interval": 5, "expires_in": 600,
			"verification_uri":          "intent://evil#Intent;end",
			"verification_uri_complete": "intent://evil#Intent;end",
		})
	}))
	defer srv.Close()
	old := cloudURL
	defer func() { cloudURL = old }()
	cloudURL = srv.URL
	s := &CloudService{}
	if _, err := s.Login(t.Context()); err == nil || !strings.Contains(err.Error(), "intent://") {
		t.Fatalf("Login = %v, want the unsafe address refused", err)
	}
}

// Syncing a vault that is not linked says so and asks nothing of the Cloud
// (before, it looked like the vault had been deleted elsewhere).
func TestSyncNotLinked(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BLUP_CLOUD_URL", "")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, `{"code":"NOT_FOUND"}`, http.StatusNotFound)
	}))
	defer srv.Close()
	old := cloudURL
	defer func() { cloudURL = old }()
	cloudURL = srv.URL
	if err := prepare(); err != nil {
		t.Fatal(err)
	}
	creds := config.Credentials{BlupToken: "tok", NotesKey: base64.StdEncoding.EncodeToString(make([]byte, 32))}
	if err := creds.Save(); err != nil {
		t.Fatal(err)
	}
	r, err := (&CloudService{}).Sync(t.Context())
	if err == nil || r.Gone {
		t.Fatalf("Sync = %+v, %v; want the not linked error", r, err)
	}
	if n := calls.Load(); n != 0 {
		t.Fatalf("the Cloud got %d requests", n)
	}
}

// A sync parada por segurança chega à tela com o aviso e, no JSON que o
// Wails manda como cause, stuck: true (o que mostra "Comparar do zero").
func TestStuckErrorReachesTheScreen(t *testing.T) {
	defer useLang(appLang())
	for lang, want := range map[string]string{"pt": "Nada foi apagado.", "en": "Nothing was deleted.", "es": "No se borró nada."} {
		useLang(lang)
		var err error = stuckError{Stuck: true}
		if !strings.HasSuffix(err.Error(), want) {
			t.Errorf("%s: %q", lang, err.Error())
		}
		b, jerr := json.Marshal(&err)
		if jerr != nil || string(b) != `{"stuck":true}` {
			t.Errorf("cause = %s, %v", b, jerr)
		}
	}
}

func TestFromNotes(t *testing.T) {
	got := fromNotes("https://cloud.getblup.io/device?user_code=ABCD1234")
	if got != "https://cloud.getblup.io/device?app=notes&user_code=ABCD1234" {
		t.Fatalf("got %s", got)
	}
	if !safeURL(got) {
		t.Fatal("the marked address must still be a safe one")
	}
}
