package blupcloud

import (
	"context"
	"errors"
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
	if a.Usage.StorageGraceUntil == nil || *a.Usage.StorageGraceLimitBytes != 10737418240 || a.Usage.HistoryBytes != 3 {
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
	body = `{"id":"u","plan":{"name":"pro","limits":{}},"usage":{"historyBytes":-1}}`
	if _, err := c.Me(context.Background()); !errors.Is(err, ErrBadAnswer) {
		t.Fatalf("negative history accepted: %v", err)
	}
}
