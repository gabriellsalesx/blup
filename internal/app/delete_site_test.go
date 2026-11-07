package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

func TestDeleteSite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	base := t.TempDir()
	notes := filepath.Join(base, "vault", "Site")
	os.MkdirAll(notes, 0o755)
	os.WriteFile(filepath.Join(notes, "Post.md"), []byte("x"), 0o644)
	siteDir := filepath.Join(base, "site")
	os.MkdirAll(siteDir, 0o755)
	os.WriteFile(filepath.Join(siteDir, "blup.toml"), []byte("version = 1\ntitle = 'X'\nmodel = 'mochi'\nvault = '"+filepath.Join(base, "vault")+"'\nfolder = 'Site'\n"), 0o644)
	if err := RememberSite(siteDir); err != nil {
		t.Fatal(err)
	}
	parts, err := PartsOf(siteDir)
	if err != nil || parts.NotesOwned || parts.NotesDir != notes {
		t.Fatalf("parts: %+v %v", parts, err)
	}
	problems, err := DeleteSite(context.Background(), siteDir, DeleteChoice{Site: true, Notes: true})
	if err != nil || len(problems) != 0 {
		t.Fatalf("delete: %v %v", problems, err)
	}
	for _, p := range []string{siteDir, notes} {
		if _, err := os.Stat(p); err == nil {
			t.Fatalf("%s still there", p)
		}
	}
	if _, err := os.Stat(filepath.Join(data, "Trash", "files", "Site", "Post.md")); err != nil {
		t.Fatal("notes not in the trash")
	}
	if _, err := os.Stat(filepath.Join(base, "vault")); err != nil {
		t.Fatal("the vault itself was deleted")
	}
	if g, _ := config.LoadGlobal(); len(g.Sites) != 0 {
		t.Fatalf("still listed: %v", g.Sites)
	}
}

// Apagar o site também o tira do Blup Cloud quando ele está publicado lá
// (DELETE /api/sites/<id>). Um site que já não existe lá não é problema;
// uma falha de verdade volta com o link do painel.
func TestDeleteSiteFromBlupCloud(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var mu sync.Mutex
	var calls []string
	fail, listed := false, true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
		switch {
		case r.Method == http.MethodDelete && fail:
			w.WriteHeader(500)
		case r.Method == http.MethodDelete && !listed:
			w.WriteHeader(404)
			w.Write([]byte(`{"code":"NOT_FOUND"}`))
		case r.Method == http.MethodDelete:
			w.Write([]byte(`{"ok":true}`))
		case r.URL.Path == "/api/sites" && listed:
			w.Write([]byte(`{"limit":1,"sites":[{"id":"s1","slug":"meu-blog"}]}`))
		case r.URL.Path == "/api/sites":
			w.Write([]byte(`{"limit":1,"sites":[]}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	if err := (config.Credentials{BlupToken: "tok"}).Save(); err != nil {
		t.Fatal(err)
	}
	newSite := func() string {
		base := t.TempDir()
		os.MkdirAll(filepath.Join(base, "vault"), 0o755)
		dir := filepath.Join(base, "site")
		s := config.NewSite(dir)
		os.MkdirAll(dir, 0o755)
		s.Title, s.Model, s.Vault = "Meu blog", "mochi", filepath.Join(base, "vault")
		s.Deploy.Provider = "blup"
		s.Deploy.Blup = config.BlupSite{ID: "s1", Slug: "meu-blog", URL: "https://meu-blog.getblup.site"}
		s.Sync.Remote, s.Sync.Vault = config.SyncBlup, "6f1c1f9e-1b2a-4c3d-9e8f-0a1b2c3d4e5f"
		if err := s.Save(); err != nil {
			t.Fatal(err)
		}
		RememberSite(dir)
		return dir
	}
	ctx := context.Background()

	dir := newSite()
	if p, err := PartsOf(dir); err != nil || p.BlupSite != "meu-blog.getblup.site" {
		t.Fatalf("parts = %+v %v", p, err)
	}
	// Only the Blup Cloud site: the folder stays and forgets the address.
	if problems, err := DeleteSite(ctx, dir, DeleteChoice{BlupSite: true}); err != nil || len(problems) != 0 {
		t.Fatalf("delete: %v %v", problems, err)
	}
	if strings.Join(calls, ",") != "DELETE /api/sites/s1 Bearer tok" {
		t.Fatalf("calls = %v", calls)
	}
	if s, _ := config.LoadSite(dir); s.Deploy.Blup.ID != "" || s.Deploy.Provider != "" {
		t.Fatalf("still points at Blup Cloud: %+v", s.Deploy)
	}
	if p, _ := PartsOf(dir); p.BlupSite != "" {
		t.Fatalf("parts after = %+v", p)
	}

	// Not asked: nothing is sent.
	calls = nil
	dir = newSite()
	if _, err := DeleteSite(ctx, dir, DeleteChoice{Site: true}); err != nil || len(calls) != 0 {
		t.Fatalf("not asked: %v %v", calls, err)
	}

	// Already gone (deleted in the panel): fine.
	dir = newSite()
	listed = false
	if problems, err := DeleteSite(ctx, dir, DeleteChoice{BlupSite: true, Site: true}); err != nil || len(problems) != 0 {
		t.Fatalf("already gone: %v %v", problems, err)
	}

	// A real failure comes back with the panel's link.
	dir = newSite()
	listed, fail = true, true
	problems, err := DeleteSite(ctx, dir, DeleteChoice{BlupSite: true})
	if err != nil || len(problems) != 1 || problems[0].Part != "blupSite" || !strings.HasPrefix(problems[0].Link, srv.URL) {
		t.Fatalf("failure: %+v %v", problems, err)
	}
	if s, _ := config.LoadSite(dir); s.Deploy.Blup.ID != "s1" {
		t.Fatal("forgot an address still online")
	}

	// The notes on Blup Cloud: the vault goes, and the folder stops syncing.
	calls, fail = nil, false
	dir = newSite()
	if p, _ := PartsOf(dir); !p.CloudNotes {
		t.Fatalf("parts = %+v", p)
	}
	if problems, err := DeleteSite(ctx, dir, DeleteChoice{CloudNotes: true}); err != nil || len(problems) != 0 {
		t.Fatalf("cloud notes: %v %v", problems, err)
	}
	if strings.Join(calls, ",") != "DELETE /api/sync/vaults/6f1c1f9e-1b2a-4c3d-9e8f-0a1b2c3d4e5f Bearer tok" {
		t.Fatalf("calls = %v", calls)
	}
	if s, _ := config.LoadSite(dir); s.Sync.Vault != "" || s.Sync.Remote != "" {
		t.Fatalf("still syncs: %+v", s.Sync)
	}
	// It failing keeps the link, and says where to do it by hand.
	dir, fail = newSite(), true
	if problems, _ := DeleteSite(ctx, dir, DeleteChoice{CloudNotes: true}); len(problems) != 1 || problems[0].Part != "cloudNotes" {
		t.Fatalf("failure: %+v", problems)
	}
	if s, _ := config.LoadSite(dir); s.Sync.Vault == "" {
		t.Fatal("forgot a vault still online")
	}
}
