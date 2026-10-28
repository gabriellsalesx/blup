package notesync

import (
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

	"github.com/gabriellsalesx/blup/internal/deploy/github"
)

// fakeGH is an in-memory GitHub with one user and its repositories,
// implementing the endpoints notesync uses.
type fakeGH struct {
	t     *testing.T
	mu    sync.Mutex
	srv   *httptest.Server
	repos map[string]*fakeRepo
	calls []string
	// hook runs (under the lock) before normal handling; returning true
	// means it answered.
	hook func(w http.ResponseWriter, r *http.Request, body map[string]any) bool
}

type fakeRepo struct {
	name    string
	private bool
	empty   bool
	blobs   map[string][]byte
	trees   map[string]map[string]string // sha -> path -> blob
	commits map[string]fakeCommit
	refs    map[string]string
}

type fakeCommit struct {
	tree    string
	parents []string
	message string
}

func newFakeGH(t *testing.T) *fakeGH {
	f := &fakeGH{t: t, repos: map[string]*fakeRepo{}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeGH) client() *github.Client {
	c := github.New("tok")
	c.BaseURL = f.srv.URL
	c.MinWriteInterval = -1
	c.MaxRetries = 1
	return c
}

func (r *fakeRepo) putBlob(b []byte) string {
	s := github.BlobSHA(b)
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
	cp := map[string]string{}
	for k, v := range flat {
		cp[k] = v
	}
	r.trees[s] = cp
	return s
}

func (r *fakeRepo) putCommit(c fakeCommit) string {
	h := sha1.New()
	fmt.Fprintf(h, "%s %v %s %d", c.tree, c.parents, c.message, len(r.commits))
	s := hex.EncodeToString(h.Sum(nil))
	r.commits[s] = c
	return s
}

func (r *fakeRepo) isAncestor(anc, c string) bool {
	stack := []string{c}
	seen := map[string]bool{}
	for len(stack) > 0 {
		x := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if x == anc {
			return true
		}
		if !seen[x] {
			seen[x] = true
			stack = append(stack, r.commits[x].parents...)
		}
	}
	return false
}

// files returns path -> content on main.
func (f *fakeGH) files(repo string) map[string]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.repos[repo]
	out := map[string]string{}
	for p, s := range r.trees[r.commits[r.refs["main"]].tree] {
		out[p] = string(r.blobs[s])
	}
	return out
}

// push simulates another computer committing on main: set non-empty
// values, delete keys with empty values.
func (f *fakeGH) push(repo string, change map[string]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pushLocked(f.repos[repo], change)
}

func (f *fakeGH) pushLocked(r *fakeRepo, change map[string]string) {
	head := r.refs["main"]
	flat := map[string]string{}
	for k, v := range r.trees[r.commits[head].tree] {
		flat[k] = v
	}
	for p, c := range change {
		if c == "" {
			delete(flat, p)
		} else {
			flat[p] = r.putBlob([]byte(c))
		}
	}
	var parents []string
	if head != "" {
		parents = []string{head}
	}
	r.refs["main"] = r.putCommit(fakeCommit{tree: r.putTree(flat), parents: parents, message: "other computer"})
	r.empty = false
}

func (f *fakeGH) count(prefix string) int {
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

func (f *fakeGH) commitCount(repo string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.repos[repo].commits)
}

func reply(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, m string) {
	reply(w, code, map[string]any{"message": m})
}

