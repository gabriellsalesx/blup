package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

// recorder is a fake Blup Cloud that keeps every request in order (method,
// path, token and body). Down, it drops the connection, as when offline.
type recorder struct {
	mu   sync.Mutex
	log  []string
	down bool
}

func (rc *recorder) start(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc.mu.Lock()
		down := rc.down
		if !down {
			b, _ := io.ReadAll(r.Body)
			rc.log = append(rc.log, r.Method+" "+r.URL.Path+" "+strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")+" "+strings.TrimSpace(string(b)))
		}
		rc.mu.Unlock()
		if down {
			hj, _ := w.(http.Hijacker)
			c, _, _ := hj.Hijack()
			c.Close()
			return
		}
		w.Write([]byte(`{"limit":50,"used":0}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	return srv
}

func (rc *recorder) requests() []string {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return append([]string(nil), rc.log...)
}

// Sair da conta cancela os lembretes esperando no Blup Cloud (lista vazia)
// antes de encerrar a sessão; sem rede, a sessão sai daqui mesmo assim e o
// resto fica para o próximo arranque ou login.
func TestBlupSignOutCancelsReminders(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var rc recorder
	rc.start(t)
	ctx := context.Background()
	if err := (config.Credentials{BlupToken: "tok1", NotesKey: "k"}).Save(); err != nil {
		t.Fatal(err)
	}
	pending, err := BlupSignOut(ctx)
	if err != nil || pending {
		t.Fatalf("sign out: %v pending %v", err, pending)
	}
	want := []string{`PUT /api/reminders tok1 {"reminders":[]}`, "POST /api/auth/sign-out tok1 {}"}
	if got := rc.requests(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s", strings.Join(got, "\n"))
	}
	if c, _ := config.LoadCredentials(); c.BlupToken != "" || c.NotesKey != "" {
		t.Fatalf("still signed in: %+v", c)
	}
	if SignOutPending() {
		t.Fatal("pending after a sign out that went through")
	}

	// Offline.
	if err := (config.Credentials{BlupToken: "tok2"}).Save(); err != nil {
		t.Fatal(err)
	}
	rc.mu.Lock()
	rc.down, rc.log = true, nil
	rc.mu.Unlock()
	pending, err = BlupSignOut(ctx)
	if err != nil || !pending || !SignOutPending() {
		t.Fatalf("offline sign out: %v pending %v", err, pending)
	}
	if c, _ := config.LoadCredentials(); c.BlupToken != "" {
		t.Fatal("the session stayed here")
	}
	cfg, _ := config.Dir()
	if st, err := os.Stat(cfg + "/signout-pending"); err != nil || (os.PathSeparator == '/' && st.Mode().Perm() != 0o600) {
		t.Fatalf("pending file: %v %v", st, err)
	}
	if RetrySignOut(ctx) || !SignOutPending() {
		t.Fatal("still offline: must stay pending")
	}
	// Back online: the old session's reminders go empty and it ends.
	rc.mu.Lock()
	rc.down = false
	rc.mu.Unlock()
	if !RetrySignOut(ctx) || SignOutPending() {
		t.Fatal("retry did not finish")
	}
	want = []string{`PUT /api/reminders tok2 {"reminders":[]}`, "POST /api/auth/sign-out tok2 {}"}
	if got := rc.requests(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("retry requests:\n%s", strings.Join(got, "\n"))
	}
	if RetrySignOut(ctx) {
		t.Fatal("nothing pending, nothing done")
	}
}
