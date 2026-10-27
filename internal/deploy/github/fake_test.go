package github

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeGitHub is an in-memory GitHub REST API good enough for this package.
type fakeGitHub struct {
	t     *testing.T
	mu    sync.Mutex
	srv   *httptest.Server
	token string
	login string

	repos   map[string]*fakeRepo // "owner/name" lowercased
	calls   []string             // "METHOD path"
	hook    func(w http.ResponseWriter, r *http.Request) bool
	inlined []string // paths received with inline content
}

type fakeRepo struct {
	owner, name   string
	private       bool
	push          bool
	empty         bool
	defaultBranch string
	blobs         map[string][]byte
	trees         map[string]map[string]string // tree sha -> flat path->blob sha
	commits       map[string]fakeCommit
	refs          map[string]string // branch -> commit
	pages         *pagesSite
	pagesUnavail  bool
	forceTrunc    bool
	builds        []map[string]any
}

type fakeCommit struct {
	tree    string
	parents []string
	message string
}

func newFake(t *testing.T) *fakeGitHub {
	f := &fakeGitHub{t: t, token: "good-token", login: "Maria", repos: map[string]*fakeRepo{}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeGitHub) client() *Client {
	c := New(f.token)
	c.BaseURL = f.srv.URL
	c.MinWriteInterval = -1
	c.sleep = func(ctx context.Context, d time.Duration) error { return ctx.Err() }
	return c
}

func (f *fakeGitHub) addRepo(owner, name string) *fakeRepo {
	r := &fakeRepo{
		owner: owner, name: name, push: true, defaultBranch: "main",
		blobs: map[string][]byte{}, trees: map[string]map[string]string{},
		commits: map[string]fakeCommit{}, refs: map[string]string{},
	}
	f.repos[strings.ToLower(owner+"/"+name)] = r
	return r
}

// autoInit adds a README commit on the default branch.
func (r *fakeRepo) autoInit() {
	sha := r.putBlob([]byte("# " + r.name + "\n"))
	tree := r.putTree(map[string]string{"README.md": sha})
	c := r.putCommit(fakeCommit{tree: tree, message: "Initial commit"})
	r.refs[r.defaultBranch] = c
	r.empty = false
}

func (r *fakeRepo) putBlob(b []byte) string {
	s := BlobSHA(b)
	r.blobs[s] = append([]byte(nil), b...)
	return s
}

func (r *fakeRepo) putTree(flat map[string]string) string {
	keys := make([]string, 0, len(flat))
	for k := range flat {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha1.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%s %s\n", k, flat[k])
	}
	s := hex.EncodeToString(h.Sum(nil))
	r.trees[s] = flat
	return s
}

func (r *fakeRepo) putCommit(c fakeCommit) string {
	h := sha1.New()
	fmt.Fprintf(h, "%s %v %s %d", c.tree, c.parents, c.message, len(r.commits))
	s := hex.EncodeToString(h.Sum(nil))
	r.commits[s] = c
	return s
}

// files returns path->content of branch.
func (r *fakeRepo) files(branch string) map[string]string {
	c := r.commits[r.refs[branch]]
	out := map[string]string{}
	for p, s := range r.trees[c.tree] {
		out[p] = string(r.blobs[s])
	}
	return out
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func msg(w http.ResponseWriter, code int, m string) {
	writeJSON(w, code, map[string]any{"message": m, "documentation_url": "https://docs.github.com/rest"})
}

func (f *fakeGitHub) count(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func (f *fakeGitHub) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)
	if r.Header.Get("X-GitHub-Api-Version") != APIVersion {
		msg(w, 400, "bad api version")
		return
	}
	if f.hook != nil && f.hook(w, r) {
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+f.token {
		msg(w, 401, "Bad credentials")
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var body map[string]any
	if r.Body != nil && r.Method != http.MethodGet {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	switch {
	case r.URL.Path == "/user" && r.Method == http.MethodGet:
		writeJSON(w, 200, map[string]any{"login": f.login})
		return
	case r.URL.Path == "/user/repos" && r.Method == http.MethodPost:
		name := body["name"].(string)
		if _, ok := f.repos[strings.ToLower(f.login+"/"+name)]; ok {
			writeJSON(w, 422, map[string]any{"message": "Repository creation failed.", "errors": []any{map[string]any{"resource": "Repository", "code": "custom", "field": "name", "message": "name already exists on this account"}}})
			return
		}
		if body["private"] != false || body["auto_init"] != true {
			msg(w, 400, "test expects public auto_init repo")
			return
		}
		rp := f.addRepo(f.login, name)
		rp.autoInit()
		writeJSON(w, 201, f.repoJSON(rp))
		return
	}
	if len(parts) < 3 || parts[0] != "repos" {
		msg(w, 404, "Not Found")
		return
	}
	rp := f.repos[strings.ToLower(parts[1]+"/"+parts[2])]
	if rp == nil {
		msg(w, 404, "Not Found")
		return
	}
	rest := parts[3:]
	switch {
	case len(rest) == 0 && r.Method == http.MethodGet:
		writeJSON(w, 200, f.repoJSON(rp))
	case len(rest) >= 1 && rest[0] == "git" && rp.empty:
		msg(w, 409, "Git Repository is empty.")
	case len(rest) == 2 && rest[0] == "contents" && r.Method == http.MethodPut:
		if !rp.empty {
			msg(w, 422, "Invalid request. \"sha\" wasn't supplied.")
			return
		}
		rp.autoInit()
		writeJSON(w, 201, map[string]any{})
	case len(rest) >= 4 && rest[0] == "git" && rest[1] == "ref" && rest[2] == "heads" && r.Method == http.MethodGet:
		br := strings.Join(rest[3:], "/")
		sha, ok := rp.refs[br]
		if !ok {
			msg(w, 404, "Not Found")
			return
		}
		writeJSON(w, 200, map[string]any{"ref": "refs/heads/" + br, "object": map[string]any{"sha": sha, "type": "commit"}})
	case len(rest) == 3 && rest[0] == "git" && rest[1] == "commits" && r.Method == http.MethodGet:
		c, ok := rp.commits[rest[2]]
		if !ok {
			msg(w, 404, "Not Found")
			return
		}
		writeJSON(w, 200, map[string]any{"sha": rest[2], "tree": map[string]any{"sha": c.tree}})
	case len(rest) == 3 && rest[0] == "git" && rest[1] == "trees" && r.Method == http.MethodGet:
		f.getTree(w, r, rp, rest[2])
	case len(rest) == 2 && rest[0] == "git" && rest[1] == "blobs" && r.Method == http.MethodPost:
		if body["encoding"] != "base64" {
			msg(w, 422, "encoding")
			return
		}
		data, err := base64.StdEncoding.DecodeString(body["content"].(string))
		if err != nil {
			msg(w, 422, "bad base64")
			return
		}
		writeJSON(w, 201, map[string]any{"sha": rp.putBlob(data)})
	case len(rest) == 2 && rest[0] == "git" && rest[1] == "trees" && r.Method == http.MethodPost:
		flat := map[string]string{}
		if b, ok := body["base_tree"].(string); ok {
			base, ok := rp.trees[b]
			if !ok {
				msg(w, 422, "base_tree not found")
				return
			}
			for k, v := range base {
				flat[k] = v
			}
		}
		for _, e := range body["tree"].([]any) {
			m := e.(map[string]any)
			p := m["path"].(string)
			if c, ok := m["content"].(string); ok {
				flat[p] = rp.putBlob([]byte(c))
				f.inlined = append(f.inlined, p)
				continue
			}
			s, _ := m["sha"].(string)
			if _, ok := rp.blobs[s]; !ok {
				msg(w, 422, "tree.sha "+s+" is not a valid blob")
				return
			}
			flat[p] = s
		}
		writeJSON(w, 201, map[string]any{"sha": rp.putTree(flat)})
	case len(rest) == 2 && rest[0] == "git" && rest[1] == "commits" && r.Method == http.MethodPost:
		tree := body["tree"].(string)
		if _, ok := rp.trees[tree]; !ok {
			msg(w, 422, "tree not found")
			return
		}
		var parents []string
		for _, p := range body["parents"].([]any) {
			parents = append(parents, p.(string))
		}
		sha := rp.putCommit(fakeCommit{tree: tree, parents: parents, message: body["message"].(string)})
		writeJSON(w, 201, map[string]any{"sha": sha})
	case len(rest) == 2 && rest[0] == "git" && rest[1] == "refs" && r.Method == http.MethodPost:
		ref := strings.TrimPrefix(body["ref"].(string), "refs/heads/")
		if _, ok := rp.refs[ref]; ok {
			msg(w, 422, "Reference already exists")
			return
		}
		rp.refs[ref] = body["sha"].(string)
		writeJSON(w, 201, map[string]any{})
	case len(rest) >= 4 && rest[0] == "git" && rest[1] == "refs" && rest[2] == "heads" && r.Method == http.MethodPatch:
		br := strings.Join(rest[3:], "/")
		if _, ok := rp.refs[br]; !ok {
			msg(w, 422, "Reference does not exist")
			return
		}
		if body["force"] != true {
			msg(w, 422, "expected force")
			return
		}
		rp.refs[br] = body["sha"].(string)
		writeJSON(w, 200, map[string]any{})
	case len(rest) == 1 && rest[0] == "pages":
		f.pages(w, r, rp, body)
	case len(rest) == 3 && rest[0] == "pages" && rest[1] == "builds" && rest[2] == "latest":
		if len(rp.builds) == 0 {
			msg(w, 404, "Not Found")
			return
		}
		b := rp.builds[0]
		if len(rp.builds) > 1 {
			rp.builds = rp.builds[1:]
		}
		writeJSON(w, 200, b)
	default:
		msg(w, 404, "Not Found: "+r.Method+" "+r.URL.Path)
	}
}

func (f *fakeGitHub) repoJSON(rp *fakeRepo) map[string]any {
	return map[string]any{
		"name": rp.name, "full_name": rp.owner + "/" + rp.name, "private": rp.private,
		"html_url": "https://github.com/" + rp.owner + "/" + rp.name, "default_branch": rp.defaultBranch,
		"owner":       map[string]any{"login": rp.owner},
		"permissions": map[string]any{"admin": rp.push, "push": rp.push},
	}
}

func (f *fakeGitHub) getTree(w http.ResponseWriter, r *http.Request, rp *fakeRepo, sha string) {
	flat, ok := rp.trees[sha]
	if !ok {
		msg(w, 404, "Not Found")
		return
	}
	var entries []map[string]any
	if r.URL.Query().Get("recursive") != "" {
		if rp.forceTrunc {
			writeJSON(w, 200, map[string]any{"sha": sha, "tree": []any{}, "truncated": true})
			return
		}
		for p, s := range flat {
			entries = append(entries, map[string]any{"path": p, "type": "blob", "sha": s, "mode": "100644"})
		}
		writeJSON(w, 200, map[string]any{"sha": sha, "tree": entries, "truncated": false})
		return
	}
	// Non-recursive: one level; subdirectories become synthetic trees.
	subs := map[string]map[string]string{}
	for p, s := range flat {
		if i := strings.Index(p, "/"); i >= 0 {
			d := p[:i]
			if subs[d] == nil {
				subs[d] = map[string]string{}
			}
			subs[d][p[i+1:]] = s
			continue
		}
		entries = append(entries, map[string]any{"path": p, "type": "blob", "sha": s})
	}
	for d, m := range subs {
		entries = append(entries, map[string]any{"path": d, "type": "tree", "sha": rp.putTree(m)})
	}
	writeJSON(w, 200, map[string]any{"sha": sha, "tree": entries, "truncated": false})
}

func (f *fakeGitHub) pages(w http.ResponseWriter, r *http.Request, rp *fakeRepo, body map[string]any) {
	url := ExpectedURL(rp.owner, rp.name)
	setSource := func() bool {
		src, _ := body["source"].(map[string]any)
		if src == nil || src["branch"] == nil || src["path"] != "/" || body["build_type"] != "legacy" {
			msg(w, 422, "Invalid source")
			return false
		}
		rp.pages = &pagesSite{HTMLURL: url, BuildType: "legacy", Source: &struct {
			Branch string `json:"branch"`
			Path   string `json:"path"`
		}{src["branch"].(string), "/"}}
		return true
	}
	switch r.Method {
	case http.MethodGet:
		if rp.pages == nil {
			msg(w, 404, "Not Found")
			return
		}
		writeJSON(w, 200, rp.pages)
	case http.MethodPost:
		if rp.pagesUnavail {
			msg(w, 422, "Your current plan does not support GitHub Pages for this repository.")
			return
		}
		if rp.pages != nil {
			msg(w, 409, "GitHub Pages is already enabled.")
			return
		}
		if setSource() {
			writeJSON(w, 201, rp.pages)
		}
	case http.MethodPut:
		if rp.pages == nil {
			msg(w, 404, "Not Found")
			return
		}
		if setSource() {
			w.WriteHeader(204)
		}
	}
}
