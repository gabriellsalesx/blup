package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBlobSHAMatchesGitHashObject(t *testing.T) {
	// Vectors computed with `git hash-object --stdin`.
	cases := []struct{ in, want string }{
		{"", "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"},
		{"hello world\n", "3b18e512dba79e4c8300dd08aeb37f8e728b8dad"},
		{"Olá, mundo! \xc3\xa7\x00\xff binary", "02b96d449daa76501df3652698cc42a2e947227d"},
	}
	for _, c := range cases {
		if got := BlobSHA([]byte(c.in)); got != c.want {
			t.Errorf("BlobSHA(%q) = %s, want %s", c.in, got, c.want)
		}
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	os.WriteFile(p, []byte("hello world\n"), 0o644)
	if got, _ := hashFile(p, 12); got != cases[1].want {
		t.Errorf("hashFile = %s", got)
	}
}

func TestExpectedURL(t *testing.T) {
	cases := [][3]string{
		{"Maria", "maria.github.io", "https://maria.github.io/"},
		{"maria", "Maria.GitHub.io", "https://maria.github.io/"},
		{"Maria", "blog", "https://maria.github.io/blog/"},
		{"maria", "other.github.io", "https://maria.github.io/other.github.io/"},
	}
	for _, c := range cases {
		if got := ExpectedURL(c[0], c[1]); got != c[2] {
			t.Errorf("ExpectedURL(%s,%s) = %s, want %s", c[0], c[1], got, c[2])
		}
	}
}

func TestViewer(t *testing.T) {
	f := newFake(t)
	login, err := f.client().Viewer(context.Background())
	if err != nil || login != "Maria" {
		t.Fatalf("Viewer = %q, %v", login, err)
	}
	c := f.client()
	c.Token = "wrong"
	if _, err := c.Viewer(context.Background()); !errors.Is(err, ErrBadToken) {
		t.Fatalf("want ErrBadToken, got %v", err)
	}
}

func TestEnsureRepo(t *testing.T) {
	ctx := context.Background()
	f := newFake(t)
	c := f.client()

	r, err := c.EnsureRepo(ctx, "", "notes")
	if err != nil {
		t.Fatal(err)
	}
	if !r.Created || r.Owner != "Maria" || r.Private || r.DefaultBranch != "main" {
		t.Fatalf("created repo = %+v", r)
	}
	if f.count("POST /user/repos") != 1 {
		t.Fatal("expected one create call")
	}

	r, err = c.EnsureRepo(ctx, "maria", "NOTES")
	if err != nil {
		t.Fatal(err)
	}
	if r.Created {
		t.Fatal("existing repo reported as created")
	}
	if f.count("POST /user/repos") != 1 {
		t.Fatal("existing repo must not be created again")
	}

	// Someone else's repo the user cannot push to.
	other := f.addRepo("joao", "site")
	other.push = false
	if _, err := c.EnsureRepo(ctx, "joao", "site"); !errors.Is(err, ErrRepoConflict) {
		t.Fatalf("want ErrRepoConflict, got %v", err)
	}
	// Cannot create under another account.
	if _, err := c.EnsureRepo(ctx, "someone", "x"); !errors.Is(err, ErrRepoConflict) {
		t.Fatalf("want ErrRepoConflict, got %v", err)
	}
	// Create reports "name already exists" (e.g. repo invisible to token).
	f.hook = func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/repos/") {
			msg(w, 404, "Not Found")
			return true
		}
		return false
	}
	if _, err := c.EnsureRepo(ctx, "", "notes"); !errors.Is(err, ErrRepoConflict) {
		t.Fatalf("want ErrRepoConflict on 422, got %v", err)
	}
	// Missing Administration permission.
	f.hook = func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPost && r.URL.Path == "/user/repos" {
			msg(w, 403, "Resource not accessible by personal access token")
			return true
		}
		return false
	}
	if _, err := c.EnsureRepo(ctx, "", "brand-new"); !errors.Is(err, ErrPermission) {
		t.Fatalf("want ErrPermission, got %v", err)
	}
}

