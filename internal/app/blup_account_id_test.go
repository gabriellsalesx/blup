package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

// The account's id reaches the Studio, to show and copy when asking for
// help.
func TestBlupAccountID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/me":
			w.Write([]byte(`{"id":"0199f2a4-7c3e-7b1a-9d2e-4f5a6b7c8d9e","email":"ana@example.org","plan":{"name":"free","label":"Grátis","limits":{"sites":0,"devices":-1}},"usage":{}}`))
		case "/api/sync/key":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"code":"NO_KEY"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	if err := (config.Credentials{BlupToken: "tok"}).Save(); err != nil {
		t.Fatal(err)
	}
	a, err := LoadBlupAccount(context.Background())
	if err != nil || !a.SignedIn || a.ID != "0199f2a4-7c3e-7b1a-9d2e-4f5a6b7c8d9e" {
		t.Fatalf("account = %+v, %v", a, err)
	}
}
