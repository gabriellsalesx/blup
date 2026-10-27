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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// Vectors computed with wrangler's own libraries (blake3-wasm@2.1.5 and
// Node's path.extname): hash(base64(content)+ext).hex().slice(0,32).
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
		"index.html": "text/html; charset=utf-8", "a/b.CSS": "text/css; charset=utf-8",
		"x.js": "application/javascript", "f.woff2": "font/woff2", "i.svg": "image/svg+xml",
		"noext": "application/null", ".htaccess": "application/null", "index.xml": "application/xml",
	} {
		if got := contentTypeFor(in); got != want {
			t.Errorf("contentTypeFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScriptNames(t *testing.T) {
	for in, want := range map[string]string{
		"Meu Blog de Tiro":        "meu-blog-de-tiro",
		"  Ação & Reação!! 2026 ": "acao-e-reacao-2026",
		"___":                     "site",
		strings.Repeat("ab ", 40): strings.TrimRight(strings.Repeat("ab-", 22)[:63], "-"),
	} {
		got := ScriptNameFromTitle(in)
		if got != want {
			t.Errorf("ScriptNameFromTitle(%q) = %q, want %q", in, got, want)
		}
		if err := ValidateScriptName(got); err != nil {
			t.Errorf("normalized %q invalid: %v", got, err)
		}
	}
	for _, bad := range []string{"", "-a", "a-", "A", "a_b", "a.b", strings.Repeat("a", 64)} {
		if err := ValidateScriptName(bad); !errors.Is(err, ErrInvalidScriptName) {
			t.Errorf("ValidateScriptName(%q) = %v", bad, err)
		}
	}
}

// fakeCF is an in-memory Workers API.
type fakeCF struct {
	t  *testing.T
	mu sync.Mutex

	token        string
	subdomain    string // "" => 10007
	scripts      map[string]bool
	stored       map[string]bool // asset hashes already on Cloudflare
	uploaded     map[string]string
	partTypes    map[string]string
	sessions     int
	uploadCalls  int
	expireFirst  bool // uploads with the first session's JWT get 401
	singleMode   bool
	bucketSize   int
	lastManifest map[string]manifestEntry
	metadata     map[string]any
	subdomainReq map[string]bool
	apiDateHdr   string
}

func newFake(t *testing.T) (*fakeCF, *httptest.Server) {
	f := &fakeCF{t: t, token: "good-token", subdomain: "gabriell", scripts: map[string]bool{},
		stored: map[string]bool{}, uploaded: map[string]string{}, partTypes: map[string]string{}, bucketSize: 2}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	return f, srv
}

func ok(w http.ResponseWriter, status int, result any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"success": true, "errors": []any{}, "result": result})
}

func fail(w http.ResponseWriter, status, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []any{map[string]any{"code": code, "message": msg}}, "result": nil})
}

func fakeJWT(claims map[string]any) string {
	enc := func(v any) string { b, _ := json.Marshal(v); return base64.RawURLEncoding.EncodeToString(b) }
	return enc(map[string]any{"alg": "none"}) + "." + enc(claims) + ".sig"
}