func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	os.RemoveAll(dir)
	for p, c := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		os.MkdirAll(filepath.Dir(full), 0o755)
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPublishOrphanThenIncremental(t *testing.T) {
	ctx := context.Background()
	f := newFake(t)
	rp := f.addRepo("Maria", "blog")
	rp.autoInit()
	c := f.client()
	c.InlineTextLimit = -1 // force blobs so we can count them
	dir := filepath.Join(t.TempDir(), "public")

	png := "\x89PNG\x00\x01binary"
	writeTree(t, dir, map[string]string{
		"index.html":       "<h1>home</h1>",
		"posts/a.html":     "a",
		"posts/b.html":     "b",
		"img/logo.png":     png,
		"img/copy.png":     png, // same content: one upload
		".git/config":      "ignored",
		"posts/.nojekyll2": "x",
	})

	var mu sync.Mutex
	var phases []string
	res, err := c.Publish(ctx, "Maria", "blog", dir, PublishOptions{CNAME: "blog.example.com"}, func(p Progress) {
		mu.Lock()
		phases = append(phases, p.Phase)
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Orphan || res.Branch != "gh-pages" || res.CommitSHA == "" {
		t.Fatalf("result = %+v", res)
	}
	cm := rp.commits[rp.refs["gh-pages"]]
	if len(cm.parents) != 0 {
		t.Fatalf("first publish must be an orphan commit, parents=%v", cm.parents)
	}
	got := rp.files("gh-pages")
	if got["CNAME"] != "blog.example.com\n" || got[".nojekyll"] != "" {
		t.Fatalf("missing CNAME/.nojekyll: %v", got)
	}
	if _, ok := got[".nojekyll"]; !ok {
		t.Fatal(".nojekyll missing")
	}
	if _, ok := got[".git/config"]; ok {
		t.Fatal(".git must be skipped")
	}
	if got["img/copy.png"] != png || got["posts/a.html"] != "a" || len(got) != 8 {
		t.Fatalf("tree = %v", got)
	}
	// 7 distinct contents (two identical PNGs): index,a,b,png,.nojekyll2,.nojekyll,CNAME.
	if n := f.count("POST /repos/Maria/blog/git/blobs"); n != 7 {
		t.Fatalf("blob uploads = %d, want 7", n)
	}
	if f.count("POST /repos/Maria/blog/git/refs") != 1 {
		t.Fatal("expected ref creation")
	}
	if !slices.Contains(phases, PhaseUpload) || phases[len(phases)-1] != PhaseDone {
		t.Fatalf("phases = %v", phases)
	}
	// main branch untouched.
	if rp.files("main")["README.md"] == "" {
		t.Fatal("main branch changed")
	}

	// Second publish: change one file, delete one, add one.
	first := rp.refs["gh-pages"]
	writeTree(t, dir, map[string]string{
		"index.html":   "<h1>home v2</h1>",
		"posts/a.html": "a",
		"img/logo.png": png,
		"img/copy.png": png,
		"posts/c.html": "c",
	})
	before := f.count("POST /repos/Maria/blog/git/blobs")
	res, err = c.Publish(ctx, "Maria", "blog", dir, PublishOptions{Message: "update"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n := f.count("POST /repos/Maria/blog/git/blobs") - before; n != 2 {
		t.Fatalf("second publish uploaded %d blobs, want 2 (index + c)", n)
	}
	if res.Orphan || res.Uploaded != 2 || res.Skipped != 4 || res.Deleted != 3 {
		// deleted: posts/b.html, posts/.nojekyll2, CNAME
		t.Fatalf("result = %+v", res)
	}
	cm = rp.commits[rp.refs["gh-pages"]]
	if len(cm.parents) != 1 || cm.parents[0] != first || cm.message != "update" {
		t.Fatalf("commit = %+v", cm)
	}
	got = rp.files("gh-pages")
	if _, ok := got["posts/b.html"]; ok {
		t.Fatal("deleted file still present")
	}
	if _, ok := got["CNAME"]; ok {
		t.Fatal("CNAME should disappear when not set")
	}
	if got["index.html"] != "<h1>home v2</h1>" || got["posts/c.html"] != "c" {
		t.Fatalf("tree = %v", got)
	}
	if f.count("PATCH /repos/Maria/blog/git/refs/heads/gh-pages") != 1 {
		t.Fatal("expected ref update")
	}

	// Third publish, nothing changed: no blobs, no commit.
	commits := f.count("POST /repos/Maria/blog/git/commits")
	res, err = c.Publish(ctx, "Maria", "blog", dir, PublishOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Unchanged || res.Uploaded != 0 || f.count("POST /repos/Maria/blog/git/commits") != commits {
		t.Fatalf("expected unchanged publish, got %+v", res)
	}
}

func TestPublishInlineTextAndChunking(t *testing.T) {
	ctx := context.Background()
	f := newFake(t)
	rp := f.addRepo("Maria", "site")
	rp.autoInit()
	c := f.client()
	c.TreeChunkEntries = 2
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"a.html": "<p>á</p>", "b.css": "body{}", "c/d.js": "x()", "bin.dat": "\x00\x01", "e.txt": "e",
	})
	if _, err := c.Publish(ctx, "Maria", "site", dir, PublishOptions{}, nil); err != nil {
		t.Fatal(err)
	}
	// Only bin.dat and the empty .nojekyll go as blobs; text goes inline.
	if n := f.count("POST /repos/Maria/site/git/blobs"); n != 2 {
		t.Fatalf("blob uploads = %d, want 2", n)
	}
	if !slices.Contains(f.inlined, "a.html") || slices.Contains(f.inlined, "bin.dat") {
		t.Fatalf("inlined = %v", f.inlined)
	}
	// 6 entries in chunks of 2 = 3 tree requests chained via base_tree.
	if n := f.count("POST /repos/Maria/site/git/trees"); n != 3 {
		t.Fatalf("tree requests = %d, want 3", n)
	}
	got := rp.files("gh-pages")
	if len(got) != 6 || got["a.html"] != "<p>á</p>" || got["bin.dat"] != "\x00\x01" {
		t.Fatalf("tree = %v", got)
	}
}

func TestPublishTruncatedTreeAndEmptyRepo(t *testing.T) {
	ctx := context.Background()
	f := newFake(t)
	rp := f.addRepo("Maria", "big")
	rp.empty = true // created without auto_init
	c := f.client()
	c.InlineTextLimit = -1
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{"x/y/z.html": "z", "index.html": "i"})
	if _, err := c.Publish(ctx, "Maria", "big", dir, PublishOptions{}, nil); err != nil {
		t.Fatal(err)
	}
	if f.count("PUT /repos/Maria/big/contents/README.md") != 1 {
		t.Fatal("empty repo should be initialized")
	}
	rp.forceTrunc = true
	writeTree(t, dir, map[string]string{"x/y/z.html": "z", "index.html": "i2"})
	before := f.count("POST /repos/Maria/big/git/blobs")
	res, err := c.Publish(ctx, "Maria", "big", dir, PublishOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n := f.count("POST /repos/Maria/big/git/blobs") - before; n != 1 || res.Skipped != 2 {
		t.Fatalf("after truncated tree walk uploaded %d, result %+v", n, res)
	}
}

func TestPublishRetriesSecondaryRateLimit(t *testing.T) {
	ctx := context.Background()
	f := newFake(t)
	rp := f.addRepo("Maria", "rl")
	rp.autoInit()
	c := f.client()
	c.InlineTextLimit = -1
	c.Concurrency = 1 // one request at a time so the retry sequence is fixed
	var slept []time.Duration
	var smu sync.Mutex
	c.sleep = func(ctx context.Context, d time.Duration) error {
		smu.Lock()
		slept = append(slept, d)
		smu.Unlock()
		return nil
	}
	limited := 0
	f.hook = func(w http.ResponseWriter, r *http.Request) bool {
		if strings.HasSuffix(r.URL.Path, "/git/blobs") && limited < 2 {
			limited++
			if limited == 1 {
				w.Header().Set("Retry-After", "7")
				msg(w, 403, "You have exceeded a secondary rate limit.")
			} else {
				msg(w, 429, "Too Many Requests")
			}
			return true
		}
		return false
	}
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{"index.html": "i"})
	if _, err := c.Publish(ctx, "Maria", "rl", dir, PublishOptions{}, nil); err != nil {
		t.Fatal(err)
	}
	// Same request: attempt 0 gets 403+Retry-After (wait 7s), attempt 1 gets
	// a bare 429 (secondary limit without hints: 1m << attempt = 2m).
	if want := []time.Duration{7 * time.Second, 2 * time.Minute}; !slices.Equal(slept, want) {
		t.Fatalf("slept = %v, want %v", slept, want)
	}

	// Persistent limiting ends with ErrRateLimited.
	f.hook = func(w http.ResponseWriter, r *http.Request) bool {
		if strings.HasSuffix(r.URL.Path, "/git/blobs") {
			msg(w, 429, "Too Many Requests")
			return true
		}
		return false
	}
	c.MaxRetries = 2
	writeTree(t, dir, map[string]string{"index.html": "new content"})
	if _, err := c.Publish(ctx, "Maria", "rl", dir, PublishOptions{}, nil); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("want ErrRateLimited, got %v", err)
	}
}

func TestPublishErrors(t *testing.T) {
	f := newFake(t)
	c := f.client()
	if _, err := c.Publish(context.Background(), "Maria", "x", t.TempDir(), PublishOptions{}, nil); !errors.Is(err, ErrNothingToPublish) {
		t.Fatalf("want ErrNothingToPublish, got %v", err)
	}
}

func TestEnsurePages(t *testing.T) {
	ctx := context.Background()
	f := newFake(t)
	rp := f.addRepo("Maria", "blog")
	c := f.client()

	u, err := c.EnsurePages(ctx, "Maria", "blog", "")
	if err != nil || u != "https://maria.github.io/blog/" {
		t.Fatalf("EnsurePages = %q, %v", u, err)
	}
	if f.count("POST /repos/Maria/blog/pages") != 1 || rp.pages.Source.Branch != "gh-pages" {
		t.Fatal("expected Pages to be created on gh-pages")
	}
	// Already enabled with same source: no write.
	if _, err := c.EnsurePages(ctx, "Maria", "blog", "gh-pages"); err != nil {
		t.Fatal(err)
	}
	if f.count("PUT /repos/Maria/blog/pages") != 0 {
		t.Fatal("unexpected update")
	}
	// Different branch: update via PUT.
	if _, err := c.EnsurePages(ctx, "Maria", "blog", "site"); err != nil {
		t.Fatal(err)
	}
	if f.count("PUT /repos/Maria/blog/pages") != 1 || rp.pages.Source.Branch != "site" {
		t.Fatal("expected Pages update")
	}

	priv := f.addRepo("Maria", "secret")
	priv.private, priv.pagesUnavail = true, true
	if _, err := c.EnsurePages(ctx, "Maria", "secret", ""); !errors.Is(err, ErrPagesUnavailable) {
		t.Fatalf("want ErrPagesUnavailable, got %v", err)
	}

	f.hook = func(w http.ResponseWriter, r *http.Request) bool {
		if strings.HasSuffix(r.URL.Path, "/pages") && r.Method == http.MethodPost {
			msg(w, 403, "Resource not accessible by personal access token")
			return true
		}
		return false
	}
	f.addRepo("Maria", "noperm")
	if _, err := c.EnsurePages(ctx, "Maria", "noperm", ""); !errors.Is(err, ErrPermission) {
		t.Fatalf("want ErrPermission, got %v", err)
	}
}

func TestWaitBuild(t *testing.T) {
	ctx := context.Background()
	f := newFake(t)
	rp := f.addRepo("Maria", "blog")
	c := f.client()
	rp.builds = []map[string]any{
		{"status": "built", "commit": "old"},
		{"status": "building", "commit": "abc"},
		{"status": "built", "commit": "abc", "duration": 1500, "error": map[string]any{"message": nil}},
	}
	b, err := c.WaitBuild(ctx, "Maria", "blog", "abc", time.Minute)
	if err != nil || b.Status != "built" || b.Commit != "abc" {
		t.Fatalf("WaitBuild = %+v, %v", b, err)
	}
	rp.builds = []map[string]any{{"status": "errored", "commit": "def", "error": map[string]any{"message": "Page build failed."}}}
	if _, err := c.WaitBuild(ctx, "Maria", "blog", "", time.Minute); !errors.Is(err, ErrPagesBuildFailed) {
		t.Fatalf("want ErrPagesBuildFailed, got %v", err)
	}
}

func TestDeviceFlow(t *testing.T) {
	var mu sync.Mutex
	polls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("Accept") != "application/json" {
			http.Error(w, "need json", 400)
			return
		}
		r.ParseForm()
		switch r.URL.Path {
		case "/login/device/code":
			if r.Form.Get("client_id") != "Iv1.abc" || r.Form.Get("scope") != "repo" {
				writeJSON(w, 200, map[string]any{"error": "incorrect_client_credentials"})
				return
			}
			writeJSON(w, 200, map[string]any{
				"device_code": "dev123", "user_code": "WDJB-MJHT",
				"verification_uri": "https://github.com/login/device", "expires_in": 900, "interval": 5,
			})
		case "/login/oauth/access_token":
			if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" || r.Form.Get("device_code") != "dev123" {
				writeJSON(w, 200, map[string]any{"error": "unsupported_grant_type"})
				return
			}
			polls++
			switch polls {
			case 1:
				writeJSON(w, 200, map[string]any{"error": "authorization_pending"})
			case 2:
				writeJSON(w, 200, map[string]any{"error": "slow_down", "interval": 10})
			case 3:
				writeJSON(w, 200, map[string]any{"error": "slow_down"})
			default:
				writeJSON(w, 200, map[string]any{"access_token": "gho_x", "token_type": "bearer", "scope": "repo"})
			}
		}
	}))
	defer srv.Close()

	var waits []time.Duration
	df := &DeviceFlow{ClientID: "Iv1.abc", BaseURL: srv.URL, sleep: func(ctx context.Context, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}}
	dc, err := df.Start(context.Background())
	if err != nil || dc.UserCode != "WDJB-MJHT" || dc.Interval != 5 {
		t.Fatalf("Start = %+v, %v", dc, err)
	}
	tok, err := df.Poll(context.Background(), dc)
	if err != nil || tok.AccessToken != "gho_x" {
		t.Fatalf("Poll = %+v, %v", tok, err)
	}
	want := []time.Duration{5 * time.Second, 5 * time.Second, 10 * time.Second, 15 * time.Second}
	if !slices.Equal(waits, want) {
		t.Fatalf("waits = %v, want %v", waits, want)
	}

	// Terminal errors.
	for code, want := range map[string]error{
		"expired_token":        ErrDeviceExpired,
		"access_denied":        ErrAccessDenied,
		"device_flow_disabled": ErrDeviceFlowDisabled,
	} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 200, map[string]any{"error": code})
		}))
		df := &DeviceFlow{ClientID: "x", BaseURL: s.URL, sleep: func(context.Context, time.Duration) error { return nil }}
		_, err := df.Poll(context.Background(), &DeviceCode{DeviceCode: "d", Interval: 1, ExpiresIn: 60})
		s.Close()
		if !errors.Is(err, want) {
			t.Errorf("%s: got %v", code, err)
		}
	}
	// Expiry without answer.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"error": "authorization_pending"})
	}))
	defer s.Close()
	df = &DeviceFlow{ClientID: "x", BaseURL: s.URL, sleep: func(context.Context, time.Duration) error { return nil }}
	if _, err := df.Poll(context.Background(), &DeviceCode{DeviceCode: "d", Interval: 5, ExpiresIn: 20}); !errors.Is(err, ErrDeviceExpired) {
		t.Fatalf("want ErrDeviceExpired, got %v", err)
	}
}

