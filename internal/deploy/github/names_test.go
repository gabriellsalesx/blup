package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(t *testing.T, h http.HandlerFunc) (*Client, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	c := New("tok")
	c.BaseURL = srv.URL
	c.MinWriteInterval = -1
	c.sleep = func(ctx context.Context, d time.Duration) error { return ctx.Err() }
	return c, &hits
}

// Names that came from a synced site.toml or the window and would point
// the API somewhere else (or read as options, or as a git transport) are
// refused before any request.
func TestBadNamesNeverSent(t *testing.T) {
	c, hits := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"login": "Maria"})
	})
	bad := []string{
		"..", ".", "../user", "a/b", "a?b", "a#b", "%2e%2e", "-oProxyCommand=x", "--upload-pack=x",
		"ext::sh -c touch% /tmp/x", "file:///etc", "fd::0", "x.git", "a b", "a\nb", "á", "",
		strings.Repeat("a", 101),
	}
	ctx := context.Background()
	for _, name := range bad {
		if _, err := c.FileContent(ctx, "Maria", name, "x"); !errors.Is(err, ErrBadName) {
			t.Errorf("repo %q: want ErrBadName, got %v", name, err)
		}
		if err := c.EnableIssues(ctx, "Maria", name); !errors.Is(err, ErrBadName) {
			t.Errorf("repo %q (EnableIssues): want ErrBadName, got %v", name, err)
		}
		if _, err := c.EnsureRepoWith(ctx, "Maria", name, EnsureRepoOptions{}); err == nil {
			t.Errorf("repo %q: EnsureRepoWith accepted it", name)
		}
	}
	for _, owner := range []string{"..", "-x", "a/b", "a--b", "x-", "ext::sh", strings.Repeat("a", 40)} {
		if _, err := c.FileContent(ctx, owner, "site", "x"); !errors.Is(err, ErrBadName) {
			t.Errorf("owner %q: want ErrBadName, got %v", owner, err)
		}
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("%d requests went out for invalid names", n)
	}
	for _, ok := range []string{"site", "maria.github.io", ".github", "my_site-2", "A"} {
		if !ValidRepoName(ok) {
			t.Errorf("valid name %q refused", ok)
		}
	}
}

// A key limited to chosen repositories cannot create one: the error says
// which repository to create and links to GitHub's form, filled in.
func TestRepoMissingWhenKeyCannotCreate(t *testing.T) {
	var posted atomic.Bool
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/user":
			writeJSON(w, 200, map[string]any{"login": "Maria"})
		case r.Method == http.MethodGet && r.URL.Path == "/repos/Maria/site-notas":
			msg(w, 404, "Not Found")
		case r.Method == http.MethodPost && r.URL.Path == "/user/repos":
			posted.Store(true)
			msg(w, 403, "Resource not accessible by personal access token")
		default:
			msg(w, 500, "unexpected "+r.Method+" "+r.URL.Path)
		}
	})
	_, err := c.EnsureRepoWith(context.Background(), "Maria", "site-notas", EnsureRepoOptions{Private: true, Description: "Notas"})
	if !posted.Load() {
		t.Fatal("creation was not tried (a classic key could still create it)")
	}
	if !errors.Is(err, ErrRepoMissing) || errors.Is(err, ErrPermission) {
		t.Fatalf("want ErrRepoMissing only, got %v", err)
	}
	var rm *RepoMissingError
	if !errors.As(err, &rm) {
		t.Fatal("not a RepoMissingError")
	}
	u, _ := url.Parse(rm.CreateURL)
	q := u.Query()
	if u.Host != "github.com" || u.Path != "/new" || q.Get("name") != "site-notas" || q.Get("visibility") != "private" || q.Get("owner") != "Maria" {
		t.Fatalf("create link: %s", rm.CreateURL)
	}
}

// The key form asks for the repository permissions Blup uses and no
// Administration (which would let the key delete or change any repository).
func TestTokenURLLeastPrivilege(t *testing.T) {
	u, err := url.Parse(TokenURL("blup", "Publicar"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Host != "github.com" || u.Path != "/settings/personal-access-tokens/new" {
		t.Fatalf("url %s", u)
	}
	for _, p := range []string{"administration", "workflows", "secrets", "actions", "members", "organization_administration"} {
		if q.Has(p) {
			t.Errorf("asks for %s", p)
		}
	}
	want := map[string]string{"contents": "write", "pages": "write", "issues": "write", "metadata": "read", "name": "blup", "expires_in": "366"}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, q.Get(k), v)
		}
	}
}

// Issues already on: nothing is changed (that would need Administration).
func TestEnableIssuesReadsFirst(t *testing.T) {
	var patched atomic.Bool
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patched.Store(true)
			msg(w, 403, "Resource not accessible by personal access token")
			return
		}
		writeJSON(w, 200, map[string]any{"has_issues": true})
	})
	if err := c.EnableIssues(context.Background(), "Maria", "site-feedback"); err != nil {
		t.Fatal(err)
	}
	if patched.Load() {
		t.Fatal("PATCH sent although issues were on")
	}
}
