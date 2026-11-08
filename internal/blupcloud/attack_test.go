package blupcloud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/notesync"
)

// fakeCloud is an in-memory Blup Cloud with the sync API, that can turn
// malicious: show an old version (rollback), make up vault ids, or answer
// huge bodies.
type fakeCloud struct {
	mu        sync.Mutex
	srv       *httptest.Server
	head      int64
	manifests map[int64][]byte
	blocks    map[string][]byte
	// evil knobs
	showHead int64  // > 0: answer this head instead of the real one
	vaultID  string // id answered by CreateVault and Vaults
	seen     []string
	// race runs (unlocked) when a version is sent, before it is saved:
	// another device saving first, for the 409 flow.
	race func()
}

func newFakeCloud(t *testing.T) *fakeCloud {
	f := &fakeCloud{manifests: map[int64][]byte{}, blocks: map[string][]byte{}, vaultID: "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeCloud) client() *Client {
	return &Client{BaseURL: f.srv.URL, Token: "tok", HTTP: f.srv.Client()}
}

func (f *fakeCloud) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := strings.TrimPrefix(r.URL.Path, "/api/sync")
	f.seen = append(f.seen, r.Method+" "+p)
	js := func(v any) { w.Header().Set("Content-Type", "application/json"); json.NewEncoder(w).Encode(v) }
	switch {
	case r.Method == http.MethodGet && p == "/vaults":
		js(map[string]any{"vaults": []map[string]any{{"id": f.vaultID, "head": f.head}, {"id": "../../escape", "head": 1}}})
	case r.Method == http.MethodPost && p == "/vaults":
		js(map[string]any{"id": f.vaultID, "head": 0})
	case r.Method == http.MethodPost && p == "/blocks/missing":
		var in struct{ IDs []string }
		json.NewDecoder(r.Body).Decode(&in)
		var miss []string
		for _, id := range in.IDs {
			if f.blocks[id] == nil {
				miss = append(miss, id)
			}
		}
		js(map[string]any{"missing": miss})
	case strings.HasPrefix(p, "/blocks/"):
		id := strings.TrimPrefix(p, "/blocks/")
		if r.Method == http.MethodPut {
			b, _ := io.ReadAll(r.Body)
			f.blocks[id] = b
			js(map[string]any{"ok": true})
			return
		}
		if b := f.blocks[id]; b != nil {
			w.Write(b)
			return
		}
		w.WriteHeader(404)
	case strings.HasSuffix(p, "/versions") && r.Method == http.MethodPost:
		if race := f.race; race != nil {
			f.race = nil
			f.mu.Unlock()
			race()
			f.mu.Lock()
		}
		r.ParseMultipartForm(64 << 20)
		parent, _ := strconv.ParseInt(r.FormValue("parent"), 10, 64)
		if parent != f.head {
			w.WriteHeader(409)
			js(map[string]any{"code": "MOVED"})
			return
		}
		file, _, _ := r.FormFile("manifest")
		b, _ := io.ReadAll(file)
		f.head++
		f.manifests[f.head] = b
		js(map[string]any{"n": f.head})
	case strings.Contains(p, "/versions/"):
		n, _ := strconv.ParseInt(p[strings.LastIndex(p, "/")+1:], 10, 64)
		if b := f.manifests[n]; b != nil {
			w.Write(b)
			return
		}
		w.WriteHeader(404)
	case strings.HasPrefix(p, "/vaults/"):
		h := f.head
		if f.showHead > 0 {
			h = f.showHead
		}
		js(map[string]any{"id": strings.TrimPrefix(p, "/vaults/"), "head": h})
	default:
		w.WriteHeader(404)
	}
}

type device struct {
	dir string
	s   *notesync.Syncer
}

func newDevice(t *testing.T, f *fakeCloud, keys *cloudcrypt.Keys) *device {
	root := t.TempDir()
	dir := filepath.Join(root, "notas")
	os.MkdirAll(dir, 0o755)
	return &device{dir: dir, s: &notesync.Syncer{
		Remote:    &Remote{Client: f.client(), Keys: keys, Vault: f.vaultID, Cache: filepath.Join(root, "cache", f.vaultID)},
		Dir:       dir,
		StateFile: filepath.Join(root, ".blup", "sync-state.json"),
	}}
}

