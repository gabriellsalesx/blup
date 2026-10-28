package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
)

// fakeIssues is the part of GitHub's issues API that syncIssues uses.
type fakeIssues struct {
	mu      sync.Mutex
	issues  []github.Issue
	labels  map[string]bool
	creates int
	updates int
}

func (f *fakeIssues) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	switch {
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/labels"):
		var l struct{ Name string }
		json.Unmarshal(body, &l)
		if f.labels[l.Name] {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"message":"Validation Failed"}`))
			return
		}
		f.labels[l.Name] = true
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/issues"):
		json.NewEncoder(w).Encode(f.issues)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/issues"):
		var in github.Issue
		json.Unmarshal(body, &in)
		in.Number = len(f.issues) + 1
		f.issues = append(f.issues, in)
		f.creates++
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(in)
	case r.Method == http.MethodPatch:
		var in github.Issue
		json.Unmarshal(body, &in)
		for i := range f.issues {
			if strings.HasSuffix(r.URL.Path, "/issues/"+itoa(f.issues[i].Number)) {
				f.issues[i].Title, f.issues[i].Body = in.Title, in.Body
			}
		}
		f.updates++
		w.Write([]byte(`{}`))
	default:
		http.NotFound(w, r)
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestSyncIssues(t *testing.T) {
	fake := &fakeIssues{labels: map[string]bool{}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	c := github.New("t")
	c.BaseURL = srv.URL
	site := config.NewSite(t.TempDir())
	site.Lang, site.Model = "pt", "bento"
	site.Feedback = config.Feedback{Enabled: true, Owner: "ana", Repo: "site-feedback"}
	pages := []feedbackPage{
		{Key: "roadmap/mais temas.md", Kind: "roadmap", Title: "Mais temas", URL: "https://x.pages.dev/roadmap/mais-temas/"},
		{Key: "changelog/v1.md", Kind: "changelog", Title: "Versão 1", URL: "https://x.pages.dev/changelog/v1/"},
	}
	ctx := context.Background()

	issues := map[string]int{}
	created, updated, err := syncIssues(ctx, c, site, pages, issues)
	if err != nil || created != 2 || updated != 0 || issues["roadmap/mais temas.md"] != 1 || issues["changelog/v1.md"] != 2 {
		t.Fatalf("first sync: %d %d %v %v", created, updated, issues, err)
	}
	if !strings.Contains(fake.issues[0].Body, "<!-- blup:roadmap/mais temas.md -->") {
		t.Fatalf("marker missing: %q", fake.issues[0].Body)
	}

	// Nothing changed: no new issue, no update.
	issues = map[string]int{}
	created, updated, err = syncIssues(ctx, c, site, pages, issues)
	if err != nil || created != 0 || updated != 0 || len(issues) != 2 {
		t.Fatalf("second sync: %d %d %v %v", created, updated, issues, err)
	}

	// A renamed page keeps its issue, with the new title.
	pages[0].Title = "Muitos temas"
	created, updated, err = syncIssues(ctx, c, site, pages, issues)
	if err != nil || created != 0 || updated != 1 || fake.issues[0].Title != "Muitos temas" {
		t.Fatalf("rename: %d %d %v", created, updated, err)
	}
}
