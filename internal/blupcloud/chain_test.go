package blupcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/notesync"
)

// Ataques e compatibilidade da cadeia de manifestos (chain.go).

// fork tira do servidor as versões depois de n, como um servidor que
// esconde de um aparelho o que outro salvou, e devolve o que tirou.
func (f *fakeCloud) fork(n int64) map[int64][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	hidden := map[int64][]byte{}
	for v, b := range f.manifests {
		if v > n {
			hidden[v] = b
			delete(f.manifests, v)
		}
	}
	f.head = n
	return hidden
}

func (d *device) state(t *testing.T) map[string]any {
	t.Helper()
	b, err := os.ReadFile(d.s.StateFile)
	if err != nil {
		t.Fatal(err)
	}
	var st map[string]any
	json.Unmarshal(b, &st)
	return st
}

// O servidor bifurca o cofre: A salva a versão 2, o servidor a esconde de
// B, B salva outra versão 2. Mostrar a A a versão de B (mesmo número,
// conteúdo diferente) ou uma 3 que descende dela é recusado, e nada muda
// em A.
func TestAttackForkRefused(t *testing.T) {
	ctx := context.Background()
	f := newFakeCloud(t)
	keys := testKeys(t)
	a, b := newDevice(t, f, keys), newDevice(t, f, keys)

	a.write(t, "diario.md", "comum")
	mustSync(t, a)
	mustSync(t, b)
	if a.state(t)["chain"] == nil {
		t.Fatal("chain token not saved")
	}

	a.write(t, "diario.md", "versão de A")
	mustSync(t, a) // versão 2 de A
	f.fork(1)
	b.write(t, "outra.md", "versão de B")
	mustSync(t, b) // outra versão 2, só B a vê

	for round := range 2 {
		_, err := a.s.Sync(ctx, nil)
		if !errors.Is(err, notesync.ErrForked) {
			t.Fatalf("round %d: fork not detected: %v", round, err)
		}
		if a.read("diario.md") != "versão de A" || a.read("outra.md") != "" {
			t.Fatalf("round %d: the fork changed local notes", round)
		}
		b.write(t, "mais.md", "versão 3 de B")
		mustSync(t, b) // a bifurcação cresce: 3 descende do 2 de B
	}
}

// Fluxo legítimo de conflito: A e B estão na mesma versão, A salva
// primeiro, B recebe 409, compara de novo em cima da versão de A e salva a
// seguinte. A cadeia continua e os dois seguem sincronizando.
func TestChainSurvivesConcurrentCommits(t *testing.T) {
	f := newFakeCloud(t)
	keys := testKeys(t)
	a, b := newDevice(t, f, keys), newDevice(t, f, keys)
	a.write(t, "base.md", "base")
	mustSync(t, a)
	mustSync(t, b)

	a.write(t, "a.md", "de A")
	b.write(t, "b.md", "de B")
	f.mu.Lock()
	f.race = func() { mustSync(t, a) }
	f.mu.Unlock()
	mustSync(t, b) // 409 no meio, refeito em cima da versão de A
	if f.head != 3 {
		t.Fatalf("head = %d", f.head)
	}
	mustSync(t, a)
	if a.read("b.md") != "de B" || b.read("a.md") != "de A" {
		t.Fatal("notes not exchanged")
	}
}

// Aparelho que ficou para trás mais de chainBack versões: a cadeia é
// seguida pelas versões que o servidor ainda guarda. Uma bifurcação longa
// é detectada; com o histórico apagado, não há como seguir (limite).
func TestChainAcrossLongGaps(t *testing.T) {
	ctx := context.Background()
	f := newFakeCloud(t)
	keys := testKeys(t)
	a, b := newDevice(t, f, keys), newDevice(t, f, keys)
	a.write(t, "n.md", "0")
	mustSync(t, a)
	mustSync(t, b)
	gap := chainBack*2 + 5
	for i := range gap {
		b.write(t, "n.md", fmt.Sprint(i+1))
		mustSync(t, b)
	}
	mustSync(t, a) // segue a cadeia pelo histórico: aceita
	if a.read("n.md") != fmt.Sprint(gap) {
		t.Fatal("did not receive")
	}

	// Bifurcação longa com histórico guardado: detectada.
	start := f.head
	a.write(t, "n.md", "só A")
	mustSync(t, a)
	f.fork(start)
	for i := range gap {
		b.write(t, "n.md", fmt.Sprint("B", i))
		mustSync(t, b)
	}
	if _, err := a.s.Sync(ctx, nil); !errors.Is(err, notesync.ErrForked) {
		t.Fatalf("long fork with history: %v", err)
	}

	// Limite conhecido: sem as versões do meio (plano sem histórico), a
	// cadeia não pode ser seguida e a mais nova é aceita.
	f.mu.Lock()
	for v := range f.manifests {
		if v != f.head {
			delete(f.manifests, v)
		}
	}
	f.mu.Unlock()
	if err := os.RemoveAll(a.s.Remote.(*Remote).Cache); err != nil {
		t.Fatal(err)
	}
	if _, err := a.s.Sync(ctx, nil); err != nil {
		t.Fatalf("pruned history: %v", err)
	}
}

