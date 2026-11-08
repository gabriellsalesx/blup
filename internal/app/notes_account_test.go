package app

import (
	"context"
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
	// The account's vaults (id -> sealed name) and whether it has a key.
	vaults := map[string][]byte{}
	hasKey := true
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mu.Lock()
		id, list, keyed := me, map[string][]byte{}, hasKey
		for k, v := range vaults {
			list[k] = v
		}
		mu.Unlock()
		rw.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/me":
			json.NewEncoder(rw).Encode(map[string]any{"id": id, "plan": map[string]any{"name": "free", "label": "Grátis", "limits": map[string]any{"devices": -1}}, "usage": map[string]any{}})
		case "/api/sync/key":
			if !keyed {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{"code":"NO_KEY"}`))
				return
			}
			json.NewEncoder(rw).Encode(map[string]any{"wrapped": w})
		case "/api/sync/vaults":
			out := []map[string]any{}
			for k, v := range list {
				out = append(out, map[string]any{"id": k, "name": v, "head": 1})
			}
			json.NewEncoder(rw).Encode(map[string]any{"vaults": out})
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

	// Chave antiga, sem conta, deixada por outra conta: a conta logada tem
	// cofres e a chave não abre o nome de nenhum. Não é dela: sai, em vez
	// de ser adotada e cifrar os cofres dela com a chave de outra pessoa.
	other, _ := base64.StdEncoding.DecodeString(key)
	otherKeys, err := cloudcrypt.Derive(other)
	if err != nil {
		t.Fatal(err)
	}
	mine := make([]byte, cloudcrypt.KeySize)
	mine[0] = 1
	mineKeys, _ := cloudcrypt.Derive(mine)
	const v1 = "0199f2a4-7c3e-7b1a-9d2e-111111111111"
	name, _ := mineKeys.SealName(v1, "Diário")
	mu.Lock()
	vaults[v1] = name
	mu.Unlock()
	if err := (config.Credentials{BlupToken: "tok", NotesKey: key}).Save(); err != nil {
		t.Fatal(err)
	}
	if notes, c := state(); notes != "locked" || c.NotesKey != "" || c.NotesAccount != "" {
		t.Fatalf("foreign legacy key adopted: %s %+v", notes, c)
	}
	// A chave da própria conta (abre um dos nomes) continua adotada.
	legacy, _ := otherKeys.SealLegacyName("Trabalho")
	mu.Lock()
	vaults["0199f2a4-7c3e-7b1a-9d2e-222222222222"] = legacy
	mu.Unlock()
	if err := (config.Credentials{BlupToken: "tok", NotesKey: key}).Save(); err != nil {
		t.Fatal(err)
	}
	if notes, c := state(); notes != "ready" || c.NotesKey != key || c.NotesAccount != beto {
		t.Fatalf("own legacy key: %s %+v", notes, c)
	}
	// Uma conta sem chave das notas não pode ser dona de chave nenhuma.
	mu.Lock()
	hasKey = false
	clear(vaults)
	mu.Unlock()
	if err := (config.Credentials{BlupToken: "tok", NotesKey: key}).Save(); err != nil {
		t.Fatal(err)
	}
	if notes, c := state(); notes != "new" || c.NotesKey != "" {
		t.Fatalf("key kept for an account without key: %s %+v", notes, c)
	}
}

// Criar a senha das notas sem conseguir ler a conta não deixa uma chave
// sem dona (que outra conta adotaria depois): nada vai para a nuvem.
func TestCreateNotesPasswordNeedsAccount(t *testing.T) {
	var saved bool
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sync/key" && r.Method == http.MethodPut {
			saved = true
			rw.Write([]byte(`{"ok":true}`))
			return
		}
		rw.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	if err := (config.Credentials{BlupToken: "tok"}).Save(); err != nil {
		t.Fatal(err)
	}
	if r, err := CreateNotesPassword(context.Background(), "senha das notas"); err == nil || r != "" {
		t.Fatalf("created without the account: %q %v", r, err)
	}
	if saved {
		t.Fatal("the wrapped key reached the cloud")
	}
	if c, _ := config.LoadCredentials(); c.NotesKey != "" || c.NotesAccount != "" {
		t.Fatalf("kept %+v", c)
	}
}
