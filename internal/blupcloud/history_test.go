package blupcloud

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const vaultID = "6f1c1f9e-1b2a-4c3d-9e8f-0a1b2c3d4e5f"

func TestPurgeHistory(t *testing.T) {
	var got []string
	answer := func(w http.ResponseWriter) { w.Write([]byte(`{"freedBytes":1048576,"storageBytes":5000}`)) }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.URL.Path)
		answer(w)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client(), Token: "t"}
	ctx := context.Background()

	r, err := c.PurgeHistory(ctx, vaultID)
	if err != nil || r.FreedBytes != 1048576 || r.StorageBytes != 5000 {
		t.Fatalf("%+v %v", r, err)
	}
	if len(got) != 1 || got[0] != "DELETE /api/sync/vaults/"+vaultID+"/history" {
		t.Fatalf("sent %v", got)
	}
	// A vault id that isn't one never leaves.
	if _, err := c.PurgeHistory(ctx, "../me"); !errors.Is(err, ErrVaultMissing) || len(got) != 1 {
		t.Fatalf("bad id: %v, %v", err, got)
	}
	for _, c2 := range []struct {
		status int
		body   string
		want   error
	}{
		{409, `{"code":"SYNC_BUSY"}`, ErrSyncBusy},
		{404, `{"code":"NOT_FOUND"}`, ErrVaultMissing},
		{401, `{"code":"UNAUTHORIZED"}`, ErrSignedOut},
		{200, `{"freedBytes":-1,"storageBytes":0}`, ErrBadAnswer},
	} {
		answer = func(w http.ResponseWriter) { w.WriteHeader(c2.status); w.Write([]byte(c2.body)) }
		if _, err := c.PurgeHistory(ctx, vaultID); !errors.Is(err, c2.want) {
			t.Errorf("%d %s: %v, want %v", c2.status, c2.body, err, c2.want)
		}
	}
}

func TestMeStorageGrace(t *testing.T) {
	body := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client(), Token: "t"}
	me := func(usage string) Account {
		body = `{"id":"u","plan":{"name":"pro","limits":{"storageMB":1024}},"usage":` + usage + `}`
		a, err := c.Me(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	a := me(`{"storageBytes":9,"storageLimitBytes":1073741824,"historyBytes":3,"storageGraceUntil":"2026-11-16T00:00:00Z","storageGraceLimitBytes":10737418240}`)
	if a.Usage.StorageGraceUntil == nil || *a.Usage.StorageGraceLimitBytes != 10737418240 || a.Usage.HistoryBytes == nil || *a.Usage.HistoryBytes != 3 {
		t.Fatalf("%+v", a.Usage)
	}
	for _, u := range []string{
		`{"storageGraceUntil":"amanhã","storageGraceLimitBytes":10}`,
		`{"storageGraceUntil":"2026-11-16T00:00:00Z"}`,
		`{"storageGraceUntil":null,"storageGraceLimitBytes":10}`,
	} {
		if a := me(u); a.Usage.StorageGraceUntil != nil || a.Usage.StorageGraceLimitBytes != nil {
			t.Errorf("%s kept a grace: %+v", u, a.Usage)
		}
	}
	// Unknown (null) stays unknown, not 0.
	if a := me(`{"historyBytes":null}`); a.Usage.HistoryBytes != nil {
		t.Fatal("an unknown history became a number")
	}
	body = `{"id":"u","plan":{"name":"pro","limits":{}},"usage":{"historyBytes":-1}}`
	if _, err := c.Me(context.Background()); !errors.Is(err, ErrBadAnswer) {
		t.Fatalf("negative history accepted: %v", err)
	}
}

func TestDeleteVault(t *testing.T) {
	var got []string
	status, body := 200, `{"freedBytes":0,"storageBytes":0}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodDelete {
			w.WriteHeader(status)
		}
		w.Write([]byte(body))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client(), Token: "t"}
	ctx := context.Background()
	if err := c.DeleteVault(ctx, vaultID); err != nil || len(got) != 1 || got[0] != "DELETE /api/sync/vaults/"+vaultID {
		t.Fatalf("%v %v", err, got)
	}
	if err := c.DeleteVault(ctx, "../me"); !errors.Is(err, ErrVaultMissing) || len(got) != 1 {
		t.Fatalf("bad id: %v, %v", err, got)
	}
	// Already gone (not listed) is done; a sync in the way is not.
	status, body = 404, `{"code":"NOT_FOUND","vaults":[]}`
	if err := c.DeleteVault(ctx, vaultID); err != nil {
		t.Fatalf("gone: %v", err)
	}
	// A 404 for a vault still listed: a cloud without the route.
	status, body = 404, `{"code":"NOT_FOUND","vaults":[{"id":"`+vaultID+`"}]}`
	if err := c.DeleteVault(ctx, vaultID); !errors.Is(err, ErrCantDelete) {
		t.Fatalf("old cloud: %v", err)
	}
	status, body = 409, `{"code":"SYNC_BUSY"}`
	if err := c.DeleteVault(ctx, vaultID); !errors.Is(err, ErrSyncBusy) {
		t.Fatalf("busy: %v", err)
	}
}

func TestRenameVault(t *testing.T) {
	var got []string
	status := 200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = append(got, r.Method+" "+r.URL.Path+" "+string(b))
		w.WriteHeader(status)
		w.Write([]byte(`{"code":"NOT_FOUND"}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client(), Token: "t"}
	if err := c.RenameVault(context.Background(), vaultID, []byte("novo")); err != nil || len(got) != 1 || got[0] != `PATCH /api/sync/vaults/`+vaultID+` {"name":"bm92bw=="}` {
		t.Fatalf("%v %v", err, got)
	}
	status = 404
	if err := c.RenameVault(context.Background(), vaultID, []byte("novo")); !errors.Is(err, ErrVaultMissing) {
		t.Fatalf("gone: %v", err)
	}
}

