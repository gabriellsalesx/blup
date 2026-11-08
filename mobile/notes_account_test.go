package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/config"
)

// A chave das notas do celular é da conta que a abriu: outra conta que
// entre no mesmo aparelho não a herda; uma chave antiga, sem conta, é
// adotada pela conta logada.
func TestNotesKeyBoundToAccount(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BLUP_CLOUD_URL", "")
	const ana, beto = "0199f2a4-7c3e-7b1a-9d2e-4f5a6b7c8d9e", "0199f2a4-7c3e-7b1a-9d2e-000000000000"
	_, _, w, err := cloudcrypt.NewKeys("senha das notas")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	me := ana
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mu.Lock()
		id := me
		mu.Unlock()
		rw.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/me":
			json.NewEncoder(rw).Encode(map[string]any{"id": id, "plan": map[string]any{"name": "free", "label": "Grátis", "limits": map[string]any{"devices": -1}}, "usage": map[string]any{}})
		case "/api/sync/key":
			json.NewEncoder(rw).Encode(map[string]any{"wrapped": w})
		case "/api/sync/vaults":
			// No vault yet: nothing tells the legacy key is someone
			// else's, so it is adopted (app.BindNotesKey).
			rw.Write([]byte(`{"vaults":[]}`))
		default:
			http.NotFound(rw, r)
		}
	}))
	defer srv.Close()
	old := cloudURL
	defer func() { cloudURL = old }()
	cloudURL = srv.URL
	if err := prepare(); err != nil {
		t.Fatal(err)
	}
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	if err := (config.Credentials{BlupToken: "tok", NotesKey: key}).Save(); err != nil {
		t.Fatal(err)
	}
	s := &CloudService{}
	check := func(step, notes, account string, kept bool) {
		t.Helper()
		a, err := s.Account(t.Context())
		if err != nil {
			t.Fatalf("%s: %v", step, err)
		}
		c, _ := config.LoadCredentials()
		if a.Notes != notes || c.NotesAccount != account || (c.NotesKey == key) != kept {
			t.Fatalf("%s: notes %s, %+v", step, a.Notes, c)
		}
		dir, _ := config.Dir()
		if info, err := os.Stat(filepath.Join(dir, "credentials.toml")); err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("%s: credentials.toml mode %v %v", step, info, err)
		}
	}
	check("legacy key adopted", "ready", ana, true)
	check("same account", "ready", ana, true)
	mu.Lock()
	me = beto
	mu.Unlock()
	check("other account", "locked", "", false)
	if err := s.Unlock(t.Context(), "senha das notas"); err != nil {
		t.Fatal(err)
	}
	c, _ := config.LoadCredentials()
	if c.NotesKey == "" || c.NotesAccount != beto {
		t.Fatalf("unlocked: %+v", c)
	}
}