func (d *device) write(t *testing.T, name, text string) {
	p := filepath.Join(d.dir, filepath.FromSlash(name))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (d *device) read(name string) string {
	b, _ := os.ReadFile(filepath.Join(d.dir, filepath.FromSlash(name)))
	return string(b)
}

func testKeys(t *testing.T) *cloudcrypt.Keys {
	master, _, _, err := cloudcrypt.NewKeys("senha")
	if err != nil {
		t.Fatal(err)
	}
	k, _ := cloudcrypt.Derive(master)
	return k
}

// Servidor malicioso mostra uma versão antiga (ainda selada de verdade):
// sem a proteção, o aparelho voltaria a nota editada ao texto antigo e
// mandaria para a lixeira a nota criada depois. Agora a sync é recusada e
// nada muda no computador.
func TestAttackRollbackRefused(t *testing.T) {
	ctx := context.Background()
	f := newFakeCloud(t)
	keys := testKeys(t)
	a, b := newDevice(t, f, keys), newDevice(t, f, keys)

	a.write(t, "diario.md", "v1")
	if _, err := a.s.Sync(ctx, nil); err != nil {
		t.Fatal(err)
	}
	a.write(t, "diario.md", "v2 com segredo novo")
	a.write(t, "nova.md", "nota nova")
	if _, err := a.s.Sync(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := b.s.Sync(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if b.read("nova.md") != "nota nova" {
		t.Fatal("second device did not receive")
	}

	for _, old := range []int64{1, -1} { // version 1, and "empty" (head 0)
		f.mu.Lock()
		f.showHead = old
		if old < 0 {
			f.showHead = 0
			f.head, f.manifests = 0, map[int64][]byte{}
		}
		f.mu.Unlock()
		_, err := b.s.Sync(ctx, nil)
		if !errors.Is(err, notesync.ErrRolledBack) {
			t.Fatalf("rollback to %d: err = %v", old, err)
		}
		if b.read("diario.md") != "v2 com segredo novo" || b.read("nova.md") != "nota nova" {
			t.Fatalf("rollback to %d changed local notes", old)
		}
	}
}

// Um manifesto de outro cofre ou de outra versão, servido no lugar do
// certo, não abre: o servidor não troca o conteúdo das notas.
func TestAttackManifestSwapRefused(t *testing.T) {
	ctx := context.Background()
	f := newFakeCloud(t)
	keys := testKeys(t)
	a, b := newDevice(t, f, keys), newDevice(t, f, keys)
	a.write(t, "x.md", "um")
	a.s.Sync(ctx, nil)
	a.write(t, "x.md", "dois")
	a.s.Sync(ctx, nil)
	f.mu.Lock()
	f.manifests[2] = f.manifests[1] // version 1's manifest served as version 2
	f.mu.Unlock()
	if _, err := b.s.Sync(ctx, nil); !errors.Is(err, cloudcrypt.ErrCorrupt) {
		t.Fatalf("swapped manifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(b.dir, "x.md")); err == nil {
		t.Fatal("a file was written from a swapped manifest")
	}
	// A block swapped for another is refused too.
	f.mu.Lock()
	f.manifests[2] = nil
	f.head = 1
	var ids []string
	for id := range f.blocks {
		ids = append(ids, id)
	}
	if len(ids) == 2 {
		f.blocks[ids[0]], f.blocks[ids[1]] = f.blocks[ids[1]], f.blocks[ids[0]]
	}
	f.mu.Unlock()
	rep, err := b.s.Sync(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if b.read("x.md") != "" || rep.SkipReasons["x.md"] != notesync.SkipWriteFailed {
		t.Fatalf("swapped block accepted: %q %+v", b.read("x.md"), rep)
	}
}

// Ids de cofre inventados pelo servidor ("../") nunca viram pastas: o
// cache do histórico ficaria fora da pasta do site.
func TestAttackVaultIDTraversal(t *testing.T) {
	ctx := context.Background()
	f := newFakeCloud(t)
	f.vaultID = "../../../../tmp/blup-escape"
	c := f.client()
	if _, err := c.CreateVault(ctx, []byte("x")); !errors.Is(err, ErrBadVault) {
		t.Fatalf("CreateVault: %v", err)
	}
	vs, err := c.Vaults(ctx)
	if err != nil || len(vs) != 0 {
		t.Fatalf("Vaults kept a bad id: %+v %v", vs, err)
	}
	r := &Remote{Client: c, Keys: testKeys(t), Vault: f.vaultID, Cache: filepath.Join(t.TempDir(), f.vaultID)}
	if _, err := r.Head(ctx, true); !errors.Is(err, ErrBadVault) {
		t.Fatalf("Head: %v", err)
	}
	if _, err := r.manifestAt(ctx, 1); !errors.Is(err, ErrBadVault) {
		t.Fatalf("manifestAt: %v", err)
	}
	for _, ok := range []string{"0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b", "v1"} {
		if !ValidVaultID(ok) {
			t.Fatalf("%q refused", ok)
		}
	}
	for _, bad := range []string{"", "..", "a/b", `a\b`, "a b", strings.Repeat("a", 65)} {
		if ValidVaultID(bad) {
			t.Fatalf("%q accepted", bad)
		}
	}
}

// O cache do histórico só guarda manifestos que abriram: o que o servidor
// inventa não fica no disco.
func TestAttackHistoryCacheOnlyAuthentic(t *testing.T) {
	ctx := context.Background()
	f := newFakeCloud(t)
	keys := testKeys(t)
	a := newDevice(t, f, keys)
	a.write(t, "x.md", "um")
	a.s.Sync(ctx, nil)
	r := a.s.Remote.(*Remote)
	f.mu.Lock()
	f.manifests[1] = []byte("lixo do servidor")
	f.mu.Unlock()
	if _, err := r.manifestAt(ctx, 1); err == nil {
		t.Fatal("forged manifest opened")
	}
	if _, err := os.Stat(filepath.Join(r.Cache, "1")); err == nil {
		t.Fatal("forged manifest cached")
	}
}

// Respostas gigantes do servidor não enchem a memória.
func TestAttackHugeAnswer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chunk := make([]byte, 1<<20)
		for range maxBody>>20 + 2 {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	if _, err := c.Manifest(context.Background(), "v1", 1); err == nil {
		t.Fatal("huge answer accepted")
	}
	if _, err := c.Block(context.Background(), "../../x"); !errors.Is(err, cloudcrypt.ErrCorrupt) {
		t.Fatalf("bad block id: %v", err)
	}
}
