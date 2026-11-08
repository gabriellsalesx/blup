package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/config"
)

func TestRenameSite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var sent []byte
	status := 200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/me" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"u","plan":{"name":"free","limits":{}}}`))
			return
		}
		if r.Method == http.MethodPatch && r.URL.Path == "/api/sync/vaults/6f1c1f9e-1b2a-4c3d-9e8f-0a1b2c3d4e5f" {
			sent, _ = io.ReadAll(r.Body)
		}
		w.WriteHeader(status)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	master := make([]byte, cloudcrypt.KeySize)
	if err := (config.Credentials{BlupToken: "tok", NotesKey: base64.StdEncoding.EncodeToString(master), NotesAccount: "u"}).Save(); err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	dir := filepath.Join(base, "site")
	os.MkdirAll(dir, 0o755)
	s := config.NewSite(dir)
	s.Title, s.Model, s.Vault = "Antigo", "mochi", base
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// Not on the Cloud: only here.
	if ok, err := RenameSite(ctx, dir, "  Meu   cofre "); err != nil || !ok || sent != nil {
		t.Fatalf("%v %v %s", ok, err, sent)
	}
	if s, _ := config.LoadSite(dir); s.Title != "Meu cofre" {
		t.Fatalf("title = %q", s.Title)
	}
	if _, err := RenameSite(ctx, dir, "   "); err == nil {
		t.Fatal("an empty name was saved")
	}

	// On the Cloud: the sealed new name goes there too.
	s, _ = config.LoadSite(dir)
	s.Sync.Remote, s.Sync.Vault = config.SyncBlup, "6f1c1f9e-1b2a-4c3d-9e8f-0a1b2c3d4e5f"
	s.Save()
	if ok, err := RenameSite(ctx, dir, "Trabalho"); err != nil || !ok {
		t.Fatalf("%v %v", ok, err)
	}
	var body struct{ Name []byte }
	keys, _ := cloudcrypt.Derive(master)
	if json.Unmarshal(sent, &body) != nil {
		t.Fatalf("sent %s", sent)
	}
	// Sealed bound to the vault's id (format 2): it opens only there.
	if name, legacy, err := keys.OpenName("6f1c1f9e-1b2a-4c3d-9e8f-0a1b2c3d4e5f", body.Name); err != nil || legacy || name != "Trabalho" {
		t.Fatalf("name = %q %v %v", name, legacy, err)
	}
	if _, _, err := keys.OpenName("0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b", body.Name); err == nil {
		t.Fatal("the name opened as another vault's")
	}
	// The Cloud failing doesn't stop the name here.
	status = 500
	if ok, err := RenameSite(ctx, dir, "Casa"); err != nil || ok {
		t.Fatalf("%v %v", ok, err)
	}
	if s, _ := config.LoadSite(dir); s.Title != "Casa" {
		t.Fatalf("title = %q", s.Title)
	}
}

func TestRenameSiteFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	os.MkdirAll(filepath.Join(home, "Documents"), 0o755)
	ctx := context.Background()
	vault := func(notes string) string {
		os.MkdirAll(notes, 0o755)
		os.WriteFile(filepath.Join(notes, "Nota.md"), []byte("oi"), 0o644)
		dir := filepath.Join(t.TempDir(), "site")
		os.MkdirAll(dir, 0o755)
		s := config.NewSite(dir)
		s.Title, s.Kind, s.Model, s.Vault = filepath.Base(notes), config.KindNotes, "mochi", notes
		if err := s.Save(); err != nil {
			t.Fatal(err)
		}
		return dir
	}

	// Blup's own folder follows the name, with what is in it.
	vaults := filepath.Join(home, "Documents", "Blup", "Vaults")
	dir := vault(filepath.Join(vaults, "ok"))
	if _, err := RenameSite(ctx, dir, "o"); err != nil {
		t.Fatal(err)
	}
	s, _ := config.LoadSite(dir)
	if s.Title != "o" || s.Vault != filepath.Join(vaults, "o") {
		t.Fatalf("site = %q %q", s.Title, s.Vault)
	}
	if _, err := os.Stat(filepath.Join(vaults, "o", "Nota.md")); err != nil {
		t.Fatal("the notes didn't go with the folder")
	}
	if _, err := os.Stat(filepath.Join(vaults, "ok")); !os.IsNotExist(err) {
		t.Fatal("the old folder is still there")
	}

	// Taken: nothing changes.
	os.MkdirAll(filepath.Join(vaults, "Trabalho"), 0o755)
	if _, err := RenameSite(ctx, dir, "Trabalho"); err == nil {
		t.Fatal("renamed over another folder")
	}
	if s, _ := config.LoadSite(dir); s.Title != "o" || s.Vault != filepath.Join(vaults, "o") {
		t.Fatalf("changed anyway: %q %q", s.Title, s.Vault)
	}

	// A folder the person picked keeps its name.
	mine := filepath.Join(home, "Minhas notas")
	dir = vault(mine)
	if _, err := RenameSite(ctx, dir, "Outro nome"); err != nil {
		t.Fatal(err)
	}
	if s, _ := config.LoadSite(dir); s.Title != "Outro nome" || s.Vault != mine {
		t.Fatalf("picked folder: %q %q", s.Title, s.Vault)
	}
}
