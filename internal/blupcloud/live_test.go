//go:build cloudlive

// Runs against a real Blup Cloud (usually wrangler dev on this computer):
//
//	BLUP_CLOUD_URL=http://localhost:8787 BLUP_CLOUD_TOKEN=<Studio session> \
//	  go test -tags cloudlive ./internal/blupcloud
//
// The account must be new (no notes password yet).
package blupcloud

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/notesync"
)

func TestLiveSync(t *testing.T) {
	token := os.Getenv("BLUP_CLOUD_TOKEN")
	if token == "" {
		t.Skip("BLUP_CLOUD_TOKEN not set")
	}
	ctx := context.Background()
	c := New(token)
	me, err := c.Me(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("conta %s, plano %s", me.ID, me.Plan.Name)

	// Turning sync on: the notes password and the recovery key.
	if _, err := c.Key(ctx); !errors.Is(err, ErrNoKey) {
		t.Fatalf("Key before setup: %v", err)
	}
	master, recovery, w, err := cloudcrypt.NewKeys("senha das notas")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.SaveKey(ctx, w); err != nil {
		t.Fatal(err)
	}
	// Another computer: fetches the wrapping and opens it with the password.
	w2, err := c.Key(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := cloudcrypt.Unlock(w2, "senha das notas"); err != nil || !bytes.Equal(got, master) {
		t.Fatalf("Unlock on the other computer: %v", err)
	}
	if got, err := cloudcrypt.Recover(w2, recovery); err != nil || !bytes.Equal(got, master) {
		t.Fatalf("Recover: %v", err)
	}
	keys, _ := cloudcrypt.Derive(master)

	name, _ := keys.SealName("Meu blog")
	v, err := c.CreateVault(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreateVault(ctx, name); err != nil {
		t.Fatalf("a second vault (no limit): %v", err)
	}

	maxFile := int64(me.Plan.Limits.MaxFileMB) << 20
	computer := func() (string, *notesync.Syncer) {
		root := t.TempDir()
		dir := filepath.Join(root, "notas")
		os.MkdirAll(dir, 0o755)
		return dir, &notesync.Syncer{
			Remote:    &Remote{Client: New(token), Keys: keys, Vault: v.ID, MaxFile: maxFile},
			Dir:       dir,
			StateFile: filepath.Join(root, ".blup", "sync-state.json"),
		}
	}
	write := func(dir, name string, data []byte) {
		os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755)
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	read := func(dir, name string) string {
		b, _ := os.ReadFile(filepath.Join(dir, name))
		return string(b)
	}

	a, sa := computer()
	b, sb := computer()
	write(a, "index.md", []byte("# Início\n"))
	write(a, "Notas/receita de bolo.md", []byte("Farinha, ovos e leite.\n"))
	big := bytes.Repeat([]byte("0123456789abcdef"), 5<<20/16) // 5 MiB: two blocks
	write(a, "anexos/foto.bin", big)
	write(a, "anexos/grande.bin", make([]byte, maxFile+1)) // over the plan's limit

	rep, err := sa.Sync(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rep.Uploaded, []string{"Notas/receita de bolo.md", "anexos/foto.bin", "index.md"}) {
		t.Fatalf("uploaded = %v", rep.Uploaded)
	}
	if rep.SkipReasons["anexos/grande.bin"] != notesync.SkipTooLarge {
		t.Fatalf("over the limit: %v", rep.SkipReasons)
	}

	rep, err = sb.Sync(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Downloaded) != 3 || read(b, "Notas/receita de bolo.md") != "Farinha, ovos e leite.\n" || read(b, "anexos/foto.bin") != string(big) {
		t.Fatalf("second computer: %+v", rep)
	}

	// Both edit the same note: the second keeps a conflict copy.
	write(a, "index.md", []byte("# Início (A)\n"))
	write(b, "index.md", []byte("# Início (B)\n"))
	if _, err := sa.Sync(ctx, nil); err != nil {
		t.Fatal(err)
	}
	rep, err = sb.Sync(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.ConflictCopies) != 1 || read(b, rep.ConflictCopies[0]) != "# Início (A)\n" || read(b, "index.md") != "# Início (B)\n" {
		t.Fatalf("conflict: %+v", rep)
	}

	// Deleted on one computer, moved to the trash on the other.
	os.Remove(filepath.Join(b, "Notas/receita de bolo.md"))
	if _, err := sb.Sync(ctx, nil); err != nil {
		t.Fatal(err)
	}
	rep, err = sa.Sync(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rep.DeletedLocal, []string{"Notas/receita de bolo.md"}) || read(rep.Trash, "Notas/receita de bolo.md") != "Farinha, ovos e leite.\n" {
		t.Fatalf("deletion: %+v", rep)
	}
	if read(a, "index.md") != "# Início (B)\n" {
		t.Fatalf("A did not receive B's version: %q", read(a, "index.md"))
	}

	me, _ = c.Me(ctx)
	t.Logf("uso: %d bytes, %d cofres", me.Usage.StorageBytes, me.Usage.Vaults)
}

// The account's live room (the account must be on a live plan): a new
// version saved by one computer reaches the others at once.
func TestLiveRoom(t *testing.T) {
	token := os.Getenv("BLUP_CLOUD_LIVE_TOKEN")
	if token == "" {
		t.Skip("BLUP_CLOUD_LIVE_TOKEN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := New(token)
	master, _, w, _ := cloudcrypt.NewKeys("senha das notas")
	if err := c.SaveKey(ctx, w); err != nil {
		t.Fatal(err)
	}
	keys, _ := cloudcrypt.Derive(master)
	name, _ := keys.SealName("Ao vivo")
	v, err := c.CreateVault(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	got := make(chan Moved, 1)
	go c.Listen(ctx, func(m Moved) { got <- m })
	time.Sleep(time.Second) // joined

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "nota.md"), []byte("oi\n"), 0o644)
	sy := &notesync.Syncer{Remote: &Remote{Client: New(token), Keys: keys, Vault: v.ID}, Dir: dir, StateFile: filepath.Join(t.TempDir(), "state.json")}
	if _, err := sy.Sync(ctx, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-got:
		if m.Vault != v.ID || m.N != 1 {
			t.Fatalf("moved = %+v", m)
		}
	case <-ctx.Done():
		t.Fatal("no message from the live room")
	}
}
