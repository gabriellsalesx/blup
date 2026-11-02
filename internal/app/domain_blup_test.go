package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

func TestNormalizeBlupHost(t *testing.T) {
	for in, want := range map[string]string{
		" Blog.MeuSite.com.br ": "blog.meusite.com.br",
		"café.com.":             "xn--caf-dma.com",
	} {
		if got, err := normalizeBlupHost(in); err != nil || got != want {
			t.Errorf("%q = %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"https://meusite.com", "meusite.com/blog", "meusite.com:8080", "nome.getblup.site", "getblup.io", "localhost", "192.168.0.1", "meusite", ""} {
		if got, err := normalizeBlupHost(in); err == nil {
			t.Errorf("%q accepted as %q", in, got)
		}
	}
}

// fakeBlupCloud answers /api/me with plan and the domain routes of one site.
func fakeBlupCloud(t *testing.T, customDomain bool) (*httptest.Server, *[]string) {
	t.Helper()
	var calls []string
	var linked any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/me":
			json.NewEncoder(w).Encode(map[string]any{"id": "u", "plan": map[string]any{"name": "max", "limits": map[string]any{"customDomain": customDomain, "sites": 5}}})
		case strings.HasSuffix(r.URL.Path, "/domain") && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(linked)
		case strings.HasSuffix(r.URL.Path, "/domain") && r.Method == http.MethodPost:
			if !customDomain {
				w.WriteHeader(403)
				json.NewEncoder(w).Encode(map[string]string{"code": "PLAN"})
				return
			}
			var in struct{ Hostname string }
			json.NewDecoder(r.Body).Decode(&in)
			linked = map[string]any{"hostname": in.Hostname, "status": "pending_ownership", "ssl": "pending", "records": []map[string]string{
				{"type": "TXT", "name": "_blup." + in.Hostname, "value": "blup-verify=1"},
				{"type": "CNAME", "name": in.Hostname, "value": "sites.example.net"},
			}}
			json.NewEncoder(w).Encode(linked)
		case strings.HasSuffix(r.URL.Path, "/domain") && r.Method == http.MethodDelete:
			linked = nil
			w.Write([]byte(`{"ok":true}`))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	if err := (config.Credentials{BlupToken: "tok"}).Save(); err != nil {
		t.Fatal(err)
	}
	return srv, &calls
}

func TestBlupDomainFlow(t *testing.T) {
	_, calls := fakeBlupCloud(t, true)
	ctx := context.Background()
	site := settingsSite(t)

	// Not on Blup Cloud yet: nothing is asked of the domain routes.
	d, err := LoadBlupDomain(ctx, site)
	if err != nil || d.Published || !d.Available || d.Domain != nil {
		t.Fatalf("unpublished: %+v %v", d, err)
	}
	if _, err := SetBlupDomain(ctx, site, "meusite.com.br"); err == nil {
		t.Fatal("linked a domain to a site not on Blup Cloud")
	}

	site.Deploy.Blup.ID = "6f1c1f9e-1b2a-4c3d-9e8f-0a1b2c3d4e5f"
	d, err = SetBlupDomain(ctx, site, "MeuSite.com.br")
	if err != nil || d.Domain == nil || d.Domain.Hostname != "meusite.com.br" || !d.Apex || len(d.Domain.Records) != 2 {
		t.Fatalf("set: %+v %v", d, err)
	}
	if d, err = LoadBlupDomain(ctx, site); err != nil || d.Domain == nil || d.Domain.Status != "pending_ownership" {
		t.Fatalf("load: %+v %v", d, err)
	}
	if d, err = RemoveBlupDomain(ctx, site); err != nil || d.Domain != nil {
		t.Fatalf("remove: %+v %v", d, err)
	}
	n := len(*calls)
	if _, err := SetBlupDomain(ctx, site, "https://meusite.com.br"); err == nil || len(*calls) != n {
		t.Fatal("a domain with https:// went to the cloud")
	}
}

// Free and Pro: the plan comes from the cloud; the POST is not even tried.
func TestBlupDomainPlan(t *testing.T) {
	_, calls := fakeBlupCloud(t, false)
	site := settingsSite(t)
	site.Deploy.Blup.ID = "6f1c1f9e-1b2a-4c3d-9e8f-0a1b2c3d4e5f"
	d, err := LoadBlupDomain(context.Background(), site)
	if err != nil || d.Available {
		t.Fatalf("load: %+v %v", d, err)
	}
	if _, err := SetBlupDomain(context.Background(), site, "meusite.com.br"); err == nil {
		t.Fatal("linked without the plan")
	}
	for _, c := range *calls {
		if strings.HasPrefix(c, "POST") {
			t.Fatalf("POST sent without the plan: %v", *calls)
		}
	}
}

func TestGitHubLinksLeastPrivilege(t *testing.T) {
	u, _ := url.Parse(GitHubKeyLink())
	if u.Query().Has("administration") || u.Query().Get("contents") != "write" {
		t.Fatalf("key link %s", u)
	}
	site := settingsSite(t)
	site.Title = "Revista Nori"
	notes, _ := url.Parse(GitHubNewRepoLink(site, "notes", ""))
	if notes.Query().Get("visibility") != "private" || !strings.HasSuffix(notes.Query().Get("name"), "-notas") && !strings.HasSuffix(notes.Query().Get("name"), "-notes") {
		t.Fatalf("notes link %s", notes)
	}
	pub, _ := url.Parse(GitHubNewRepoLink(site, "site", "revista"))
	if pub.Query().Get("visibility") != "public" || pub.Query().Get("name") != "revista" {
		t.Fatalf("site link %s", pub)
	}
	for _, bad := range []string{"../x", "-x", "ext::sh", "a/b"} {
		if GitHubNewRepoLink(site, "site", bad) != "" {
			t.Errorf("link for %q", bad)
		}
		if err := SetTarget(site, Target{Provider: "github", Repo: bad}); err == nil {
			t.Errorf("SetTarget accepted %q", bad)
		}
	}
	if !notesRepoName("Revista-Notas") || notesRepoName("revista") {
		t.Fatal("notesRepoName")
	}
}