func (f *fakeCF) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	p := r.URL.Path
	const acc = "/accounts/acc1"
	if strings.HasPrefix(p, acc+"/workers/assets/upload") {
		claims := jwtClaims(auth)
		if claims == nil || claims["kind"] != "upload" {
			fail(w, 401, 10001, "Unauthorized")
			return
		}
		if f.expireFirst && claims["session"].(float64) == 1 {
			fail(w, 401, 10001, "Unauthorized: token expired")
			return
		}
		f.uploadCalls++
		if strings.HasSuffix(p, "/upload") {
			if r.URL.Query().Get("base64") != "true" {
				f.t.Errorf("missing base64=true")
			}
			if err := r.ParseMultipartForm(64 << 20); err != nil {
				f.t.Fatalf("multipart: %v", err)
			}
			for name, fhs := range r.MultipartForm.File {
				fh := fhs[0]
				if fh.Filename != name {
					f.t.Errorf("part filename %q != name %q", fh.Filename, name)
				}
				file, _ := fh.Open()
				b, _ := io.ReadAll(file)
				raw, err := base64.StdEncoding.DecodeString(string(b))
				if err != nil {
					f.t.Errorf("part %s not base64", name)
				}
				f.uploaded[name] = string(raw)
				f.partTypes[name] = fh.Header.Get("Content-Type")
				f.stored[name] = true
			}
		} else { // single-asset mode: /upload/{hash}, raw body
			h := p[strings.LastIndex(p, "/")+1:]
			b, _ := io.ReadAll(r.Body)
			f.uploaded[h] = string(b)
			f.partTypes[h] = r.Header.Get("Content-Type")
			f.stored[h] = true
		}
		// completion token once all session hashes are stored
		res := map[string]any{}
		all := true
		for _, m := range f.lastManifest {
			all = all && f.stored[m.Hash]
		}
		if all {
			res["jwt"] = "completion-token"
			ok(w, 201, res)
			return
		}
		ok(w, 200, res)
		return
	}
	if auth != f.token {
		fail(w, 401, 1000, "Invalid API Token")
		return
	}
	switch {
	case p == "/user/tokens/verify":
		ok(w, 200, map[string]any{"status": "active"})
	case p == "/accounts":
		ok(w, 200, []Account{{ID: "acc1", Name: "Gabriell"}})
	case p == acc+"/workers/subdomain":
		if f.subdomain == "" {
			fail(w, 404, 10007, "This subdomain does not exist")
			return
		}
		ok(w, 200, map[string]string{"subdomain": f.subdomain})
	case strings.HasPrefix(p, acc+"/workers/workers/"):
		if !f.scripts[strings.TrimPrefix(p, acc+"/workers/workers/")] {
			fail(w, 404, 10007, "worker not found")
			return
		}
		ok(w, 200, map[string]any{"subdomain": map[string]any{"enabled": true}})
	case strings.HasSuffix(p, "/assets-upload-session"):
		f.sessions++
		var body struct{ Manifest map[string]manifestEntry }
		json.NewDecoder(r.Body).Decode(&body)
		f.lastManifest = body.Manifest
		seen := map[string]bool{}
		var missing []string
		for _, m := range body.Manifest {
			if !f.stored[m.Hash] && !seen[m.Hash] {
				seen[m.Hash] = true
				missing = append(missing, m.Hash)
			}
		}
		sort.Strings(missing)
		buckets := [][]string{}
		for i := 0; i < len(missing); i += f.bucketSize {
			buckets = append(buckets, missing[i:min(i+f.bucketSize, len(missing))])
		}
		jwt := "completion-token"
		if len(missing) > 0 {
			jwt = fakeJWT(map[string]any{"kind": "upload", "session": f.sessions, "exp": time.Now().Add(time.Hour).Unix(),
				"wrangler_single_asset_uploads": f.singleMode, "edge_kv_upload_concurrency": 4})
		}
		ok(w, 200, map[string]any{"jwt": jwt, "buckets": buckets})
	case strings.HasPrefix(p, acc+"/workers/scripts/") && strings.HasSuffix(p, "/subdomain"):
		f.apiDateHdr = r.Header.Get("Cloudflare-Workers-Script-Api-Date")
		json.NewDecoder(r.Body).Decode(&f.subdomainReq)
		ok(w, 200, f.subdomainReq)
	case strings.HasPrefix(p, acc+"/workers/scripts/") && r.Method == http.MethodPut:
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			f.t.Fatalf("script multipart: %v", err)
		}
		if len(r.MultipartForm.File) != 0 || len(r.MultipartForm.Value) != 1 {
			f.t.Errorf("assets-only upload must carry only metadata, got %v / %v", r.MultipartForm.Value, r.MultipartForm.File)
		}
		f.metadata = nil
		json.Unmarshal([]byte(r.FormValue("metadata")), &f.metadata)
		f.scripts[strings.TrimPrefix(p, acc+"/workers/scripts/")] = true
		ok(w, 200, map[string]any{"id": "blog", "deployment_id": "dep-9", "etag": "e"})
	default:
		f.t.Errorf("unexpected %s %s", r.Method, p)
		fail(w, 404, 7003, "no route")
	}
}

