package blupcloud

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriellsalesx/blup/internal/notesync"
)

// "Comparar do zero" (Syncer.Forget): o servidor bifurca o cofre, a sync
// de A para com ErrForked e continua parada; só quando a pessoa escolhe
// comparar do zero A esquece a base e a cadeia, compara como um aparelho
// novo e volta a sincronizar, sem apagar arquivo de nenhum lado: o que
// difere vira cópia de conflito, o que só um lado tem vai para o outro.
func TestForgetAfterForkSyncsAgainWithoutLosingFiles(t *testing.T) {
	ctx := context.Background()
	f := newFakeCloud(t)
	keys := testKeys(t)
	a, b := newDevice(t, f, keys), newDevice(t, f, keys)

	a.write(t, "diario.md", "comum")
	a.write(t, "velha.md", "velha")
	a.write(t, "fica.md", "fica")
	mustSync(t, a)
	mustSync(t, b)

	// Ramo de A (versão 2): edita o diário e apaga velha.md.
	a.write(t, "diario.md", "versão de A")
	if err := os.Remove(filepath.Join(a.dir, "velha.md")); err != nil {
		t.Fatal(err)
	}
	mustSync(t, a)
	// O servidor esconde a versão 2 de A; B salva outra versão 2 em que
	// edita o diário, cria outra.md e apaga fica.md.
	f.fork(1)
	b.write(t, "diario.md", "versão de B")
	b.write(t, "outra.md", "de B")
	if err := os.Remove(filepath.Join(b.dir, "fica.md")); err != nil {
		t.Fatal(err)
	}
	mustSync(t, b)

	// Parada por segurança, e continua parada a cada tentativa.
	for range 2 {
		_, err := a.s.Sync(ctx, nil)
		if !errors.Is(err, notesync.ErrForked) || !notesync.Stuck(err) {
			t.Fatalf("fork not refused: %v", err)
		}
	}
	if a.read("diario.md") != "versão de A" || a.read("outra.md") != "" || a.read("fica.md") != "fica" {
		t.Fatal("the refused sync changed local notes")
	}

	// Um manifesto antigo do ramo abandonado no cache local some junto.
	cache := a.s.Remote.(*Remote).Cache
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "2"), []byte("ramo de A"), 0o600); err != nil {
		t.Fatal(err)
	}

	// O botão: esquece e compara do zero.
	if err := a.s.Forget(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(a.s.StateFile); !os.IsNotExist(err) {
		t.Fatalf("state kept: %v", err)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("cache kept: %v", err)
	}
	rep, err := a.s.Sync(ctx, nil)
	if err != nil {
		t.Fatalf("sync after forget: %v", err)
	}
	if len(rep.Deleted)+len(rep.DeletedLocal) != 0 {
		t.Fatalf("deleted files: %+v", rep)
	}
	if len(rep.Conflicts) != 1 || rep.Conflicts[0] != "diario.md" || rep.ConflictCopies[0] == "" {
		t.Fatalf("conflicts = %v %v", rep.Conflicts, rep.ConflictCopies)
	}
	copyName := rep.ConflictCopies[0]
	want := map[string]string{
		"diario.md": "versão de A", copyName: "versão de B",
		"outra.md": "de B", "velha.md": "velha", "fica.md": "fica",
	}
	for p, v := range want {
		if got := a.read(p); got != v {
			t.Fatalf("A %s = %q, want %q", p, got, v)
		}
	}

	// A sync voltou a funcionar nos dois sentidos, com a cadeia nova.
	mustSync(t, b)
	for p, v := range want {
		if got := b.read(p); got != v {
			t.Fatalf("B %s = %q, want %q", p, got, v)
		}
	}
	if a.state(t)["chain"] == nil {
		t.Fatal("chain not saved again")
	}
	b.write(t, "depois.md", "depois")
	mustSync(t, b)
	mustSync(t, a)
	if a.read("depois.md") != "depois" {
		t.Fatal("A no longer receives")
	}
}

// O servidor volta para uma versão mais antiga do que a sincronizada
// (ErrRolledBack): comparar do zero também destrava, e o que o aparelho
// tem e a versão antiga não tem é enviado de novo, nunca apagado.
func TestForgetAfterRollbackKeepsNewerFiles(t *testing.T) {
	ctx := context.Background()
	f := newFakeCloud(t)
	keys := testKeys(t)
	a := newDevice(t, f, keys)
	a.write(t, "um.md", "1")
	mustSync(t, a)
	a.write(t, "dois.md", "2")
	mustSync(t, a)
	f.fork(1)

	_, err := a.s.Sync(ctx, nil)
	if !errors.Is(err, notesync.ErrRolledBack) || !notesync.Stuck(err) {
		t.Fatalf("rollback not refused: %v", err)
	}
	if err := a.s.Forget(); err != nil {
		t.Fatal(err)
	}
	rep, err := a.s.Sync(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Deleted)+len(rep.DeletedLocal) != 0 || len(rep.Uploaded) != 1 || rep.Uploaded[0] != "dois.md" {
		t.Fatalf("report = %+v", rep)
	}
	if a.read("um.md") != "1" || a.read("dois.md") != "2" {
		t.Fatal("lost a file")
	}
	mustSync(t, a)
}
