package cfpages

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
		"index.html": "text/html", "a/b.CSS": "text/css", "x.js": "application/javascript",
		"f.woff2": "font/woff2", "i.svg": "image/svg+xml", "noext": "application/octet-stream",
		".htaccess": "application/octet-stream", "index.xml": "application/xml",
	} {
		if got := contentTypeFor(in); got != want {
			t.Errorf("contentTypeFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProjectNames(t *testing.T) {
	for in, want := range map[string]string{
		"Meu Blog de Tiro":        "meu-blog-de-tiro",
		"  Ação & Reação!! 2026 ": "acao-e-reacao-2026",
		"___":                     "site",
		"Über Café":               "uber-cafe",
		strings.Repeat("ab ", 40): strings.TrimRight(strings.Repeat("ab-", 20)[:58], "-"),
	} {
		got := ProjectNameFromTitle(in)
		if got != want {
			t.Errorf("ProjectNameFromTitle(%q) = %q, want %q", in, got, want)
		}
		if err := ValidateProjectName(got); err != nil {
			t.Errorf("normalized name %q invalid: %v", got, err)
		}
	}
	for _, bad := range []string{"", "-a", "a-", "A", "a_b", strings.Repeat("a", 59)} {
		if err := ValidateProjectName(bad); !errors.Is(err, ErrInvalidProjectName) {
			t.Errorf("ValidateProjectName(%q) = %v, want ErrInvalidProjectName", bad, err)
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
	if len(perms) != 1 || perms[0]["key"] != "page" || perms[0]["type"] != "edit" ||
		q.Get("accountId") != "*" || q.Get("zoneId") != "all" || q.Get("name") != "blup" {
		t.Errorf("template = %v %v", perms, q)
	}
}

func TestMakeBuckets(t *testing.T) {
	var files []fileEntry
	for i := 0; i < 4500; i++ {
		files = append(files, fileEntry{name: fmt.Sprint(i), size: 100})
	}
	files = append(files, fileEntry{name: "big1", size: 30 << 20}, fileEntry{name: "big2", size: 30 << 20}, fileEntry{name: "big3", size: 30 << 20}, fileEntry{name: "big4", size: 20 << 20})
	bs := makeBuckets(files)
	n := 0
	for _, b := range bs {
		var sz int64
		for _, f := range b.files {
			sz += f.size
		}
		if len(b.files) > maxBucketFiles || sz > maxBucketSize {
			t.Fatalf("bucket over limits: %d files, %d bytes", len(b.files), sz)
		}
		n += len(b.files)
	}
	if n != len(files) {
		t.Fatalf("bucketed %d files, want %d", n, len(files))
	}
	if len(bs) < 4 {
		t.Fatalf("expected an extra bucket for the 4th big file, got %d", len(bs))
	}
}

// fakeCF is an in-memory Cloudflare API.
type fakeCF struct {
	t     *testing.T
	mu    sync.Mutex
	token string

	projects      map[string]apiProject
	stored        map[string]bool
	uploaded      map[string]uploadItem
	uploadCalls   int
	maxBucketSeen int
	upserted      []string
	tokenCalls    int
	expireFirst   bool // first upload with first JWT returns 8000013
	rateLimitOnce bool // first upload returns 429
	manifest      map[string]string
	branch        string
	headers       string
	redirects     string
	createdNames  []string
}

func newFake(t *testing.T) (*fakeCF, *httptest.Server) {
	f := &fakeCF{t: t, token: "good-token", projects: map[string]apiProject{}, stored: map[string]bool{}, uploaded: map[string]uploadItem{}}
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

func fakeJWT(n int, exp int64) string {
	enc := func(v any) string { b, _ := json.Marshal(v); return base64.RawURLEncoding.EncodeToString(b) }
	return enc(map[string]any{"alg": "none"}) + "." + enc(map[string]any{"exp": exp, "n": n, "max_file_count_allowed": 20000}) + ".sig"
}

func (f *fakeCF) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	p := r.URL.Path
	if strings.HasPrefix(p, "/pages/assets/") {
		if !strings.Contains(auth, ".") {
			fail(w, 401, codePagesUnauthorized, "bad jwt")
			return
		}
	} else if auth != f.token {
		fail(w, 401, 1000, "Invalid API Token")
		return
	}
	switch {
	case p == "/user/tokens/verify":
		ok(w, map[string]any{"id": "tok", "status": "active"})
	case p == "/accounts":
		ok(w, []Account{{ID: "acc1", Name: "Gabriell"}})
	case p == "/accounts/acc1/pages/projects" && r.Method == http.MethodPost:
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["production_branch"] != "main" {
			f.t.Errorf("production_branch = %q", body["production_branch"])
		}
		f.createdNames = append(f.createdNames, body["name"])
		if body["name"] == "taken" {
			fail(w, 409, 8000002, "A project with this name already exists")
			return
		}
		pr := apiProject{Name: body["name"], Subdomain: body["name"] + "-x7k.pages.dev", ProductionBranch: "main"}
		pr.Domains = []string{pr.Subdomain}
		f.projects[pr.Name] = pr
		ok(w, pr)
	case strings.HasPrefix(p, "/accounts/acc1/pages/projects/") && strings.HasSuffix(p, "/upload-token"):
		f.tokenCalls++
		ok(w, map[string]string{"jwt": fakeJWT(f.tokenCalls, time.Now().Add(time.Hour).Unix())})
	case strings.HasPrefix(p, "/accounts/acc1/pages/projects/") && strings.HasSuffix(p, "/deployments"):
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			f.t.Errorf("multipart: %v", err)
		}
		f.manifest = nil
		if err := json.Unmarshal([]byte(r.FormValue("manifest")), &f.manifest); err != nil {
			f.t.Errorf("manifest: %v", err)
		}
		f.branch = r.FormValue("branch")
		read := func(name string) string {
			file, _, err := r.FormFile(name)
			if err != nil {
				return ""
			}
			b, _ := io.ReadAll(file)
			return string(b)
		}
		f.headers, f.redirects = read("_headers"), read("_redirects")
		ok(w, map[string]string{"id": "dep-123", "url": "https://abc123.blog-x7k.pages.dev"})
	case strings.HasPrefix(p, "/accounts/acc1/pages/projects/") && r.Method == http.MethodGet:
		name := strings.TrimPrefix(p, "/accounts/acc1/pages/projects/")
		pr, found := f.projects[name]
		if !found {
			fail(w, 404, codePagesNotFound, "Project not found")
			return
		}
		ok(w, pr)
	case p == "/pages/assets/check-missing":
		var body struct{ Hashes []string }
		json.NewDecoder(r.Body).Decode(&body)
		missing := []string{}
		for _, h := range body.Hashes {
			if !f.stored[h] {
				missing = append(missing, h)
			}
		}
		ok(w, missing)
	case p == "/pages/assets/upload":
		f.uploadCalls++
		if f.expireFirst && jwtClaims(auth)["n"].(float64) == 1 {
			fail(w, 401, codePagesUnauthorized, "Unauthorized")
			return
		}
		if f.rateLimitOnce {
			f.rateLimitOnce = false
			fail(w, 429, 971, "Please wait and consider throttling your request speed")
			return
		}
		var items []uploadItem
		if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
			f.t.Errorf("upload body: %v", err)
		}
		if len(items) > maxBucketFiles {
			f.t.Errorf("bucket with %d files", len(items))
		}
		f.maxBucketSeen = max(f.maxBucketSeen, len(items))
		for _, it := range items {
			if !it.Base64 || it.Metadata["contentType"] == "" {
				f.t.Errorf("bad upload item %+v", it.Key)
			}
			if _, err := base64.StdEncoding.DecodeString(it.Value); err != nil {
				f.t.Errorf("item %s not base64", it.Key)
			}
			f.uploaded[it.Key] = it
			f.stored[it.Key] = true
		}
		ok(w, nil)
	case p == "/pages/assets/upsert-hashes":
		var body struct{ Hashes []string }
		json.NewDecoder(r.Body).Decode(&body)
		f.upserted = body.Hashes
		ok(w, true)
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
		"_headers":                "/*\n  X-Frame-Options: DENY\n",
		"_redirects":              "/old /new 301\n",
		".DS_Store":               "junk",
		"posts/.DS_Store":         "junk",
		"node_modules/x/index.js": "junk",
		".git/HEAD":               "junk",
		"_worker.js":              "junk",
		"_routes.json":            "{}",
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
	f.expireFirst = true
	c := newClient(srv)
	ctx := context.Background()

	if err := c.VerifyToken(ctx); err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	accts, err := c.Accounts(ctx)
	if err != nil || len(accts) != 1 || accts[0].ID != "acc1" {
		t.Fatalf("Accounts = %v, %v", accts, err)
	}
	proj, err := c.EnsureProject(ctx, "acc1", "blog")
	if err != nil {
		t.Fatalf("EnsureProject: %v", err)
	}
	if proj.Subdomain != "blog-x7k.pages.dev" || proj.URL() != "https://blog-x7k.pages.dev/" {
		t.Fatalf("project = %+v", proj)
	}
	if _, err := c.EnsureProject(ctx, "acc1", "blog"); err != nil || len(f.createdNames) != 1 {
		t.Fatalf("EnsureProject again: %v, created %v", err, f.createdNames)
	}

	dir := writeSite(t)
	cssHash := HashFile("css/site.css", []byte("body{}"))
	f.stored[cssHash] = true

	var stages []string
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

	wantPaths := []string{"/css/site.css", "/dup.html", "/functions/index.html", "/index.html", "/posts/_headers", "/posts/a/index.html", "/posts/b/index.html"}
	var got []string
	for k := range f.manifest {
		got = append(got, k)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(wantPaths, ",") {
		t.Fatalf("manifest paths = %v, want %v", got, wantPaths)
	}
	if f.manifest["/index.html"] != HashFile("index.html", []byte("<h1>home</h1>")) || f.manifest["/dup.html"] != f.manifest["/index.html"] {
		t.Fatalf("manifest hashes wrong: %v", f.manifest)
	}
	if f.branch != "main" {
		t.Errorf("branch = %q", f.branch)
	}
	if !strings.Contains(f.headers, "X-Frame-Options") || f.redirects != "/old /new 301\n" {
		t.Errorf("_headers=%q _redirects=%q", f.headers, f.redirects)
	}
	if _, up := f.uploaded[cssHash]; up {
		t.Errorf("css was re-uploaded although check-missing skipped it")
	}
	if len(f.uploaded) != 5 {
		t.Errorf("uploaded %d unique hashes, want 5", len(f.uploaded))
	}
	if it := f.uploaded[f.manifest["/index.html"]]; it.Metadata["contentType"] != "text/html" {
		t.Errorf("contentType = %q", it.Metadata["contentType"])
	}
	if f.tokenCalls < 2 {
		t.Errorf("expected JWT refresh after 8000013, token calls = %d", f.tokenCalls)
	}
	if len(f.upserted) != 6 {
		t.Errorf("upserted %d hashes, want 6 unique", len(f.upserted))
	}
	if res.URL != "https://blog-x7k.pages.dev/" || res.DeploymentID != "dep-123" || res.DeploymentURL != "https://abc123.blog-x7k.pages.dev" {
		t.Errorf("result = %+v", res)
	}
	if res.Uploaded != 6 || res.Skipped != 1 {
		t.Errorf("uploaded/skipped = %d/%d, want 6/1", res.Uploaded, res.Skipped)
	}
	want := []string{StageScanning, StageChecking, StageUploading, StageFinishing, StageDeploying, StageDone}
	if strings.Join(stages, ",") != strings.Join(want, ",") {
		t.Errorf("stages = %v", stages)
	}

	before := f.uploadCalls
	res, err = c.Deploy(ctx, "acc1", "blog", dir, nil)
	if err != nil || res.Uploaded != 0 || res.Skipped != 7 || f.uploadCalls != before {
		t.Fatalf("redeploy: %+v %v (upload calls %d -> %d)", res, err, before, f.uploadCalls)
	}
}

// A Hugo build with a pagefind index: thousands of tiny files.
func TestPagefindManySmallFiles(t *testing.T) {
	f, srv := newFake(t)
	f.rateLimitOnce = true // a 429 mid-upload must be retried, not fatal
	f.projects["blog"] = apiProject{Name: "blog", Subdomain: "blog.pages.dev", ProductionBranch: "main"}
	dir := t.TempDir()
	files := map[string]string{"index.html": "<h1>home</h1>", "pagefind/pagefind.js": "export{}", "pagefind/pagefind-entry.json": "{}"}
	const n = 5000
	for i := 0; i < n; i++ {
		files[fmt.Sprintf("pagefind/fragment/pt_%05x.pf_fragment", i)] = fmt.Sprintf("frag-%d", i)
		if i%5 == 0 {
			files[fmt.Sprintf("pagefind/index/pt_%05x.pf_index", i)] = fmt.Sprintf("idx-%d", i)
		}
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
	if len(f.uploaded) != len(files) {
		t.Errorf("uploaded %d unique hashes, want %d", len(f.uploaded), len(files))
	}
	if f.maxBucketSeen > maxBucketFiles || f.uploadCalls < len(files)/maxBucketFiles+1 {
		t.Errorf("max bucket %d, upload calls %d", f.maxBucketSeen, f.uploadCalls)
	}
	if last.Stage != StageDone || last.Done != len(files) {
		t.Errorf("last progress = %+v", last)
	}
	if ct := f.uploaded[f.manifest["/pagefind/pagefind.js"]].Metadata["contentType"]; ct != "application/javascript" {
		t.Errorf("pagefind.js contentType = %q", ct)
	}
	if ct := f.uploaded[f.manifest["/pagefind/fragment/pt_00000.pf_fragment"]].Metadata["contentType"]; ct != "application/octet-stream" {
		t.Errorf("fragment contentType = %q", ct)
	}
}

func TestErrors(t *testing.T) {
	f, srv := newFake(t)
	ctx := context.Background()

	bad := newClient(srv)
	bad.Token = "nope"
	if err := bad.VerifyToken(ctx); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("VerifyToken bad = %v", err)
	}
	c := newClient(srv)
	if _, err := c.EnsureProject(ctx, "acc1", "taken"); !errors.Is(err, ErrProjectNameTaken) {
		t.Errorf("taken = %v", err)
	}
	if _, err := c.EnsureProject(ctx, "acc1", "Bad Name"); !errors.Is(err, ErrInvalidProjectName) {
		t.Errorf("bad name = %v", err)
	}
	if _, err := c.Deploy(ctx, "acc1", "missing", t.TempDir(), nil); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing project = %v", err)
	}

	perm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fail(w, 403, 10000, "Authentication error")
	}))
	defer perm.Close()
	if _, err := newClient(perm).EnsureProject(ctx, "acc1", "blog"); !errors.Is(err, ErrMissingPermission) {
		t.Errorf("perm = %v", err)
	}

	f.projects["blog"] = apiProject{Name: "blog", Subdomain: "blog.pages.dev", ProductionBranch: "main"}
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
	if !jwtExpired(fakeJWT(1, time.Now().Add(-time.Minute).Unix())) {
		t.Error("expired JWT not detected")
	}
	if jwtExpired(fakeJWT(1, time.Now().Add(time.Hour).Unix())) {
		t.Error("valid JWT reported expired")
	}
	if maxFilesFromJWT(fakeJWT(1, 0)) != 20000 || maxFilesFromJWT("garbage") != DefaultMaxFiles {
		t.Error("max_file_count_allowed")
	}
}