func writeSite(t *testing.T) string {
	dir := t.TempDir()
	files := map[string]string{
		"index.html":              "<h1>home</h1>",
		"404.html":                "<h1>not found</h1>",
		"posts/a/index.html":      "<h1>a</h1>",
		"posts/b/index.html":      "<h1>b</h1>",
		"css/site.css":            "body{}",
		"dup.html":                "<h1>home</h1>",
		"functions/index.html":    "hugo section",
		"_headers":                "/*\n  X-Frame-Options: DENY\n",
		"_redirects":              "/old /new 301\n",
		".assetsignore":           "",
		".DS_Store":               "junk",
		"posts/.DS_Store":         "junk",
		"node_modules/x/index.js": "junk",
		".git/HEAD":               "junk",
		"_worker.js":              "junk",
		"posts/_headers":          "kept as asset",
	}
	for name, c := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func newClient(srv *httptest.Server) *Client {
	return &Client{Token: "good-token", BaseURL: srv.URL, HTTP: srv.Client(),
		sleep: func(ctx context.Context, d time.Duration) error { return ctx.Err() }}
}

func TestEndToEnd(t *testing.T) {
	for _, single := range []bool{false, true} {
		t.Run(fmt.Sprintf("single=%v", single), func(t *testing.T) {
			f, srv := newFake(t)
			f.singleMode = single
			f.expireFirst = !single // also exercise session restart once
			c := newClient(srv)
			ctx := context.Background()

			if err := c.VerifyToken(ctx); err != nil {
				t.Fatalf("VerifyToken: %v", err)
			}
			if a, err := c.Accounts(ctx); err != nil || len(a) != 1 {
				t.Fatalf("Accounts: %v %v", a, err)
			}
			site, err := c.EnsureSite(ctx, "acc1", "blog")
			if err != nil {
				t.Fatalf("EnsureSite: %v", err)
			}
			if site.URL != "https://blog.gabriell.workers.dev/" || site.Exists {
				t.Fatalf("site = %+v", site)
			}

			dir := writeSite(t)
			cssPath := filepath.Join(dir, "css", "site.css")
			f.stored[HashFile(cssPath, []byte("body{}"))] = true

			var stages []string
			res, err := c.Deploy(ctx, "acc1", "blog", dir, func(p Progress) {
				if len(stages) == 0 || stages[len(stages)-1] != p.Stage {
					stages = append(stages, p.Stage)
				}
				if p.Total > 0 && p.Done > p.Total {
					t.Errorf("progress overflow %+v", p)
				}
			})
			if err != nil {
				t.Fatalf("Deploy: %v", err)
			}

			var paths []string
			for k := range f.lastManifest {
				paths = append(paths, k)
			}
			sort.Strings(paths)
			want := "/404.html,/css/site.css,/dup.html,/functions/index.html,/index.html,/posts/_headers,/posts/a/index.html,/posts/b/index.html"
			if strings.Join(paths, ",") != want {
				t.Fatalf("manifest = %v", paths)
			}
			if m := f.lastManifest["/index.html"]; m.Size != int64(len("<h1>home</h1>")) || m.Hash != HashFile("index.html", []byte("<h1>home</h1>")) {
				t.Errorf("manifest entry = %+v", m)
			}
			if _, up := f.uploaded[HashFile("x.css", []byte("body{}"))]; up {
				t.Error("stored css was re-uploaded")
			}
			idx := f.lastManifest["/index.html"].Hash
			if f.uploaded[idx] != "<h1>home</h1>" || f.partTypes[idx] != "text/html; charset=utf-8" {
				t.Errorf("index upload = %q type %q", f.uploaded[idx], f.partTypes[idx])
			}
			if len(f.uploaded) != 6 { // 7 new paths, index/dup share a hash
				t.Errorf("uploaded %d hashes, want 6", len(f.uploaded))
			}
			wantSessions := 1
			if !single {
				wantSessions = 2
			}
			if f.sessions != wantSessions {
				t.Errorf("sessions = %d, want %d", f.sessions, wantSessions)
			}

			assets, _ := f.metadata["assets"].(map[string]any)
			cfg, _ := assets["config"].(map[string]any)
			if assets["jwt"] != "completion-token" || cfg["html_handling"] != "auto-trailing-slash" ||
				cfg["not_found_handling"] != "404-page" || cfg["_redirects"] != "/old /new 301\n" ||
				!strings.Contains(fmt.Sprint(cfg["_headers"]), "X-Frame-Options") {
				t.Errorf("metadata = %v", f.metadata)
			}
			if _, has := f.metadata["main_module"]; has || f.metadata["compatibility_date"] != DefaultCompatibilityDate {
				t.Errorf("metadata must be assets-only: %v", f.metadata)
			}
			if !f.subdomainReq["enabled"] || f.apiDateHdr != "2025-08-01" {
				t.Errorf("subdomain req = %v, header %q", f.subdomainReq, f.apiDateHdr)
			}
			if res.URL != "https://blog.gabriell.workers.dev/" || res.DeploymentID != "dep-9" || res.Uploaded != 6 || res.Skipped != 2 {
				t.Errorf("result = %+v", res)
			}
			wantStages := []string{StageScanning, StageChecking, StageUploading, StageDeploying, StageRouting, StageDone}
			if !single {
				wantStages = []string{StageScanning, StageChecking, StageUploading, StageChecking, StageUploading, StageDeploying, StageRouting, StageDone}
			}
			if strings.Join(stages, ",") != strings.Join(wantStages, ",") {
				t.Errorf("stages = %v", stages)
			}

			// Redeploy: nothing new; the session JWT is the completion token.
			before := f.uploadCalls
			res, err = c.Deploy(ctx, "acc1", "blog", dir, nil)
			if err != nil || res.Uploaded != 0 || res.Skipped != 8 || f.uploadCalls != before {
				t.Fatalf("redeploy: %+v %v", res, err)
			}
			if site, _ := c.EnsureSite(ctx, "acc1", "blog"); !site.Exists {
				t.Error("Exists should be true after deploy")
			}
		})
	}
}

func TestErrors(t *testing.T) {
	f, srv := newFake(t)
	ctx := context.Background()

	bad := newClient(srv)
	bad.Token = "nope"
	if err := bad.VerifyToken(ctx); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("bad token = %v", err)
	}
	c := newClient(srv)
	if _, err := c.EnsureSite(ctx, "acc1", "Bad Name"); !errors.Is(err, ErrInvalidScriptName) {
		t.Errorf("bad name = %v", err)
	}
	f.subdomain = ""
	_, err := c.EnsureSite(ctx, "acc1", "blog")
	if !errors.Is(err, ErrNoWorkersSubdomain) || !strings.Contains(err.Error(), "dash.cloudflare.com/acc1/workers/onboarding") {
		t.Errorf("no subdomain = %v", err)
	}
	f.subdomain = "gabriell"

	perm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fail(w, 403, 10000, "Authentication error")
	}))
	defer perm.Close()
	if _, err := newClient(perm).EnsureSite(ctx, "acc1", "blog"); !errors.Is(err, ErrMissingPermission) {
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

	// Server-side file-count limit (Free plan) maps to ErrTooManyFiles.
	quota := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/workers/subdomain"):
			ok(w, 200, map[string]string{"subdomain": "s"})
		case strings.Contains(r.URL.Path, "/workers/workers/"):
			fail(w, 404, 10007, "not found")
		default:
			fail(w, 400, 10200, "Too many assets: maximum asset count is 20000")
		}
	}))
	defer quota.Close()
	if _, err := newClient(quota).Deploy(ctx, "acc1", "blog", dir2, nil); !errors.Is(err, ErrTooManyFiles) {
		t.Errorf("server quota = %v", err)
	}
}

