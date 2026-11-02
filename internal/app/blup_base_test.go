package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
)

const baseSiteID = "6f1c1f9e-1b2a-4c3d-9e8f-0a1b2c3d4e5f"

// fakeDomainCloud answers /api/me with the plan, the site's domain with
// *domain (JSON, "null" for none; "" fails with 500) and takes deploys.
func fakeDomainCloud(t *testing.T, plan *bool, domain *string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/me":
			json.NewEncoder(w).Encode(map[string]any{"id": "u", "plan": map[string]any{"name": "max", "limits": map[string]any{"customDomain": *plan, "sites": 5}}})
		case strings.HasSuffix(r.URL.Path, "/domain") && r.Method == http.MethodGet:
			if *domain == "" {
				w.WriteHeader(500)
				w.Write([]byte(`{"code":"INTERNAL"}`))
				return
			}
			w.Write([]byte(*domain))
		case strings.HasSuffix(r.URL.Path, "/domain") && r.Method == http.MethodDelete:
			*domain = "null"
			w.Write([]byte(`{"ok":true}`))
		case strings.HasSuffix(r.URL.Path, "/missing"):
			w.Write([]byte(`{"missing":[]}`))
		case strings.HasSuffix(r.URL.Path, "/deploys"):
			w.Write([]byte(`{"url":"https://meu-blog.getblup.site"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("BLUP_NO_SEARCH", "1") // no Pagefind download in tests
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	if err := (config.Credentials{BlupToken: "tok"}).Save(); err != nil {
		t.Fatal(err)
	}
}

func domainJSON(status, ssl, disableAt string) string {
	d := map[string]any{"hostname": "meusite.com.br", "status": status, "ssl": ssl, "records": []any{}, "disableAt": nil}
	if disableAt != "" {
		d["disableAt"] = disableAt
	}
	b, _ := json.Marshal(d)
	return string(b)
}

func blupSiteOn(t *testing.T) *config.Site {
	site := settingsSite(t)
	site.Deploy.Provider = "blup"
	site.Deploy.Blup = config.BlupSite{ID: baseSiteID, Slug: "meu-blog", URL: "https://meu-blog.getblup.site"}
	return site
}

// The official address: the own domain only while it is active with its
// HTTPS and the plan includes it (or the 14 days after a downgrade run).
func TestBlupBaseChoice(t *testing.T) {
	future := time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339)
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	const own, native = "https://meusite.com.br/", "https://meu-blog.getblup.site"
	for _, c := range []struct {
		name   string
		plan   bool
		domain string
		want   string
	}{
		{"none", true, "null", native},
		{"waiting for the TXT", true, domainJSON("pending_ownership", "pending", ""), native},
		{"active without HTTPS yet", true, domainJSON("active", "pending", ""), native},
		{"active", true, domainJSON("active", "active", ""), own},
		{"downgrade without grace (old server)", false, domainJSON("active", "active", ""), native},
		{"downgrade within the grace", false, domainJSON("active", "active", future), own},
		{"grace over", false, domainJSON("active", "active", past), native},
		{"bad date", false, domainJSON("active", "active", "amanhã"), native},
		{"disabled", false, domainJSON("disabled", "pending", past), native},
		{"error", true, domainJSON("error", "active", ""), native},
	} {
		t.Run(c.name, func(t *testing.T) {
			plan, domain := c.plan, c.domain
			fakeDomainCloud(t, &plan, &domain)
			site := blupSiteOn(t)
			cl, _, err := blupClient()
			if err != nil {
				t.Fatal(err)
			}
			got, isOwn := blupBase(context.Background(), cl, site)
			if got != c.want || isOwn != (c.want == own) {
				t.Fatalf("base = %q (own %v), want %q", got, isOwn, c.want)
			}
		})
	}
}

// When the cloud can't tell, the last publish's address stays.
func TestBlupBaseKeepsLastOnError(t *testing.T) {
	plan, domain := true, ""
	fakeDomainCloud(t, &plan, &domain)
	site := blupSiteOn(t)
	site.Deploy.Blup.Base = "https://meusite.com.br/"
	cl, _, _ := blupClient()
	if got, own := blupBase(context.Background(), cl, site); got != "https://meusite.com.br/" || !own {
		t.Fatalf("base = %q %v", got, own)
	}
	site.Deploy.Blup.Base = ""
	if got, own := blupBase(context.Background(), cl, site); got != site.Deploy.Blup.URL || own {
		t.Fatalf("base = %q %v", got, own)
	}
}

// The Publish screen learns when the links point to the wrong address.
func TestBlupDomainRepublish(t *testing.T) {
	plan, domain := true, domainJSON("active", "active", "")
	fakeDomainCloud(t, &plan, &domain)
	ctx := context.Background()
	site := blupSiteOn(t)

	// Published before the base was kept (getblup.site): now active.
	d, err := LoadBlupDomain(ctx, site)
	if err != nil || !d.Republish {
		t.Fatalf("turned active: %+v %v", d, err)
	}
	site.Deploy.Blup.Base = "https://meusite.com.br/"
	if d, _ = LoadBlupDomain(ctx, site); d.Republish {
		t.Fatal("asked again for the address already published")
	}
	// Removed: back to getblup.site.
	if d, err = RemoveBlupDomain(ctx, site); err != nil || !d.Republish {
		t.Fatalf("removed: %+v %v", d, err)
	}
	// Turned off after the grace: same.
	domain = domainJSON("disabled", "pending", "2020-01-01T00:00:00Z")
	if d, _ = LoadBlupDomain(ctx, site); !d.Republish || d.Domain == nil || d.Domain.Status != "disabled" {
		t.Fatalf("disabled: %+v", d)
	}
	// Another destination: Blup Cloud's links are not this site's.
	site.Deploy.Provider = "github"
	if d, _ = LoadBlupDomain(ctx, site); d.Republish {
		t.Fatal("republish asked for a site published elsewhere")
	}
}

// A real build: canonical, feed, sitemap and robots carry the own domain
// while it is active, and the native address once it is gone.
func TestDeployBlupUsesOwnDomain(t *testing.T) {
	if _, err := exec.LookPath("hugo"); err != nil {
		t.Skip("hugo not installed")
	}
	plan, domain := true, domainJSON("active", "active", "")
	fakeDomainCloud(t, &plan, &domain)
	site := blupSiteOn(t)
	ctx := context.Background()

	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(site.Dir(), "public", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	live, err := deployBlup(ctx, site)
	if err != nil {
		t.Fatal(err)
	}
	if live != "https://meusite.com.br" || site.Deploy.Blup.Base != "https://meusite.com.br/" {
		t.Fatalf("live %q, base %q", live, site.Deploy.Blup.Base)
	}
	for _, f := range []string{"index.html", "sitemap.xml", "robots.txt", "index.xml"} {
		s := read(f)
		if !strings.Contains(s, "https://meusite.com.br/") || strings.Contains(s, "getblup.site") {
			t.Errorf("%s does not point to the own domain:\n%.400s", f, s)
		}
	}
	if !strings.Contains(read("index.html"), `rel=canonical href=https://meusite.com.br/`) {
		t.Error("canonical is not the own domain")
	}

	domain = "null"
	if live, err = deployBlup(ctx, site); err != nil {
		t.Fatal(err)
	}
	if live != "https://meu-blog.getblup.site" || site.Deploy.Blup.Base != "https://meu-blog.getblup.site" {
		t.Fatalf("live %q, base %q", live, site.Deploy.Blup.Base)
	}
	for _, f := range []string{"index.html", "sitemap.xml", "robots.txt", "index.xml"} {
		if s := read(f); strings.Contains(s, "meusite.com.br") {
			t.Errorf("%s still points to the removed domain", f)
		}
	}
}

func TestDomainOfficialNil(t *testing.T) {
	var d *blupcloud.DomainInfo
	if d.Official(true, time.Now()) {
		t.Fatal("no domain is official")
	}
}

// Notes not synced through Blup Cloud have no history there: nothing is
// sent.
func TestPurgeHistoryNeedsBlupSync(t *testing.T) {
	plan, domain := true, "null"
	fakeDomainCloud(t, &plan, &domain)
	site := blupSiteOn(t)
	for _, s := range []config.NoteSync{{}, {Remote: "github", Owner: "a", Repo: "b"}, {Remote: RemoteBlup}} {
		site.Sync = s
		if _, err := PurgeHistory(context.Background(), site); err == nil {
			t.Errorf("%+v: purged", s)
		}
	}
}