// O servidor não consegue forjar a cadeia: um "prev" que não aponta para a
// versão anterior é recusado ao abrir o manifesto.
func TestAttackChainLinkMustNameParent(t *testing.T) {
	ctx := context.Background()
	f := newFakeCloud(t)
	keys := testKeys(t)
	a := newDevice(t, f, keys)
	a.write(t, "n.md", "1")
	mustSync(t, a)
	r := a.s.Remote.(*Remote)
	bad, _ := json.Marshal(manifest{Files: map[string]manifestFile{}, Prev: &chainLink{N: 5, MACs: [][]byte{make([]byte, 32)}}})
	sealed, _ := keys.SealManifest(f.vaultID, 2, bad)
	f.mu.Lock()
	f.head, f.manifests[2] = 2, sealed
	f.mu.Unlock()
	if _, err := r.Head(ctx, false); !errors.Is(err, ErrBadAnswer) {
		t.Fatalf("bad prev: %v", err)
	}
}

// Compatibilidade: um cofre com manifesto do formato antigo (sem "prev",
// testdata/format1.json, gerado antes da cadeia) abre; a primeira versão
// nova começa a cadeia nele; um estado salvo antes (sem "chain") segue
// valendo; e um Blup antigo que salve sem "prev" no meio não trava ninguém.
func TestCompatLegacyManifestsAndState(t *testing.T) {
	ctx := context.Background()
	raw, err := os.ReadFile("testdata/format1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Master   []byte `json:"master"`
		Vault    string `json:"vault"`
		N        int64  `json:"manifest_n"`
		Manifest []byte `json:"manifest"`
	}
	json.Unmarshal(raw, &fx)
	keys, _ := cloudcrypt.Derive(fx.Master)
	f := newFakeCloud(t)
	f.vaultID = fx.Vault
	empty, _ := keys.SealBlock(keys.BlockID(nil), nil)
	f.blocks[keys.BlockID(nil)] = empty
	f.head, f.manifests[fx.N] = fx.N, fx.Manifest

	a, b := newDevice(t, f, keys), newDevice(t, f, keys)
	mustSync(t, a)
	if _, err := os.Stat(filepath.Join(a.dir, "antiga.md")); err != nil {
		t.Fatal("legacy manifest not read")
	}
	a.write(t, "nova.md", "nova")
	mustSync(t, a)
	r := a.s.Remote.(*Remote)
	m, err := r.manifestAt(ctx, fx.N+1)
	if err != nil || m.Prev == nil || m.Prev.N != fx.N || len(m.Prev.MACs) != 1 {
		t.Fatalf("chain did not start on the legacy manifest: %+v %v", m.Prev, err)
	}

	// Estado salvo por uma versão antiga do Blup: sem "chain".
	mustSync(t, b)
	st := b.state(t)
	delete(st, "chain")
	old, _ := json.Marshal(st)
	os.WriteFile(b.s.StateFile, old, 0o644)
	b.write(t, "b.md", "de B")
	mustSync(t, b)
	if b.state(t)["chain"] == nil {
		t.Fatal("chain not saved after an old state")
	}

	// Um Blup antigo salva a versão seguinte sem "prev".
	f.mu.Lock()
	plain, _ := json.Marshal(struct {
		Files map[string]manifestFile `json:"files"`
	}{Files: m.Files})
	sealed, _ := keys.SealManifest(fx.Vault, f.head+1, plain)
	f.head++
	f.manifests[f.head] = sealed
	f.mu.Unlock()
	mustSync(t, a)
	mustSync(t, b)
}

// Um cliente antigo lê o manifesto novo: parseManifest (igual ao de antes
// para "files") ignora "prev" e qualquer outra chave desconhecida.
func TestCompatOldReaderIgnoresPrev(t *testing.T) {
	m, err := parseManifest([]byte(`{"files":{},"prev":{"n":1,"macs":[]},"futuro":true}`))
	if err != nil || len(m.Files) != 0 {
		t.Fatalf("%v", err)
	}
	var old struct {
		Files map[string]manifestFile `json:"files"`
	}
	if err := json.Unmarshal([]byte(`{"files":{},"prev":{"n":1,"macs":[]}}`), &old); err != nil || old.Files == nil {
		t.Fatalf("old reader: %v", err)
	}
}

func mustSync(t *testing.T, d *device) {
	t.Helper()
	if _, err := d.s.Sync(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}
