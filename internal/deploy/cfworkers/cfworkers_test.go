package cfworkers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// Vectors computed with wrangler's own libraries (blake3-wasm@2.1.5 and
// Node's path.extname), i.e. hash(base64(content)+ext).hex().slice(0,32).
func TestHashFileVectors(t *testing.T) {
	cases := []struct {
		name    string
		content []byte
		want    string
	}{
		{"index.html", []byte("<h1>Olá</h1>\n"), "bffae2e436dfe908025f717d9782cff4"},
		{"css/style.css", []byte("body{color:red}"), "a1e60d295449511896a3679447c6cae3"},
		{"empty.txt", nil, "f9bc91770fa5e997cbd47fba833629fc"},
		{".htaccess", []byte("x"), "c89c27f28e24f706b72c408fe1d6537a"},
		{"a/..foo", []byte("y"), "7b79fbc9a4c2994ac362075b3aaf3213"},
		{"file.", []byte("z"), "d334b109b225b65fd7031960bee9e9a6"},
		{"img/logo.PNG", []byte{0, 1, 2, 255}, "77434014c4dec15b066f032a62942659"},
		{"noext", []byte("hello"), "324ea05bea4d7f75b8d9ed695e65b2ca"},
	}
	for _, c := range cases {
		if got := HashFile(c.name, c.content); got != c.want {
			t.Errorf("HashFile(%q) = %s, want %s", c.name, got, c.want)
		}
		// wrangler hashes the absolute path; the extension must not change.
		if got := HashFile("/abs/dir.v2/"+c.name, c.content); got != c.want {
			t.Errorf("HashFile(abs %q) = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestNodeExtname(t *testing.T) {
	for in, want := range map[string]string{
		"a.html": ".html", ".htaccess": "", "/x/.htaccess": "", "a/b.tar.gz": ".gz",
		"file.": ".", "..": "", "/x/..foo": ".foo", "noext": "", "dir.d/noext": "",
	} {
		if got := nodeExtname(in); got != want {
			t.Errorf("nodeExtname(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestContentType(t *testing.T) {
	for in, want := range map[string]string{
		"index.html": "text/html; charset=utf-8", "a/b.CSS": "text/css; charset=utf-8", "x.js": "application/javascript",
		"f.woff2": "font/woff2", "i.svg": "image/svg+xml", "noext": "application/null",
		".htaccess": "application/null", "index.xml": "application/xml", "p.pf_fragment": "application/null",
	} {
		if got := contentTypeFor(in); got != want {
			t.Errorf("contentTypeFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNames(t *testing.T) {
	for in, want := range map[string]string{
		"Meu Blog de Tiro":        "meu-blog-de-tiro",
		"  Ação & Reação!! 2026 ": "acao-e-reacao-2026",
		"___":                     "site",
		"Über Café":               "uber-cafe",
		strings.Repeat("ab ", 40): strings.TrimRight(strings.Repeat("ab-", 22)[:63], "-"),
	} {
		got := NameFromTitle(in)
		if got != want {
			t.Errorf("NameFromTitle(%q) = %q, want %q", in, got, want)
		}
		if err := ValidateName(got); err != nil {
			t.Errorf("normalized name %q invalid: %v", got, err)
		}
	}
	for _, bad := range []string{"", "-a", "a-", "A", "a_b", strings.Repeat("a", 64)} {
		if err := ValidateName(bad); !errors.Is(err, ErrInvalidName) {
			t.Errorf("ValidateName(%q) = %v, want ErrInvalidName", bad, err)
		}
	}
}

func TestTokenTemplateURL(t *testing.T) {
	u, err := url.Parse(TokenTemplateURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	var perms []map[string]string
	if err := json.Unmarshal([]byte(q.Get("permissionGroupKeys")), &perms); err != nil {
		t.Fatal(err)
	}
	if len(perms) != 1 || perms[0]["key"] != "workers_scripts" || perms[0]["type"] != "edit" ||
		q.Get("accountId") != "*" || q.Get("zoneId") != "all" || q.Get("name") != "blup" {
		t.Errorf("template = %v %v", perms, q)
	}
}

func TestURL(t *testing.T) {
	if got := URL("meu-blog", "ana"); got != "https://meu-blog.ana.workers.dev/" {
		t.Errorf("URL = %q", got)
	}
}

// fakeCF is an in-memory Cloudflare API (Workers static assets).
type fakeCF struct {
	t     *testing.T
	mu    sync.Mutex
	token string

	subdomain     string
	scripts       map[string]bool
	stored        map[string]string // hash -> content type
	sessions      int
	uploadCalls   int
	singleCalls   int
	maxBucketSeen int
	single        bool // upload JWT asks for one request per file
	rateLimitOnce bool // first upload returns 429
	manifest      map[string]manifestEntry
	meta          map[string]any
	devEnabled    map[string]bool
	apiDate       string
	pending       map[string]bool
}

func newFake(t *testing.T) (*fakeCF, *httptest.Server) {
	f := &fakeCF{t: t, token: "good-token", subdomain: "ana", scripts: map[string]bool{}, stored: map[string]string{}, devEnabled: map[string]bool{}}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	return f, srv
}

func ok(w http.ResponseWriter, result any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true, "errors": []any{}, "result": result})
}

func fail(w http.ResponseWriter, status, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []any{map[string]any{"code": code, "message": msg}}, "result": nil})
}

func fakeJWT(kind string, single bool) string {
	enc := func(v any) string { b, _ := json.Marshal(v); return base64.RawURLEncoding.EncodeToString(b) }
	claims := map[string]any{"kind": kind, "exp": time.Now().Add(time.Hour).Unix()}
	if single {
		claims["wrangler_single_asset_uploads"] = true
		claims["edge_kv_upload_concurrency"] = 4
	}
	return enc(map[string]any{"alg": "none"}) + "." + enc(claims) + ".sig"
}

func (f *fakeCF) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	p := r.URL.Path
	if strings.HasPrefix(p, "/accounts/acc1/workers/assets/upload") {
		if jwtClaims(auth)["kind"] != "upload" {
			fail(w, 401, 10001, "bad upload jwt")
			return
		}
	} else if auth != f.token {
		fail(w, 401, 1000, "Invalid API Token")
		return
	}
	const scripts = "/accounts/acc1/workers/scripts/"
	switch {
	case p == "/user/tokens/verify":
		ok(w, map[string]any{"id": "tok", "status": "active"})
	case p == "/accounts":
		ok(w, []Account{{ID: "acc1", Name: "Gabriell"}})
	case p == "/accounts/acc1/workers/subdomain" && r.Method == http.MethodGet:
		if f.subdomain == "" {
			fail(w, 404, 10007, "This subdomain does not exist")
			return
		}
		ok(w, map[string]string{"subdomain": f.subdomain})
	case p == "/accounts/acc1/workers/subdomain" && r.Method == http.MethodPut:
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["subdomain"] == "taken" {
			fail(w, 409, 10031, "Subdomain unavailable")
			return
		}
		f.subdomain = body["subdomain"]
		ok(w, map[string]string{"subdomain": f.subdomain})
	case strings.HasPrefix(p, scripts) && strings.HasSuffix(p, "/assets-upload-session"):
		f.sessions++
		var body struct{ Manifest map[string]manifestEntry }
		json.NewDecoder(r.Body).Decode(&body)
		f.manifest = body.Manifest
		f.pending = map[string]bool{}
		var missing []string
		seen := map[string]bool{}
		for _, e := range body.Manifest {
			if _, have := f.stored[e.Hash]; !have && !seen[e.Hash] {
				seen[e.Hash] = true
				missing = append(missing, e.Hash)
				f.pending[e.Hash] = true
			}
		}
		sort.Strings(missing)
		var buckets [][]string
		for i := 0; i < len(missing); i += 1000 {
			buckets = append(buckets, missing[i:min(i+1000, len(missing))])
		}
		if buckets == nil {
			buckets = [][]string{}
			ok(w, map[string]any{"jwt": fakeJWT("complete", false), "buckets": buckets})
			return
		}
		ok(w, map[string]any{"jwt": fakeJWT("upload", f.single), "buckets": buckets})
	case p == "/accounts/acc1/workers/assets/upload":
		f.uploadCalls++
		if f.rateLimitOnce {
			f.rateLimitOnce = false
			fail(w, 429, 971, "Please wait and consider throttling your request speed")
			return
		}
		if r.URL.Query().Get("base64") != "true" {
			f.t.Errorf("upload without base64=true")
		}
		mr, err := r.MultipartReader()
		if err != nil {
			f.t.Fatalf("multipart: %v", err)
		}
		n := 0
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				f.t.Fatalf("part: %v", err)
			}
			b, _ := io.ReadAll(part)
			if _, err := base64.StdEncoding.DecodeString(string(b)); err != nil {
				f.t.Errorf("part %s not base64", part.FormName())
			}
			f.stored[part.FormName()] = part.Header.Get("Content-Type")
			delete(f.pending, part.FormName())
			n++
		}
		f.maxBucketSeen = max(f.maxBucketSeen, n)
		if len(f.pending) == 0 {
			ok(w, map[string]string{"jwt": fakeJWT("complete", false)})
			return
		}
		ok(w, map[string]any{})
	case strings.HasPrefix(p, "/accounts/acc1/workers/assets/upload/"):
		f.singleCalls++
		h := strings.TrimPrefix(p, "/accounts/acc1/workers/assets/upload/")
		f.stored[h] = r.Header.Get("Content-Type")
		delete(f.pending, h)
		if len(f.pending) == 0 {
			ok(w, map[string]string{"jwt": fakeJWT("complete", false)})
			return
		}
		ok(w, map[string]any{})
	case strings.HasPrefix(p, scripts) && strings.HasSuffix(p, "/subdomain") && r.Method == http.MethodPost:
		f.apiDate = r.Header.Get("Cloudflare-Workers-Script-Api-Date")
		var body map[string]bool
		json.NewDecoder(r.Body).Decode(&body)
		if body["previews_enabled"] {
			f.t.Errorf("previews enabled")
		}
		f.devEnabled[strings.TrimSuffix(strings.TrimPrefix(p, scripts), "/subdomain")] = body["enabled"]
		ok(w, map[string]bool{"enabled": body["enabled"]})
	case strings.HasPrefix(p, scripts) && strings.HasSuffix(p, "/settings"):
		if !f.scripts[strings.TrimSuffix(strings.TrimPrefix(p, scripts), "/settings")] {
			fail(w, 404, 10007, "This Worker does not exist on your account.")
			return
		}
		ok(w, map[string]any{})
	case strings.HasPrefix(p, scripts) && r.Method == http.MethodPut:
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			f.t.Fatalf("multipart: %v", err)
		}
		f.meta = nil
		if err := json.Unmarshal([]byte(r.FormValue("metadata")), &f.meta); err != nil {
			f.t.Errorf("metadata: %v", err)
		}
		if len(r.MultipartForm.Value)+len(r.MultipartForm.File) != 1 {
			f.t.Errorf("assets-only upload with extra parts: %v %v", r.MultipartForm.Value, r.MultipartForm.File)
		}
		jwt, _ := f.meta["assets"].(map[string]any)["jwt"].(string)
		if jwtClaims(jwt)["kind"] != "complete" {
			fail(w, 400, 10021, "assets jwt is not a completion token")
			return
		}
		f.scripts[strings.TrimPrefix(p, scripts)] = true
		ok(w, map[string]any{"id": strings.TrimPrefix(p, scripts), "deployment_id": "ver-123"})
	case strings.HasPrefix(p, scripts) && r.Method == http.MethodDelete:
		name := strings.TrimPrefix(p, scripts)
		if !f.scripts[name] {
			fail(w, 404, 10007, "This Worker does not exist on your account.")
			return
		}
		delete(f.scripts, name)
		ok(w, nil)
	default:
		f.t.Errorf("unexpected %s %s", r.Method, p)
		fail(w, 404, 7003, "no route")
	}
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, c := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func writeSite(t *testing.T) string {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"index.html":              "<h1>home</h1>",
		"posts/a/index.html":      "<h1>a</h1>",
		"posts/b/index.html":      "<h1>b</h1>",
		"css/site.css":            "body{}",
		"dup.html":                "<h1>home</h1>", // same hash as index.html
		"functions/index.html":    "hugo section named functions",
		"404.html":                "<h1>not here</h1>",
		"_headers":                "/*\n  X-Frame-Options: DENY\n",
		"_redirects":              "/old /new 301\n",
		".DS_Store":               "junk",
		"posts/.DS_Store":         "junk",
		"node_modules/x/index.js": "junk",
		".git/HEAD":               "junk",
		"_worker.js":              "junk",
		".assetsignore":           "junk",
		"posts/_headers":          "kept: only root _headers is special",
	})
	return dir
}

func newClient(srv *httptest.Server) *Client {
	return &Client{Token: "good-token", BaseURL: srv.URL, HTTP: srv.Client(),
		sleep: func(ctx context.Context, d time.Duration) error { return ctx.Err() }}
}

func TestEndToEnd(t *testing.T) {
	f, srv := newFake(t)
	c := newClient(srv)
	ctx := context.Background()

	if err := c.VerifyToken(ctx); err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	accts, err := c.Accounts(ctx)
	if err != nil || len(accts) != 1 || accts[0].ID != "acc1" {
		t.Fatalf("Accounts = %v, %v", accts, err)
	}
	if sub, err := c.Subdomain(ctx, "acc1"); err != nil || sub != "ana" {
		t.Fatalf("Subdomain = %q, %v", sub, err)
	}
	if exists, err := c.Exists(ctx, "acc1", "blog"); err != nil || exists {
		t.Fatalf("Exists before = %v, %v", exists, err)
	}

	dir := writeSite(t)
	cssHash := HashFile("css/site.css", []byte("body{}"))
	f.stored[cssHash] = "text/css; charset=utf-8"

	var stages []Stage
	res, err := c.Deploy(ctx, "acc1", "blog", dir, func(p Progress) {
		if len(stages) == 0 || stages[len(stages)-1] != p.Stage {
			stages = append(stages, p.Stage)
		}
		if p.Done > p.Total && p.Total > 0 {
			t.Errorf("progress overflow %+v", p)
		}
	})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	wantPaths := []string{"/404.html", "/css/site.css", "/dup.html", "/functions/index.html", "/index.html", "/posts/_headers", "/posts/a/index.html", "/posts/b/index.html"}
	var got []string
	for k := range f.manifest {
		got = append(got, k)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(wantPaths, ",") {
		t.Fatalf("manifest paths = %v, want %v", got, wantPaths)
	}
	home := f.manifest["/index.html"]
	if home.Hash != HashFile("index.html", []byte("<h1>home</h1>")) || home.Size != int64(len("<h1>home</h1>")) || f.manifest["/dup.html"] != home {
		t.Fatalf("manifest entries wrong: %v", f.manifest)
	}
	if ct := f.stored[home.Hash]; ct != "text/html; charset=utf-8" {
		t.Errorf("index.html content type = %q", ct)
	}

	assets := f.meta["assets"].(map[string]any)
	cfg := assets["config"].(map[string]any)
	if cfg["html_handling"] != "auto-trailing-slash" || cfg["not_found_handling"] != "404-page" {
		t.Errorf("assets config = %v", cfg)
	}
	if !strings.Contains(cfg["_headers"].(string), "X-Frame-Options") || cfg["_redirects"] != "/old /new 301\n" {
		t.Errorf("_headers/_redirects = %v", cfg)
	}
	if _, has := f.meta["main_module"]; has || f.meta["compatibility_date"] != CompatibilityDate {
		t.Errorf("metadata = %v", f.meta)
	}
	if !f.devEnabled["blog"] || f.apiDate != scriptAPIDate {
		t.Errorf("workers.dev enabled = %v, api date %q", f.devEnabled, f.apiDate)
	}
	if res.VersionID != "ver-123" || res.Uploaded != 7 || res.Skipped != 1 {
		t.Errorf("result = %+v, want 7 uploaded and 1 skipped", res)
	}
	want := []Stage{StageScanning, StageChecking, StageUploading, StageDeploying, StageDone}
	if !slices.Equal(stages, want) {
		t.Errorf("stages = %v", stages)
	}
	if exists, err := c.Exists(ctx, "acc1", "blog"); err != nil || !exists {
		t.Fatalf("Exists after = %v, %v", exists, err)
	}

	// Again with nothing new: the session token completes it, no upload.
	before := f.uploadCalls
	res, err = c.Deploy(ctx, "acc1", "blog", dir, nil)
	if err != nil || res.Uploaded != 0 || res.Skipped != 8 || f.uploadCalls != before {
		t.Fatalf("redeploy: %+v %v (upload calls %d -> %d)", res, err, before, f.uploadCalls)
	}

	if err := c.Delete(ctx, "acc1", "blog"); err != nil || f.scripts["blog"] {
		t.Fatalf("Delete: %v", err)
	}
	if err := c.Delete(ctx, "acc1", "blog"); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}

// A Hugo build with a pagefind index: thousands of tiny files.
func TestPagefindManySmallFiles(t *testing.T) {
	f, srv := newFake(t)
	f.rateLimitOnce = true // a 429 mid-upload must be retried, not fatal
	dir := t.TempDir()
	files := map[string]string{"index.html": "<h1>home</h1>", "pagefind/pagefind.js": "export{}", "pagefind/pagefind-entry.json": "{}"}
	const n = 5000
	for i := 0; i < n; i++ {
		files[fmt.Sprintf("pagefind/fragment/pt_%05x.pf_fragment", i)] = fmt.Sprintf("frag-%d", i)
	}
	writeFiles(t, dir, files)

	var last Progress
	res, err := newClient(srv).Deploy(context.Background(), "acc1", "blog", dir, func(p Progress) { last = p })
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(f.manifest) != len(files) || res.Uploaded != len(files) || res.Skipped != 0 {
		t.Fatalf("manifest %d, result %+v, want %d files", len(f.manifest), res, len(files))
	}
	if f.maxBucketSeen > 1000 || f.uploadCalls < len(files)/1000+1 {
		t.Errorf("max bucket %d, upload calls %d", f.maxBucketSeen, f.uploadCalls)
	}
	if last.Stage != StageDone || last.Done != len(files) {
		t.Errorf("last progress = %+v", last)
	}
	if ct := f.stored[f.manifest["/pagefind/pagefind.js"].Hash]; ct != "application/javascript" {
		t.Errorf("pagefind.js content type = %q", ct)
	}
	if ct := f.stored[f.manifest["/pagefind/fragment/pt_00000.pf_fragment"].Hash]; ct != "application/null" {
		t.Errorf("fragment content type = %q", ct)
	}
}

// When the upload JWT asks for it, each file goes in its own request.
func TestSingleFileUploads(t *testing.T) {
	f, srv := newFake(t)
	f.single = true
	res, err := newClient(srv).Deploy(context.Background(), "acc1", "blog", writeSite(t), nil)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if f.singleCalls != 7 || f.uploadCalls != 0 || res.Uploaded != 8 {
		t.Errorf("single calls %d, bulk calls %d, result %+v", f.singleCalls, f.uploadCalls, res)
	}
}

func TestSubdomain(t *testing.T) {
	f, srv := newFake(t)
	f.subdomain = ""
	c := newClient(srv)
	ctx := context.Background()
	if _, err := c.Subdomain(ctx, "acc1"); !errors.Is(err, ErrNoSubdomain) {
		t.Fatalf("no subdomain = %v", err)
	}
	if _, err := c.RegisterSubdomain(ctx, "acc1", "taken"); !errors.Is(err, ErrSubdomainTaken) {
		t.Errorf("taken = %v", err)
	}
	if _, err := c.RegisterSubdomain(ctx, "acc1", "Bad Name"); !errors.Is(err, ErrInvalidName) {
		t.Errorf("bad = %v", err)
	}
	if sub, err := c.RegisterSubdomain(ctx, "acc1", "ana-blog"); err != nil || sub != "ana-blog" {
		t.Fatalf("register = %q, %v", sub, err)
	}
	if sub, err := c.Subdomain(ctx, "acc1"); err != nil || sub != "ana-blog" {
		t.Fatalf("after register = %q, %v", sub, err)
	}
}

func TestErrors(t *testing.T) {
	_, srv := newFake(t)
	ctx := context.Background()

	bad := newClient(srv)
	bad.Token = "nope"
	if err := bad.VerifyToken(ctx); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("VerifyToken bad = %v", err)
	}
	c := newClient(srv)
	if _, err := c.Deploy(ctx, "acc1", "Bad Name", t.TempDir(), nil); !errors.Is(err, ErrInvalidName) {
		t.Errorf("bad name = %v", err)
	}

	perm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fail(w, 403, 10000, "Authentication error")
	}))
	defer perm.Close()
	if _, err := newClient(perm).Exists(ctx, "acc1", "blog"); !errors.Is(err, ErrMissingPermission) {
		t.Errorf("perm = %v", err)
	}

	dir := t.TempDir()
	big, _ := os.Create(filepath.Join(dir, "huge.bin"))
	big.Truncate(MaxFileSize + 1)
	big.Close()
	if _, err := c.Deploy(ctx, "acc1", "blog", dir, nil); !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("too large = %v", err)
	}

	dir2 := t.TempDir()
	for i := 0; i < 4; i++ {
		os.WriteFile(filepath.Join(dir2, fmt.Sprintf("%d.html", i)), []byte("x"), 0o644)
	}
	if _, err := scan(dir2, 3, nil); !errors.Is(err, ErrTooManyFiles) || !errors.Is(err, ErrQuota) {
		t.Errorf("too many = %v", err)
	}
}

func TestJWTClaims(t *testing.T) {
	if !singleUploads(fakeJWT("upload", true)) || singleUploads(fakeJWT("upload", false)) || singleUploads("garbage") {
		t.Error("wrangler_single_asset_uploads")
	}
	if singleConcurrencyFrom(fakeJWT("upload", true)) != 4 || singleConcurrencyFrom("garbage") != singleConcurrency {
		t.Error("edge_kv_upload_concurrency")
	}
}

// TestScanSkipsSymlinks: a symlink in the built site must never publish
// the file it points to (it may be outside the site, like ~/.ssh).
func TestScanSkipsSymlinks(t *testing.T) {
	outside := t.TempDir()
	secret := filepath.Join(outside, "id_rsa")
	if err := os.WriteFile(secret, []byte("PRIVATE KEY"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>oi</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(dir, "key.txt")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "fora")); err != nil {
		t.Fatal(err)
	}
	files, err := scan(dir, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].name != "index.html" {
		t.Fatalf("want only index.html, got %+v", files)
	}
}
