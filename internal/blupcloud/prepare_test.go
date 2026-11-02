package blupcloud

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// quotaCloud is fakeCloud with a storage limit, as the cloud does it after
// a downgrade: a new block above the limit is refused (413 QUOTA), a
// version with no new block always goes, and a prepared version goes when
// what it adds is no more than what it frees. No history here: what a
// version no longer uses is freed when it is saved.
type quotaCloud struct {
	*fakeCloud
	qmu       sync.Mutex
	limit     int64
	inUse     map[string]bool // the blocks of the current version
	tickets   map[string]*ticket
	prepares  int
	cancelled int
	// expireOnce: the next ticket expires before its commit.
	expireOnce bool
	expired    string
}

type ticket struct {
	parent  int64
	blocks  map[string]bool
	missing map[string]bool
	sealed  []byte
}

func newQuotaCloud(t *testing.T) *quotaCloud {
	q := &quotaCloud{fakeCloud: newFakeCloud(t), limit: 1 << 40, inUse: map[string]bool{}, tickets: map[string]*ticket{}}
	q.srv.Config.Handler = http.HandlerFunc(q.serve)
	return q
}

func (q *quotaCloud) used() int64 {
	var n int64
	for _, b := range q.blocks {
		n += int64(len(b))
	}
	return n
}

// save makes blocks the current version's, freeing the others.
func (q *quotaCloud) save(blocks map[string]bool, sealed []byte) int64 {
	q.head++
	q.manifests[q.head] = sealed
	for id := range q.blocks {
		if !blocks[id] {
			delete(q.blocks, id)
		}
	}
	q.inUse = blocks
	return q.head
}

func (q *quotaCloud) serve(w http.ResponseWriter, r *http.Request) {
	q.qmu.Lock()
	defer q.qmu.Unlock()
	q.mu.Lock()
	p := strings.TrimPrefix(r.URL.Path, "/api/sync")
	js := func(code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(v)
	}
	fail := func(code int, c string) { js(code, map[string]string{"code": c}) }
	switch {
	case r.Method == http.MethodPut && strings.HasPrefix(p, "/blocks/"):
		id := strings.TrimPrefix(p, "/blocks/")
		b, _ := io.ReadAll(r.Body)
		if t := r.Header.Get("x-blup-upload-ticket"); t != "" {
			tk := q.tickets[t]
			if tk == nil || !tk.missing[id] {
				fail(404, "NOT_FOUND")
				break
			}
			q.blocks[id] = b
			js(200, map[string]bool{"ok": true})
			break
		}
		if q.blocks[id] == nil && q.used()+int64(len(b)) > q.limit {
			fail(413, "QUOTA")
			break
		}
		q.blocks[id] = b
		js(200, map[string]bool{"ok": true})
	case r.Method == http.MethodPost && strings.HasSuffix(p, "/versions"):
		r.ParseMultipartForm(64 << 20)
		parent, _ := strconv.ParseInt(r.FormValue("parent"), 10, 64)
		if parent != q.head {
			fail(409, "MOVED")
			break
		}
		blocks := map[string]bool{}
		for _, id := range strings.Split(r.FormValue("blocks"), "\n") {
			if q.blocks[id] == nil {
				fail(400, "MISSING_BLOCKS")
				q.mu.Unlock()
				return
			}
			blocks[id] = true
		}
		file, _, _ := r.FormFile("manifest")
		b, _ := io.ReadAll(file)
		js(200, map[string]int64{"n": q.save(blocks, b)})
	case r.Method == http.MethodPost && strings.HasSuffix(p, "/prepare"):
		q.prepares++
		r.ParseMultipartForm(64 << 20)
		parent, _ := strconv.ParseInt(r.FormValue("parent"), 10, 64)
		if parent != q.head {
			fail(409, "MOVED")
			break
		}
		var list []BlockSize
		if json.Unmarshal([]byte(r.FormValue("blocks")), &list) != nil {
			fail(400, "INVALID")
			break
		}
		tk := &ticket{parent: parent, blocks: map[string]bool{}, missing: map[string]bool{}}
		var add, free int64
		for _, b := range list {
			if tk.blocks[b.ID] || (q.blocks[b.ID] == nil && b.Size <= 0) {
				fail(400, "INVALID")
				q.mu.Unlock()
				return
			}
			tk.blocks[b.ID] = true
			if q.blocks[b.ID] == nil {
				tk.missing[b.ID] = true
				add += b.Size
			}
		}
		for id := range q.inUse {
			if !tk.blocks[id] {
				free += int64(len(q.blocks[id]))
			}
		}
		if add > free && q.used()+add-free > q.limit {
			fail(413, "QUOTA")
			break
		}
		file, _, _ := r.FormFile("manifest")
		tk.sealed, _ = io.ReadAll(file)
		name := "t" + strconv.Itoa(q.prepares)
		q.tickets[name] = tk
		if q.expireOnce { // its blocks still go up, its commit fails
			q.expireOnce = false
			q.expired = name
		}
		var miss []string
		for id := range tk.missing {
			miss = append(miss, id)
		}
		js(200, map[string]any{"ticket": name, "expiresAt": "2026-11-02T12:00:00Z", "missing": miss})
	case r.Method == http.MethodPost && strings.HasSuffix(p, "/commit"):
		var in struct{ Ticket string }
		json.NewDecoder(r.Body).Decode(&in)
		tk := q.tickets[in.Ticket]
		if in.Ticket == q.expired {
			delete(q.tickets, in.Ticket)
			tk = nil
		}
		if tk == nil {
			fail(410, "TICKET_EXPIRED")
			break
		}
		for id := range tk.missing {
			if q.blocks[id] == nil {
				fail(400, "MISSING_BLOCKS")
				q.mu.Unlock()
				return
			}
		}
		if tk.parent != q.head {
			fail(409, "MOVED")
			break
		}
		delete(q.tickets, in.Ticket)
		js(200, map[string]int64{"n": q.save(tk.blocks, tk.sealed)})
	case r.Method == http.MethodDelete && strings.HasPrefix(p, "/preparations/"):
		q.cancelled++
		delete(q.tickets, strings.TrimPrefix(p, "/preparations/"))
		js(200, map[string]bool{"ok": true})
	default:
		q.mu.Unlock()
		q.fakeCloud.serve(w, r)
		return
	}
	q.mu.Unlock()
}

