package app

import (
	"context"
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

// A chave das notas salva no aparelho é da conta que a abriu: outra conta
// que entre no mesmo usuário do sistema não a herda.
func TestNotesKeyBoundToAccount(t *testing.T) {
	const (
		ana  = "0199f2a4-7c3e-7b1a-9d2e-4f5a6b7c8d9e"
		beto = "0199f2a4-7c3e-7b1a-9d2e-000000000000"
	)
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
		default:
			http.NotFound(rw, r)
		}
	}))
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	ctx := context.Background()
	key := "a2V5IGRhIGFuYSBhcXVpIG5vIGFwYXJlbGhvIDMyIGI="
	state := func() (string, config.Credentials) {
		t.Helper()
		a, err := LoadBlupAccount(ctx)
		if err != nil {
			t.Fatal(err)
		}
		c, err := config.LoadCredentials()
		if err != nil {
			t.Fatal(err)
		}
		dir, _ := config.Dir()
		if info, err := os.Stat(filepath.Join(dir, "credentials.toml")); err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("credentials.toml mode: %v %v", info, err)
		}
		return a.Notes, c
	}

	// Chave antiga, sem conta: adotada pela conta logada, sem pedir senha.
	if err := (config.Credentials{BlupToken: "tok", NotesKey: key}).Save(); err != nil {
		t.Fatal(err)
	}
	if notes, c := state(); notes != "ready" || c.NotesKey != key || c.NotesAccount != ana {
		t.Fatalf("legacy key: %s %+v", notes, c)
	}
	// A mesma conta mantém a chave.
	if notes, c := state(); notes != "ready" || c.NotesKey != key || c.NotesAccount != ana {
		t.Fatalf("same account: %s %+v", notes, c)
	}
	// Outra conta entra (a sessão da Ana expirou): a chave sai e as notas
	// ficam trancadas, como num aparelho novo.
	mu.Lock()
	me = beto
	mu.Unlock()
	if notes, c := state(); notes != "locked" || c.NotesKey != "" || c.NotesAccount != "" || c.BlupToken != "tok" {
		t.Fatalf("other account: %s %+v", notes, c)
	}
	// A senha das notas da conta nova grava a chave com a conta dela.
	if err := UnlockNotes(ctx, "senha das notas"); err != nil {
		t.Fatal(err)
	}
	if notes, c := state(); notes != "ready" || c.NotesKey == "" || c.NotesAccount != beto {
		t.Fatalf("unlocked: %s %+v", notes, c)
	}
}
