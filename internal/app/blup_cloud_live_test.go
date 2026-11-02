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
	"strings"
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
	if _, err := BlupSignOut(ctx); err != nil {
		t.Fatal(err)
	}
	if a, _ = LoadBlupAccount(ctx); a.SignedIn {
		t.Fatal("still signed in")
	}
}

// History on a plan that keeps it (BLUP_CLOUD_LIVE_TOKEN: an account on Pro
// or above, new).
func TestBlupHistory(t *testing.T) {
	token := os.Getenv("BLUP_CLOUD_LIVE_TOKEN")
	if token == "" {
		t.Skip("BLUP_CLOUD_LIVE_TOKEN not set")
	}
	ctx := context.Background()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	(config.Credentials{BlupToken: token}).Save()
	if _, err := CreateNotesPassword(ctx, "senha das notas"); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	notes := filepath.Join(root, "notas")
	os.MkdirAll(filepath.Join(root, "site"), 0o755)
	os.MkdirAll(notes, 0o755)
	site := config.NewSite(filepath.Join(root, "site"))
	site.Title, site.Vault, site.Model, site.Lang = "Histórico", notes, "mochi", "pt"
	SetSyncRemote(site, RemoteBlup)
	note := filepath.Join(notes, "diário.md")
	for _, text := range []string{"um\n", "dois\n", "três\n"} {
		os.WriteFile(note, []byte(text), 0o644)
		if _, err := SyncNow(ctx, site); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(notes, "rascunho.md"), []byte("apagar\n"), 0o644)
	SyncNow(ctx, site)
	os.Remove(filepath.Join(notes, "rascunho.md"))
	SyncNow(ctx, site)

	h, err := NoteHistory(ctx, site, "diário.md")
	if err != nil || len(h) != 3 || !h[0].Current || h[1].Current {
		t.Fatalf("history = %+v, %v", h, err)
	}
	if text, ok, err := NoteVersionText(ctx, site, "diário.md", h[2].N); err != nil || !ok || text != "um\n" {
		t.Fatalf("oldest = %q, %v", text, err)
	}
	if err := RestoreNoteVersion(ctx, site, "diário.md", h[2].N); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(note); string(b) != "um\n" {
		t.Fatalf("restored = %q", b)
	}
	SyncNow(ctx, site)
	if h, _ = NoteHistory(ctx, site, "diário.md"); len(h) != 4 {
		t.Fatalf("restoring is a new version: %d", len(h))
	}

	d, err := DeletedNotes(ctx, site)
	if err != nil || len(d) != 1 || d[0].Path != "rascunho.md" {
		t.Fatalf("deleted = %+v, %v", d, err)
	}
	if err := RestoreNoteVersion(ctx, site, d[0].Path, d[0].N); err != nil {
		t.Fatal(err)
	}
	if d, _ = DeletedNotes(ctx, site); len(d) != 0 {
		t.Fatalf("still deleted: %+v", d)
	}
}

// Narration through Blup Cloud (BLUP_CLOUD_LIVE_TOKEN: Pro or above, with
// the Worker's OpenRouter pointed at a fake one).
func TestBlupNarration(t *testing.T) {
	token := os.Getenv("BLUP_CLOUD_LIVE_TOKEN")
	if token == "" {
		t.Skip("BLUP_CLOUD_LIVE_TOKEN not set")
	}
	ctx := context.Background()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	(config.Credentials{BlupToken: token}).Save()
	root := t.TempDir()
	notes := filepath.Join(root, "notas")
	os.MkdirAll(filepath.Join(root, "site"), 0o755)
	os.MkdirAll(notes, 0o755)
	site := config.NewSite(filepath.Join(root, "site"))
	site.Title, site.Vault, site.Model, site.Lang = "Narração", notes, "mochi", "pt"
	text := strings.Repeat("Era uma vez um blog que falava. ", 100) // ~3.200 characters
	os.WriteFile(filepath.Join(notes, "post.md"), []byte("---\ntitle: Post\n---\n"+text+"\n"), 0o644)

	voice := blupPrefix + "pt-BR-Luana:MAI-Voice-2.1"
	if !NarratesThroughBlup("pt", voice) {
		t.Fatal("blup voice not recognized")
	}
	cost, err := EstimateNarration(ctx, site, "post.md", "")
	if err != nil || !cost.Enough || cost.Percent < 5 || cost.Percent > 20 {
		t.Fatalf("estimate = %+v, %v", cost, err)
	}
	n, err := Narrate(ctx, site, "post.md", voice, nil)
	if err != nil || n.Bytes == 0 {
		t.Fatalf("narrate = %+v, %v", n, err)
	}
	after, _ := EstimateNarration(ctx, site, "post.md", "")
	if after.Left >= cost.Left {
		t.Fatalf("the allowance did not go down: %d -> %d", cost.Left, after.Left)
	}
	// A text larger than what is left is refused before anything is spent.
	big := strings.Repeat("Texto enorme para narrar de uma vez só. ", 1000)
	os.WriteFile(filepath.Join(notes, "enorme.md"), []byte("---\ntitle: Enorme\n---\n"+big+"\n"), 0o644)
	if c, _ := EstimateNarration(ctx, site, "enorme.md", ""); c.Enough {
		t.Fatalf("a huge text fits? %+v", c)
	}
	if _, err := Narrate(ctx, site, "enorme.md", voice, nil); err == nil || !strings.Contains(err.Error(), "uso de IA") {
		t.Fatalf("huge text: %v", err)
	}
}
