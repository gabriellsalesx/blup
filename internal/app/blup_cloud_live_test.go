//go:build cloudlive

// Runs against a real Blup Cloud (usually wrangler dev on this computer),
// with the session of a new account:
//
//	BLUP_CLOUD_URL=http://localhost:8787 BLUP_CLOUD_TOKEN=<Studio session> \
//	  go test -tags cloudlive -run TestBlupCloudStudio ./internal/app
package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

func TestBlupCloudStudio(t *testing.T) {
	token := os.Getenv("BLUP_CLOUD_TOKEN")
	if token == "" {
		t.Skip("BLUP_CLOUD_TOKEN not set")
	}
	ctx := context.Background()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := (config.Credentials{BlupToken: token}).Save(); err != nil {
		t.Fatal(err)
	}
	a, err := LoadBlupAccount(ctx)
	if err != nil || !a.SignedIn || a.Notes != "new" {
		t.Fatalf("new account = %+v, %v", a, err)
	}
	recovery, err := CreateNotesPassword(ctx, "senha das notas")
	if err != nil || recovery == "" {
		t.Fatal(err)
	}
	if a, _ = LoadBlupAccount(ctx); a.Notes != "ready" {
		t.Fatalf("after creating: %q", a.Notes)
	}

	// Another computer: signed in, the notes still locked.
	creds, _ := config.LoadCredentials()
	creds.NotesKey = ""
	creds.Save()
	if a, _ = LoadBlupAccount(ctx); a.Notes != "locked" {
		t.Fatalf("other computer: %q", a.Notes)
	}
	if err := UnlockNotes(ctx, "senha errada"); err == nil {
		t.Fatal("wrong password accepted")
	}
	if err := UnlockNotes(ctx, recovery); err != nil {
		t.Fatalf("recovery key: %v", err)
	}

	// A site synced through Blup Cloud: the vault is made by the first sync.
	root := t.TempDir()
	notes := filepath.Join(root, "notas")
	os.MkdirAll(notes, 0o755)
	os.WriteFile(filepath.Join(notes, "index.md"), []byte("# Olá\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "site"), 0o755)
	site := config.NewSite(filepath.Join(root, "site"))
	site.Title, site.Vault, site.Model, site.Lang = "Meu blog", notes, "mochi", "pt"
	if err := SetSyncRemote(site, RemoteBlup); err != nil {
		t.Fatal(err)
	}
	info, err := SyncNow(ctx, site)
	if err != nil {
		t.Fatal(err)
	}
	if info.Repo != "Blup Cloud" || info.Send < 1 || site.Sync.Vault == "" {
		t.Fatalf("first sync = %+v, vault %q", info, site.Sync.Vault)
	}
	st, err := SyncStatus(ctx, site)
	if err != nil || !st.UpToDate {
		t.Fatalf("status after sync = %+v, %v", st, err)
	}
	if err := BlupSignOut(ctx); err != nil {
		t.Fatal(err)
	}
	if a, _ = LoadBlupAccount(ctx); a.SignedIn {
		t.Fatal("still signed in")
	}
}
