package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/config"
)

// The cloud keeps the wrapped key and could try passwords offline: a short
// notes password is refused here, before anything goes to the cloud,
// whoever calls (the screen checks it too). Characters, not bytes, count.
func TestNotesPasswordTooShortRefused(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	if err := (config.Credentials{BlupToken: "tok"}).Save(); err != nil {
		t.Fatal(err)
	}
	for _, pw := range []string{"", "1234567", "ççççç", "😀😀😀😀"} {
		if r, err := CreateNotesPassword(context.Background(), pw); err == nil || r != "" {
			t.Fatalf("%q accepted", pw)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("a short password reached the cloud")
	}
	if c, _ := config.LoadCredentials(); c.NotesKey != "" {
		t.Fatal("a key was kept")
	}
	if _, err := CreateNotesPassword(context.Background(), "senha boa"); err != nil {
		t.Fatalf("8 characters refused: %v", err)
	}
	// The master key is on disk only for this user (0600).
	if runtime.GOOS != "windows" {
		dir, _ := config.Dir()
		st, err := os.Stat(filepath.Join(dir, "credentials.toml"))
		if err != nil || st.Mode().Perm() != 0o600 {
			t.Fatalf("credentials.toml mode %v %v", st.Mode(), err)
		}
	}
}

// A password created with spaces around it opens as typed; one typed with
// a stray space or line break around it (pasted) opens too; the recovery
// key opens; anything else is the wrong password.
func TestOpenNotesKey(t *testing.T) {
	master, rec, w, err := cloudcrypt.NewKeys("  com espaços ")
	if err != nil {
		t.Fatal(err)
	}
	for _, typed := range []string{"  com espaços ", rec} {
		if m, err := OpenNotesKey(w, typed); err != nil || string(m) != string(master) {
			t.Fatalf("%q: %v", typed, err)
		}
	}
	master2, _, w2, _ := cloudcrypt.NewKeys("senha das notas")
	if m, err := OpenNotesKey(w2, "senha das notas\n"); err != nil || string(m) != string(master2) {
		t.Fatalf("pasted with a line break: %v", err)
	}
	if _, err := OpenNotesKey(w2, "outra senha"); !errors.Is(err, cloudcrypt.ErrWrongPassword) {
		t.Fatalf("wrong password: %v", err)
	}
}
