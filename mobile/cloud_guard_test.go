package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

// A vault id names a folder of the cache here (removed with it when the
// vault is gone): an id the cloud could not have made ("../..") is refused
// before anything is saved; a short notes password never reaches the cloud.
func TestCloudGuards(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BLUP_CLOUD_URL", "")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, `{"code":"NOT_FOUND"}`, http.StatusNotFound)
	}))
	defer srv.Close()
	old := cloudURL
	defer func() { cloudURL = old }()
	cloudURL = srv.URL
	if err := prepare(); err != nil {
		t.Fatal(err)
	}
	if err := (config.Credentials{BlupToken: "tok"}).Save(); err != nil {
		t.Fatal(err)
	}
	s := &CloudService{}
	for _, id := range []string{"../..", "a/b", "..", "x\\y"} {
		if err := s.Link(t.Context(), id, "Notas"); err == nil {
			t.Fatalf("Link(%q) accepted", id)
		}
	}
	if dir, err := syncDir(); err == nil {
		if _, err := os.Stat(filepath.Join(dir, "vault")); err == nil {
			t.Fatal("a bad id was saved as the link")
		}
	}
	if r, err := s.CreatePassword(t.Context(), "curta"); err == nil || r != "" {
		t.Fatal("a short notes password was accepted")
	}
	if calls.Load() != 0 {
		t.Fatalf("%d calls reached the cloud", calls.Load())
	}
}
