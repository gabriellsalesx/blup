package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
)

// miniCloud is just enough of Blup Cloud's sync API for the Studio's
// functions: one vault, versions and blocks in memory.
type miniCloud struct {
	mu        sync.Mutex
	head      int64
	manifests map[int64][]byte
	blocks    map[string][]byte
}

func (m *miniCloud) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := strings.TrimPrefix(r.URL.Path, "/api")
	js := func(v any) { json.NewEncoder(w).Encode(v) }
	switch {
	case p == "/me":
		js(map[string]any{"id": "u", "plan": map[string]any{"name": "max", "limits": map[string]any{"maxFileMB": 10}}})
	case p == "/sync/blocks/missing":
		var in struct{ IDs []string }
		json.NewDecoder(r.Body).Decode(&in)
		miss := []string{}
		for _, id := range in.IDs {
			if m.blocks[id] == nil {
				miss = append(miss, id)
			}
		}
		js(map[string]any{"missing": miss})
	case strings.HasPrefix(p, "/sync/blocks/"):
		id := strings.TrimPrefix(p, "/sync/blocks/")
		if r.Method == http.MethodPut {
			m.blocks[id], _ = io.ReadAll(r.Body)
			js(map[string]any{"ok": true})
			return
		}
		w.Write(m.blocks[id])
	case strings.HasSuffix(p, "/versions") && r.Method == http.MethodPost:
		r.ParseMultipartForm(1 << 20)
		f, _, _ := r.FormFile("manifest")
		b, _ := io.ReadAll(f)
		m.head++
		m.manifests[m.head] = b
		js(map[string]any{"n": m.head})
	case strings.Contains(p, "/versions/"):
		n, _ := strconv.ParseInt(p[strings.LastIndex(p, "/")+1:], 10, 64)
		if m.manifests[n] == nil {
			w.WriteHeader(404)
			return
		}
		w.Write(m.manifests[n])
	case strings.HasPrefix(p, "/sync/vaults/"):
		js(map[string]any{"head": m.head})
	default:
		w.WriteHeader(404)
	}
}

// put stores version head+1 with files (path -> content).
func (m *miniCloud) put(t *testing.T, k *cloudcrypt.Keys, vault string, files map[string]string) {
	t.Helper()
	type mf struct {
		SHA    string   `json:"sha"`
		Size   int64    `json:"size"`
		Mode   string   `json:"mode"`
		Blocks []string `json:"blocks"`
	}
	man := map[string]map[string]mf{"files": {}}
	m.mu.Lock()
	defer m.mu.Unlock()
	for p, c := range files {
		id := k.BlockID([]byte(c))
		m.blocks[id], _ = k.SealBlock(id, []byte(c))
		man["files"][p] = mf{SHA: github.BlobSHA([]byte(c)), Size: int64(len(c)), Mode: "100644", Blocks: []string{id}}
	}
	plain, _ := json.Marshal(man)
	m.head++
	m.manifests[m.head], _ = k.SealManifest(vault, m.head, plain)
}

// Pôr de volta uma versão antiga não escreve através de um link simbólico
// da pasta de notas para fora dela (~/.config, ~/.ssh...).
func TestRestoreNoteVersionStaysInNotes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks")
	}
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	cloud := &miniCloud{manifests: map[int64][]byte{}, blocks: map[string][]byte{}}
	srv := httptest.NewServer(cloud)
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)

	master, _, _, err := cloudcrypt.NewKeys("senha")
	if err != nil {
		t.Fatal(err)
	}
	keys, _ := cloudcrypt.Derive(master)
	creds := config.Credentials{BlupToken: "tok", NotesKey: base64.StdEncoding.EncodeToString(master), NotesAccount: "u"}
	if err := creds.Save(); err != nil {
		t.Fatal(err)
	}
	const vault = "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"
	cloud.put(t, keys, vault, map[string]string{"link/x.md": "conteúdo antigo", "ok.md": "versão 1"})

	site := settingsSite(t)
	site.Sync = config.NoteSync{Remote: RemoteBlup, Vault: vault}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(site.ContentDir(), "link")); err != nil {
		t.Skip(err)
	}
	cloud.put(t, keys, vault, map[string]string{"ok.md": "versão 2"})

	ctx := context.Background()
	if err := RestoreNoteVersion(ctx, site, "link/x.md", 1); err == nil {
		t.Fatal("restored through a symlink")
	}
	if _, err := os.Stat(filepath.Join(outside, "x.md")); err == nil {
		t.Fatal("wrote outside the notes")
	}
	if err := RestoreNoteVersion(ctx, site, "../fora.md", 1); err == nil {
		t.Fatal("restored outside the notes")
	}
	if err := RestoreNoteVersion(ctx, site, "ok.md", 1); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(site.ContentDir(), "ok.md")); string(b) != "versão 1" {
		t.Fatalf("restored %q", b)
	}

	// A vault id that is not one (a "../" in blup.toml) never names a folder.
	site.Sync.Vault = "../../fora"
	if _, err := blupRemote(ctx, site); !errors.Is(err, blupcloud.ErrBadVault) {
		t.Fatalf("bad vault id: %v", err)
	}
}
