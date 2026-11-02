package blupcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"github.com/gabriellsalesx/blup/internal/notesync"
)

// A manifest that opens (sealed with the right keys) but makes no sense
// is refused before anything changes here, and is never read as an empty
// vault (which would send the notes of this computer to the trash).
func TestBadManifestRefused(t *testing.T) {
	ctx := context.Background()
	f := newFakeCloud(t)
	keys := testKeys(t)
	a, b := newDevice(t, f, keys), newDevice(t, f, keys)
	a.write(t, "nota.md", "texto")
	if _, err := a.s.Sync(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := b.s.Sync(ctx, nil); err != nil || b.read("nota.md") != "texto" {
		t.Fatalf("second device: %v", err)
	}
	sha := github.BlobSHA([]byte("texto"))
	block := keys.BlockID([]byte("texto"))
	file := func(sha string, size int64, mode string, blocks ...string) string {
		bs, _ := json.Marshal(blocks)
		return fmt.Sprintf(`{"sha":%q,"size":%d,"mode":%q,"blocks":%s}`, sha, size, mode, bs)
	}
	bad := map[string]string{
		"null":           `null`,
		"empty object":   `{}`,
		"files null":     `{"files":null}`,
		"array":          `[]`,
		"files array":    `{"files":[]}`,
		"short sha":      `{"files":{"nota.md":` + file("abc", 5, "100644", block) + `}}`,
		"negative size":  `{"files":{"nota.md":` + file(sha, -1, "100644", block) + `}}`,
		"huge size":      `{"files":{"nota.md":` + file(sha, 1<<40, "100644", block) + `}}`,
		"bad mode":       `{"files":{"nota.md":` + file(sha, 5, "120000", block) + `}}`,
		"bad block id":   `{"files":{"nota.md":` + file(sha, 5, "100644", "../x") + `}}`,
		"over the cap":   `{"files":{"nota.md":` + file(sha, notesync.MaxFileSize+1, "100644", manyBlocks(block, notesync.MaxFileSize+1)...) + `}}`,
		"too few blocks": `{"files":{"nota.md":` + file(sha, cloudcrypt.BlockSize+1, "100644", block) + `}}`,
		"no blocks":      `{"files":{"nota.md":` + file(sha, 5, "100644") + `}}`,
		"path dotdot":    `{"files":{"../fora.md":` + file(sha, 5, "100644", block) + `}}`,
		"path absolute":  `{"files":{"/etc/x":` + file(sha, 5, "100644", block) + `}}`,
	}
	for name, plain := range bad {
		f.mu.Lock()
		sealed, err := keys.SealManifest(f.vaultID, f.head, []byte(plain))
		if err != nil {
			t.Fatal(err)
		}
		f.manifests[f.head] = sealed
		f.mu.Unlock()
		if _, err := b.s.Sync(ctx, nil); err == nil {
			t.Errorf("%s: sync accepted it", name)
		}
		if b.read("nota.md") != "texto" {
			t.Fatalf("%s: the note here changed", name)
		}
	}
}

// Blocks that hold more than the manifest announced stop the download.
func TestDownloadLargerThanAnnounced(t *testing.T) {
	f := newFakeCloud(t)
	keys := testKeys(t)
	data := bytes.Repeat([]byte("x"), 1000)
	id := keys.BlockID(data)
	sealed, _ := keys.SealBlock(id, data)
	f.blocks[id] = sealed
	r := &Remote{Client: f.client(), Keys: keys, Vault: f.vaultID}
	ctx := context.Background()
	if _, err := r.download(ctx, manifestFile{SHA: github.BlobSHA(data), Size: 10, Blocks: []string{id}}); !errors.Is(err, cloudcrypt.ErrCorrupt) {
		t.Fatalf("bigger than announced: %v", err)
	}
	if _, err := r.download(ctx, manifestFile{SHA: github.BlobSHA(data), Size: 2000, Blocks: []string{id}}); !errors.Is(err, cloudcrypt.ErrCorrupt) {
		t.Fatalf("smaller than announced: %v", err)
	}
	if got, err := r.download(ctx, manifestFile{SHA: github.BlobSHA(data), Size: 1000, Blocks: []string{id}}); err != nil || len(got) != 1000 {
		t.Fatalf("right size: %v", err)
	}
}

func answering(t *testing.T, fn func(w http.ResponseWriter, r *http.Request)) *Client {
	srv := httptest.NewServer(http.HandlerFunc(fn))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}
}

