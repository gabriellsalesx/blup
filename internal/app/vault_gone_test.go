package app

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/config"
)

// A vault deleted on another device: this one lets go of it once, keeps
// the notes, and never makes a new vault by itself.
func TestSyncVaultGone(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const vault = "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"
	var mu sync.Mutex
	listed, created := true, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		p := strings.TrimPrefix(r.URL.Path, "/api")
		switch {
		case p == "/me":
			w.Write([]byte(`{"id":"u","plan":{"name":"free","limits":{"maxFileMB":10}}}`))
		case p == "/sync/vaults" && r.Method == http.MethodPost:
			created++
			w.Write([]byte(`{"id":"0190a1b2-c3d4-7e5f-8a9b-000000000002","head":0}`))
		case p == "/sync/vaults" && listed:
			w.Write([]byte(`{"vaults":[{"id":"` + vault + `"}]}`))
		case p == "/sync/vaults":
			w.Write([]byte(`{"vaults":[]}`))
		default:
			w.WriteHeader(404)
			w.Write([]byte(`{"code":"NOT_FOUND"}`))
		}
	}))
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	master, _, _, err := cloudcrypt.NewKeys("senha")
	if err != nil {
		t.Fatal(err)
	}
	if err := (config.Credentials{BlupToken: "tok", NotesKey: base64.StdEncoding.EncodeToString(master), NotesAccount: "u"}).Save(); err != nil {
		t.Fatal(err)
	}
	site := settingsSite(t)
	site.Sync = config.NoteSync{Remote: RemoteBlup, Vault: vault}
	site.Save()
	os.WriteFile(filepath.Join(site.ContentDir(), "Nota.md"), []byte("fica"), 0o644)
	state := filepath.Join(site.Dir(), ".blup", "sync-state.json")
	os.MkdirAll(filepath.Dir(state), 0o755)
	os.WriteFile(state, []byte(`{}`), 0o644)
	ctx := context.Background()

	// Still in the account's list (a hiccup): nothing is let go of.
	if _, err := SyncNow(ctx, site); err == nil || errors.Is(err, ErrVaultGone) {
		t.Fatalf("listed: %v", err)
	}
	if s, _ := config.LoadSite(site.Dir()); s.Sync.Vault != vault {
		t.Fatal("let go of a vault still listed")
	}

	// Gone from the list: let go of, said in words, the notes stay.
	listed = false
	_, err = SyncNow(ctx, site)
	if !errors.Is(err, ErrVaultGone) || !strings.Contains(err.Error(), "Blup Cloud") {
		t.Fatalf("gone: %v", err)
	}
	s, _ := config.LoadSite(site.Dir())
	if s.Sync.Vault != "" || s.Sync.Remote != RemoteBlup {
		t.Fatalf("sync = %+v", s.Sync)
	}
	if b, _ := os.ReadFile(filepath.Join(site.ContentDir(), "Nota.md")); string(b) != "fica" {
		t.Fatal("the notes went too")
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("the old sync state stayed")
	}

	// Looking (the sync panel opening) makes no vault; only Sync does.
	if info, err := SyncStatus(ctx, s); err != nil || info.Send == 0 || created != 0 {
		t.Fatalf("status: %+v %v, created %d", info, err, created)
	}
	if StartLive(s, func(LiveState) {}) != nil {
		t.Fatal("live sync started without a vault")
	}
}