// noise is text that doesn't compress: its blocks take their size.
func noise(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// Above the storage limit the person is never stuck: deleting syncs,
// growing is refused, an edit that frees as much as it adds goes through a
// preparation, and back under the limit all is as before.
func TestSyncAboveTheLimit(t *testing.T) {
	ctx := context.Background()
	q := newQuotaCloud(t)
	keys := testKeys(t)
	a, b := newDevice(t, q.fakeCloud, keys), newDevice(t, q.fakeCloud, keys)
	sync := func(d *device) error { _, err := d.s.Sync(ctx, nil); return err }

	a.write(t, "grande.md", noise(60_000))
	a.write(t, "outra.md", noise(40_000))
	if err := sync(a); err != nil {
		t.Fatal(err)
	}
	// A plan with less storage: now above it.
	q.qmu.Lock()
	q.limit = q.used() - 1
	q.qmu.Unlock()

	// Deleting: no new block, it goes.
	removeNote(t, a, "outra.md")
	if err := sync(a); err != nil {
		t.Fatalf("delete above the limit: %v", err)
	}
	if q.prepares != 0 {
		t.Fatal("a delete went through a preparation")
	}

	// Growing: refused, with nothing left behind.
	a.write(t, "nova.md", noise(80_000))
	if err := sync(a); !errors.Is(err, ErrQuota) {
		t.Fatalf("growing above the limit: %v", err)
	}
	if len(q.tickets) != 0 || q.blocks == nil {
		t.Fatal("a ticket was left open")
	}

	// An edit that adds less than it frees: prepared, sent, saved.
	removeNote(t, a, "nova.md")
	a.write(t, "grande.md", noise(20_000))
	before := q.prepares
	if err := sync(a); err != nil {
		t.Fatalf("shrinking edit above the limit: %v", err)
	}
	if q.prepares == before {
		t.Fatal("the edit didn't go through a preparation")
	}
	if err := sync(b); err != nil {
		t.Fatal(err)
	}
	if b.read("grande.md") != a.read("grande.md") || b.read("outra.md") != "" || b.read("nova.md") != "" {
		t.Fatal("the other computer didn't get the version")
	}

	// Back under the limit: the usual way again.
	q.qmu.Lock()
	q.limit = 1 << 40
	q.qmu.Unlock()
	before = q.prepares
	a.write(t, "depois.md", noise(10_000))
	if err := sync(a); err != nil {
		t.Fatal(err)
	}
	if q.prepares != before {
		t.Fatal("still preparing under the limit")
	}
}

// Any failure after the preparation lets the ticket go; the next sync
// prepares again.
func TestPreparedCancelledOnFailure(t *testing.T) {
	ctx := context.Background()
	q := newQuotaCloud(t)
	a := newDevice(t, q.fakeCloud, testKeys(t))
	a.write(t, "nota.md", noise(30_000))
	if _, err := a.s.Sync(ctx, nil); err != nil {
		t.Fatal(err)
	}
	q.qmu.Lock()
	q.limit, q.expireOnce = q.used()-1, true
	q.qmu.Unlock()
	a.write(t, "nota.md", noise(10_000))
	if _, err := a.s.Sync(ctx, nil); !errors.Is(err, ErrTicketExpired) {
		t.Fatalf("expired ticket: %v", err)
	}
	if q.cancelled != 1 {
		t.Fatalf("cancelled %d times", q.cancelled)
	}
	if _, err := a.s.Sync(ctx, nil); err != nil {
		t.Fatalf("again: %v", err)
	}
	if len(q.tickets) != 0 || q.prepares != 2 {
		t.Fatalf("open %d, prepared %d", len(q.tickets), q.prepares)
	}
}

func TestPrepareChecks(t *testing.T) {
	c := &Client{BaseURL: "http://127.0.0.1:1", HTTP: http.DefaultClient}
	id := strings.Repeat("a", 64)
	if _, err := c.Prepare(context.Background(), vaultID, 0, []BlockSize{{id, 1}, {id, 1}}, nil); err == nil {
		t.Fatal("a repeated block was sent")
	}
	for _, tk := range []string{"", "../x", "a b", strings.Repeat("a", 201)} {
		if validTicket(tk) {
			t.Errorf("ticket %q accepted", tk)
		}
	}
}

func removeNote(t *testing.T, d *device, name string) {
	t.Helper()
	if err := os.Remove(filepath.Join(d.dir, name)); err != nil {
		t.Fatal(err)
	}
}
