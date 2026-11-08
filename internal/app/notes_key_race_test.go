package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
)

// Two computers create the notes password at once: the one that loses
// (409 KEY_EXISTS) keeps nothing of the key it made, so it never encrypts
// with a key the cloud doesn't have.
func TestNotesPasswordLostRace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/me" {
			w.Write([]byte(`{"id":"u","plan":{"name":"free","limits":{}}}`))
			return
		}
		if r.Method == http.MethodPut && r.URL.Path == "/api/sync/key" {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"code":"KEY_EXISTS"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	if err := (config.Credentials{BlupToken: "tok"}).Save(); err != nil {
		t.Fatal(err)
	}
	recovery, err := CreateNotesPassword(context.Background(), "senha das notas")
	if err == nil || recovery != "" {
		t.Fatalf("the lost race went on: %q %v", recovery, err)
	}
	if err.Error() != describe(blupcloud.ErrKeyExists) || errors.Is(err, blupcloud.ErrNoKey) {
		t.Fatalf("message: %v", err)
	}
	if c, _ := config.LoadCredentials(); c.NotesKey != "" {
		t.Fatal("the key made here was kept")
	}
}