// null, {} and fields that cannot be are errors, not zero values.
func TestAnswersChecked(t *testing.T) {
	ctx := context.Background()
	vault := "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"
	for _, body := range []string{`null`, ``, `{}`, `{"versions":null}`, `{"versions":[{"n":0,"size":1,"createdAt":"2026-01-01T00:00:00Z"}]}`,
		`{"versions":[{"n":5,"size":-1,"createdAt":"2026-01-01T00:00:00Z"}]}`, `{"versions":[{"n":5,"size":1}]}`,
		`{"versions":[{"n":5,"size":1,"createdAt":"2026-01-01T00:00:00Z"},{"n":5,"size":1,"createdAt":"2026-01-01T00:00:00Z"}]}`} {
		c := answering(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
		if _, err := c.Versions(ctx, vault, 0); err == nil {
			t.Errorf("versions %q accepted", body)
		}
	}
	for _, body := range []string{`null`, `{}`, `{"head":-1}`, `{"head":null}`} {
		c := answering(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
		if _, err := c.VaultHead(ctx, vault); err == nil {
			t.Errorf("head %q accepted", body)
		}
	}
	for _, body := range []string{`null`, `{}`, `{"id":"u","plan":{"name":"pro","limits":{"storageMB":-5}}}`, `{"id":"u","plan":{"name":"pro"},"usage":{"storageBytes":-1}}`} {
		c := answering(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
		if _, err := c.Me(ctx); err == nil {
			t.Errorf("me %q accepted", body)
		}
	}
	for _, body := range []string{`null`, `{"monthMicros":-1}`, `{"percent":-3}`} {
		c := answering(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
		if _, err := c.AIUsage(ctx); err == nil {
			t.Errorf("ai %q accepted", body)
		}
	}
	for _, body := range []string{`null`, `{}`, `{"n":7}`, `{"n":-1}`} {
		c := answering(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
		if _, err := c.Commit(ctx, vault, 3, nil, []byte("m")); err == nil {
			t.Errorf("commit %q accepted", body)
		}
	}
	c := answering(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"n":4}`)) })
	if n, err := c.Commit(ctx, vault, 3, nil, []byte("m")); err != nil || n != 4 {
		t.Fatalf("good commit: %d %v", n, err)
	}
	c = answering(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"u","plan":{"name":"free","limits":{"sites":0,"devices":-1}},"usage":{}}`))
	})
	if _, err := c.Me(ctx); err != nil {
		t.Fatalf("good account: %v", err)
	}
}

// A cloud that answers the same full page whatever "before" says does
// not keep the history reading forever.
func TestRepeatedVersionsPageEnds(t *testing.T) {
	var calls atomic.Int32
	c := answering(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) > 5 {
			t.Error("history kept asking")
			w.WriteHeader(500)
			return
		}
		var vs []map[string]any
		for n := 100; n > 50; n-- {
			vs = append(vs, map[string]any{"n": n, "size": 1, "createdAt": time.Now().UTC().Format(time.RFC3339)})
		}
		json.NewEncoder(w).Encode(map[string]any{"versions": vs})
	})
	vault := "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"
	page, err := c.Versions(context.Background(), vault, 0)
	if err != nil || len(page) != 50 {
		t.Fatalf("first page: %d %v", len(page), err)
	}
	// The next page must start below the last version seen (51): the
	// same page again is refused, so the history loop cannot spin.
	if _, err := c.Versions(context.Background(), vault, page[len(page)-1].N); err == nil {
		t.Fatal("repeated page accepted")
	}
}

// The live room passes on only what looks like a vault's change.
func TestLiveMessagesChecked(t *testing.T) {
	vault := "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for _, m := range []string{`{"vault":"../../x","n":3}`, `{"vault":"` + vault + `","n":0}`, `{"vault":"` + vault + `","n":-2}`, `null`, `{"vault":"` + vault + `","n":7}`} {
			conn.Write(r.Context(), websocket.MessageText, []byte(m))
		}
		conn.Close(websocket.StatusNormalClosure, "")
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}
	var got []Moved
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Listen(ctx, func(m Moved) { got = append(got, m) })
	if len(got) != 1 || got[0].Vault != vault || got[0].N != 7 {
		t.Fatalf("passed on: %+v", got)
	}
}

// manyBlocks repeats id as many times as a file of size needs.
func manyBlocks(id string, size int64) []string {
	out := make([]string, blocksFor(size))
	for i := range out {
		out[i] = id
	}
	return out
}

// A manifest file above the cap is refused even with the right number of
// blocks, and a download never goes past the bytes announced.
func TestManifestFileCap(t *testing.T) {
	id := strings.Repeat("a", 64)
	sha := strings.Repeat("b", 40)
	if err := (manifestFile{SHA: sha, Size: notesync.MaxFileSize + 1, Blocks: manyBlocks(id, notesync.MaxFileSize+1)}).check(); err == nil {
		t.Fatal("file over the cap accepted")
	}
	if err := (manifestFile{SHA: sha, Size: notesync.MaxFileSize, Blocks: manyBlocks(id, notesync.MaxFileSize)}).check(); err != nil {
		t.Fatalf("file at the cap refused: %v", err)
	}
	// Two blocks of 1000 bytes for a file announced with 1500: the second
	// one is cut before it is kept.
	f := newFakeCloud(t)
	keys := testKeys(t)
	a, b := bytes.Repeat([]byte("x"), 1000), bytes.Repeat([]byte("y"), 1000)
	for _, p := range [][]byte{a, b} {
		bid := keys.BlockID(p)
		sealed, _ := keys.SealBlock(bid, p)
		f.blocks[bid] = sealed
	}
	r := &Remote{Client: f.client(), Keys: keys, Vault: f.vaultID}
	mf := manifestFile{SHA: github.BlobSHA(append(append([]byte{}, a...), b...)), Size: 1500, Blocks: []string{keys.BlockID(a), keys.BlockID(b)}}
	if _, err := r.download(context.Background(), mf); err == nil {
		t.Fatal("blocks over the announced size accepted")
	}
}