func TestEnsureRepoPrivateAndGitData(t *testing.T) {
	ctx := context.Background()
	f := newFake(t)
	c := f.client()

	r, err := c.EnsureRepoWith(ctx, "", "notas", EnsureRepoOptions{Private: true, Empty: true, Description: "Notes"})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Created || !r.Private {
		t.Fatalf("repo = %+v", r)
	}
	if f.created["private"] != true || f.created["auto_init"] != false || f.created["homepage"] != nil || f.created["description"] != "Notes" {
		t.Fatalf("create body = %v", f.created)
	}
	// Public default still sets homepage.
	if _, err := c.EnsureRepo(ctx, "", "site"); err != nil {
		t.Fatal(err)
	}
	if f.created["private"] != false || f.created["homepage"] == nil {
		t.Fatalf("public create body = %v", f.created)
	}

	// Empty repo without init: reported as missing branch, nothing written.
	tip, err := c.BranchTip(ctx, "Maria", "notas", "main", nil)
	if err != nil || tip.Commit != "" {
		t.Fatalf("tip = %+v, %v", tip, err)
	}
	if f.count("PUT ") != 0 {
		t.Fatal("BranchTip(nil) must not write")
	}
	tip, err = c.BranchTip(ctx, "Maria", "notas", "main", &InitFile{Path: ".blup/README.md", Content: []byte("hi\n")})
	if err != nil || tip.Commit == "" || tip.Tree == "" {
		t.Fatalf("tip = %+v, %v", tip, err)
	}
	files, err := c.ReadTree(ctx, "Maria", "notas", tip.Tree)
	if err != nil || len(files) != 1 || files[0].Path != ".blup/README.md" || files[0].Mode != "100644" {
		t.Fatalf("tree = %+v, %v", files, err)
	}

	bin := []byte{0, 1, 2, 0xff}
	sha, err := c.CreateBlob(ctx, "Maria", "notas", bin)
	if err != nil || sha != BlobSHA(bin) {
		t.Fatal(sha, err)
	}
	got, err := c.GetBlob(ctx, "Maria", "notas", sha)
	if err != nil || string(got) != string(bin) {
		t.Fatal(got, err)
	}
	if c.CanInline(bin) || !c.CanInline([]byte("olá")) || c.CanInline(nil) {
		t.Fatal("CanInline")
	}
	tree, err := c.CreateTree(ctx, "Maria", "notas", tip.Tree, []TreeChange{
		{Path: "a.md", Content: []byte("# A\n")},
		{Path: "img/x.bin", SHA: sha},
		{Path: ".blup/README.md", Delete: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	commit, err := c.CreateCommit(ctx, "Maria", "notas", "sync", tree, []string{tip.Commit})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateBranch(ctx, "Maria", "notas", "main", commit, false); err != nil {
		t.Fatal(err)
	}
	if got := f.repos["maria/notas"].files("main"); len(got) != 2 || got["a.md"] != "# A\n" {
		t.Fatalf("files = %v", got)
	}
	// A sibling commit is not a fast-forward.
	other, _ := c.CreateCommit(ctx, "Maria", "notas", "other", tip.Tree, []string{tip.Commit})
	if err := c.UpdateBranch(ctx, "Maria", "notas", "main", other, false); !errors.Is(err, ErrNotFastForward) {
		t.Fatalf("want ErrNotFastForward, got %v", err)
	}
	if err := c.UpdateBranch(ctx, "Maria", "notas", "main", other, true); !errors.Is(err, ErrNotFastForward) {
		t.Fatalf("create existing: want ErrNotFastForward, got %v", err)
	}
	if err := c.UpdateBranch(ctx, "Maria", "notas", "dev", other, true); err != nil {
		t.Fatal(err)
	}
}
