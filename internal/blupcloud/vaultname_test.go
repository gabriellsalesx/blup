package blupcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Nomes de cofre: criados já amarrados ao id, os antigos (formato 1)
// selados de novo uma vez ao ligar o cofre, e abrir um nome nunca escreve.
func TestVaultNamesBoundAndUpgradedOnce(t *testing.T) {
	keys := testKeys(t)
	const id = "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"
	var mu sync.Mutex
	var name []byte
	patches := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var in struct{ Name []byte }
		json.NewDecoder(r.Body).Decode(&in)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/sync/vaults":
			name = in.Name
			json.NewEncoder(w).Encode(map[string]any{"id": id, "head": 0})
		case r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/"+id):
			name = in.Name
			patches++
			w.Write([]byte(`{"ok":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/sync/vaults":
			json.NewEncoder(w).Encode(map[string]any{"vaults": []map[string]any{{"id": id, "name": name, "head": 1}}})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}
	ctx := context.Background()

	if _, err := c.CreateNamedVault(ctx, keys, "Trabalho"); err != nil {
		t.Fatal(err)
	}
	if got, legacy, err := keys.OpenName(id, name); err != nil || legacy || got != "Trabalho" || patches != 1 {
		t.Fatalf("created: %q %v %v (%d patches)", got, legacy, err, patches)
	}
	// Já no formato 2: ligar o cofre não escreve de novo.
	if err := c.UpgradeVaultName(ctx, keys, id); err != nil || patches != 1 {
		t.Fatalf("upgrade of a v2 name wrote: %v %d", err, patches)
	}
	// Um nome antigo é selado de novo uma vez, e depois não mais.
	name, _ = keys.SealLegacyName("Antigo")
	for range 2 {
		if err := c.UpgradeVaultName(ctx, keys, id); err != nil {
			t.Fatal(err)
		}
	}
	if got, legacy, err := keys.OpenName(id, name); err != nil || legacy || got != "Antigo" || patches != 2 {
		t.Fatalf("upgraded: %q %v %v (%d patches)", got, legacy, err, patches)
	}
}