func (f *fakeGH) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)
	var body map[string]any
	if r.Method != http.MethodGet {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	if f.hook != nil && f.hook(w, r, body) {
		return
	}
	if r.Header.Get("Authorization") != "Bearer tok" {
		fail(w, 401, "Bad credentials")
		return
	}
	switch {
	case r.URL.Path == "/user":
		reply(w, 200, map[string]any{"login": "maria"})
		return
	case r.URL.Path == "/user/repos" && r.Method == http.MethodPost:
		name := body["name"].(string)
		if body["private"] != true || body["auto_init"] != false {
			fail(w, 400, "test expects a private empty repo")
			return
		}
		f.repos[name] = &fakeRepo{name: name, private: true, empty: true,
			blobs: map[string][]byte{}, trees: map[string]map[string]string{},
			commits: map[string]fakeCommit{}, refs: map[string]string{}}
		reply(w, 201, f.repoJSON(f.repos[name]))
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "repos" || parts[1] != "maria" || f.repos[parts[2]] == nil {
		fail(w, 404, "Not Found")
		return
	}
	rp := f.repos[parts[2]]
	rest := parts[3:]
	get := r.Method == http.MethodGet
	switch {
	case len(rest) == 0 && get:
		reply(w, 200, f.repoJSON(rp))
	case len(rest) > 0 && rest[0] == "git" && rp.empty:
		fail(w, 409, "Git Repository is empty.")
	case len(rest) >= 2 && rest[0] == "contents" && r.Method == http.MethodPut:
		if !rp.empty {
			fail(w, 422, "sha wasn't supplied")
			return
		}
		data, _ := base64.StdEncoding.DecodeString(body["content"].(string))
		f.pushLocked(rp, map[string]string{strings.Join(rest[1:], "/"): string(data)})
		reply(w, 201, map[string]any{})
	case len(rest) >= 4 && rest[1] == "ref" && get:
		sha, ok := rp.refs[strings.Join(rest[3:], "/")]
		if !ok {
			fail(w, 404, "Not Found")
			return
		}
		reply(w, 200, map[string]any{"object": map[string]any{"sha": sha, "type": "commit"}})
	case len(rest) == 3 && rest[1] == "commits" && get:
		c, ok := rp.commits[rest[2]]
		if !ok {
			fail(w, 404, "Not Found")
			return
		}
		reply(w, 200, map[string]any{"sha": rest[2], "tree": map[string]any{"sha": c.tree}})
	case len(rest) == 3 && rest[1] == "trees" && get:
		flat, ok := rp.trees[rest[2]]
		if !ok || r.URL.Query().Get("recursive") == "" {
			fail(w, 404, "Not Found")
			return
		}
		var entries []map[string]any
		for p, s := range flat {
			entries = append(entries, map[string]any{"path": p, "type": "blob", "sha": s, "mode": "100644"})
		}
		reply(w, 200, map[string]any{"tree": entries, "truncated": false})
	case len(rest) == 3 && rest[1] == "blobs" && get:
		b, ok := rp.blobs[rest[2]]
		if !ok {
			fail(w, 404, "Not Found")
			return
		}
		w.Write(b)
	case len(rest) == 2 && rest[1] == "blobs" && r.Method == http.MethodPost:
		data, err := base64.StdEncoding.DecodeString(body["content"].(string))
		if err != nil {
			fail(w, 422, "bad base64")
			return
		}
		reply(w, 201, map[string]any{"sha": rp.putBlob(data)})
	case len(rest) == 2 && rest[1] == "trees" && r.Method == http.MethodPost:
		flat := map[string]string{}
		if b, ok := body["base_tree"].(string); ok {
			base, ok := rp.trees[b]
			if !ok {
				fail(w, 422, "base_tree not found")
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
				continue
			}
			if v, ok := m["sha"]; ok && v == nil {
				if _, ok := flat[p]; !ok {
					fail(w, 422, "delete of missing "+p)
					return
				}
				delete(flat, p)
				continue
			}
			s, _ := m["sha"].(string)
			if _, ok := rp.blobs[s]; !ok {
				fail(w, 422, "bad blob "+s)
				return
			}
			flat[p] = s
		}
		reply(w, 201, map[string]any{"sha": rp.putTree(flat)})
	case len(rest) == 2 && rest[1] == "commits" && r.Method == http.MethodPost:
		var parents []string
		for _, p := range body["parents"].([]any) {
			parents = append(parents, p.(string))
		}
		reply(w, 201, map[string]any{"sha": rp.putCommit(fakeCommit{tree: body["tree"].(string), parents: parents, message: body["message"].(string)})})
	case len(rest) == 2 && rest[1] == "refs" && r.Method == http.MethodPost:
		ref := strings.TrimPrefix(body["ref"].(string), "refs/heads/")
		if _, ok := rp.refs[ref]; ok {
			fail(w, 422, "Reference already exists")
			return
		}
		rp.refs[ref] = body["sha"].(string)
		reply(w, 201, map[string]any{})
	case len(rest) >= 4 && rest[1] == "refs" && r.Method == http.MethodPatch:
		br := strings.Join(rest[3:], "/")
		sha := body["sha"].(string)
		if body["force"] != false {
			fail(w, 400, "notesync must never force")
			return
		}
		if !rp.isAncestor(rp.refs[br], sha) {
			fail(w, 422, "Update is not a fast forward")
			return
		}
		rp.refs[br] = sha
		reply(w, 200, map[string]any{})
	default:
		fail(w, 404, "Not Found: "+r.Method+" "+r.URL.Path)
	}
}

func (f *fakeGH) repoJSON(rp *fakeRepo) map[string]any {
	return map[string]any{
		"name": rp.name, "full_name": "maria/" + rp.name, "private": rp.private,
		"default_branch": "main", "owner": map[string]any{"login": "maria"},
		"permissions": map[string]any{"admin": true, "push": true},
	}
}
