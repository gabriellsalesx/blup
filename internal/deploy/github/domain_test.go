package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPagesDomain(t *testing.T) {
	var cname any
	enforced, healthCalls := false, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/ana/site/pages":
			if r.Method == http.MethodPut {
				var b map[string]any
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &b)
				if v, ok := b["cname"]; ok {
					cname = v
				}
				if v, ok := b["https_enforced"]; ok {
					enforced = v.(bool)
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"cname": cname, "https_enforced": enforced, "https_certificate": map[string]any{"state": "approved"}})
		case "/repos/ana/site/pages/health":
			healthCalls++
			if healthCalls == 1 {
				w.WriteHeader(http.StatusAccepted)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"domain": map[string]any{"is_valid": true, "is_served_by_pages": true, "is_apex_domain": true, "is_https_eligible": true}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := New("t")
	c.BaseURL = srv.URL
	ctx := context.Background()

	if err := c.SetPagesDomain(ctx, "ana", "site", "meusite.com.br"); err != nil || cname != "meusite.com.br" {
		t.Fatalf("set: %v %v", cname, err)
	}
	if h, err := c.PagesHealth(ctx, "ana", "site"); err != nil || !h.Pending {
		t.Fatalf("first health must be pending: %+v %v", h, err)
	}
	h, err := c.PagesHealth(ctx, "ana", "site")
	if err != nil || !h.Ready || !h.Apex || !h.HTTPSEligible {
		t.Fatalf("health: %+v %v", h, err)
	}
	if err := c.EnforceHTTPS(ctx, "ana", "site"); err != nil || !enforced {
		t.Fatal("https not enforced", err)
	}
	d, err := c.GetPagesDomain(ctx, "ana", "site")
	if err != nil || d.CNAME != "meusite.com.br" || !d.HTTPSEnforced || d.CertState != "approved" {
		t.Fatalf("get: %+v %v", d, err)
	}
	if err := c.SetPagesDomain(ctx, "ana", "site", ""); err != nil || cname != nil {
		t.Fatalf("remove: %v %v", cname, err)
	}
}