func TestUploadExpiresTwice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/workers/subdomain"):
			ok(w, 200, map[string]string{"subdomain": "s"})
		case strings.Contains(r.URL.Path, "/workers/workers/"):
			fail(w, 404, 10007, "not found")
		case strings.HasSuffix(r.URL.Path, "/assets-upload-session"):
			var body struct{ Manifest map[string]manifestEntry }
			json.NewDecoder(r.Body).Decode(&body)
			var hs []string
			for _, m := range body.Manifest {
				hs = append(hs, m.Hash)
			}
			ok(w, 200, map[string]any{"jwt": fakeJWT(map[string]any{"exp": time.Now().Add(time.Hour).Unix()}), "buckets": [][]string{hs}})
		default:
			fail(w, 401, 10001, "expired")
		}
	}))
	defer srv.Close()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("x"), 0o644)
	if _, err := newClient(srv).Deploy(context.Background(), "acc1", "blog", dir, nil); !errors.Is(err, ErrUploadExpired) {
		t.Errorf("err = %v", err)
	}
}

func TestRetryOnGatewayAndRetryAfter(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	var waits []time.Duration
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/workers/subdomain"):
			ok(w, 200, map[string]string{"subdomain": "s"})
		case strings.Contains(r.URL.Path, "/workers/workers/"):
			fail(w, 404, 10007, "not found")
		case strings.HasSuffix(r.URL.Path, "/assets-upload-session"):
			var body struct{ Manifest map[string]manifestEntry }
			json.NewDecoder(r.Body).Decode(&body)
			var hs []string
			for _, m := range body.Manifest {
				hs = append(hs, m.Hash)
			}
			ok(w, 200, map[string]any{"jwt": fakeJWT(map[string]any{"exp": time.Now().Add(time.Hour).Unix()}), "buckets": [][]string{hs}})
		case strings.Contains(r.URL.Path, "/workers/assets/upload"):
			calls++
			switch calls {
			case 1:
				fail(w, 502, 0, "bad gateway")
			case 2:
				w.Header().Set("Retry-After", "7")
				fail(w, 429, 971, "Please wait and consider throttling your request speed")
			default:
				ok(w, 201, map[string]string{"jwt": "done"})
			}
		case r.Method == http.MethodPut:
			ok(w, 200, map[string]string{"id": "v1"})
		default:
			ok(w, 200, map[string]bool{"enabled": true})
		}
	}))
	defer srv.Close()
	c := newClient(srv)
	c.sleep = func(ctx context.Context, d time.Duration) error { waits = append(waits, d); return nil }
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("x"), 0o644)
	res, err := c.Deploy(context.Background(), "acc1", "blog", dir, nil)
	if err != nil || res.DeploymentID != "v1" || res.URL != "https://blog.s.workers.dev/" {
		t.Fatalf("deploy = %+v %v", res, err)
	}
	if len(waits) != 2 || waits[0] != 6*time.Second || waits[1] != 7*time.Second {
		t.Errorf("waits = %v, want [6s 7s] (1s backoff + 5s gateway, then Retry-After)", waits)
	}
}