func TestVaultGone(t *testing.T) {
	list, status := `{"vaults":[]}`, 200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sync/vaults/"+vaultID {
			w.WriteHeader(404)
			w.Write([]byte(`{"code":"NOT_FOUND"}`))
			return
		}
		w.WriteHeader(status)
		w.Write([]byte(list))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client(), Token: "t"}
	ctx := context.Background()
	if _, err := c.VaultHead(ctx, vaultID); !errors.Is(err, ErrVaultMissing) {
		t.Fatalf("head: %v", err)
	}
	if !c.VaultGone(ctx, vaultID) {
		t.Fatal("not in the list, but not gone")
	}
	list = `{"vaults":[{"id":"` + vaultID + `"}]}`
	if c.VaultGone(ctx, vaultID) {
		t.Fatal("listed, but gone")
	}
	// Without the list, never gone.
	status = 500
	if c.VaultGone(ctx, vaultID) {
		t.Fatal("offline, but gone")
	}
}

func TestStudioAI(t *testing.T) {
	body, auth := `{"voice":{"plan":{"models":["microsoft/mai-voice-2.1"],"recommended":null},"own":{"models":["microsoft/mai-voice-2.1","vendor/voz"],"recommended":"vendor/voz"}},"translate":{"plan":["anthropic/claude-haiku-5.5"],"own":["deepseek/deepseek-v4.1-flash"]}}`, "x"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/studio-ai" {
			w.WriteHeader(404)
			return
		}
		w.Write([]byte(body))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	ai, err := c.StudioAI(context.Background())
	if err != nil || ai.Voice.Own.Models[1] != "vendor/voz" || ai.Voice.Own.Recommended != "vendor/voz" || ai.Voice.Plan.Recommended != "" || ai.Translate.Plan[0] != "anthropic/claude-haiku-5.5" || auth != "" {
		t.Fatalf("%+v %v auth=%q", ai, err, auth)
	}
	// A list that isn't one is refused (the Studio keeps what it had).
	for _, bad := range []string{
		`{"voice":{"plan":{"models":["../x"]},"own":{"models":["a/b"]}},"translate":{"plan":["a/b"],"own":["a/b"]}}`,
		`{"voice":{"plan":{"models":["a/b"],"recommended":"c/d"},"own":{"models":["a/b"]}},"translate":{"plan":["a/b"],"own":["a/b"]}}`,
	} {
		body = bad
		if _, err := c.StudioAI(context.Background()); !errors.Is(err, ErrBadAnswer) {
			t.Fatalf("%s: %v", bad, err)
		}
	}
}
