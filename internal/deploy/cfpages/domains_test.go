package cfpages

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDomains(t *testing.T) {
	domains := map[string]string{} // name -> status
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base := "/accounts/acc/pages/projects/blog/domains"
		ok := func(v any) { json.NewEncoder(w).Encode(map[string]any{"success": true, "result": v}) }
		switch {
		case r.Method == http.MethodPost && r.URL.Path == base:
			var b struct{ Name string }
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &b)
			if _, dup := domains[b.Name]; dup {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []any{map[string]any{"code": 8000018, "message": "already added"}}})
				return
			}
			domains[b.Name] = "pending"
			ok(map[string]any{"name": b.Name, "status": "pending", "verification_data": map[string]any{"status": "pending", "error_message": "CNAME record not set"}})
		case strings.HasPrefix(r.URL.Path, base+"/"):
			name := strings.TrimPrefix(r.URL.Path, base+"/")
			st, found := domains[name]
			if !found {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []any{map[string]any{"code": 8000007, "message": "not found"}}})
				return
			}
			if r.Method == http.MethodDelete {
				delete(domains, name)
				ok(nil)
				return
			}
			ok(map[string]any{"name": name, "status": st, "zone_tag": "z1"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := &Client{Token: "t", BaseURL: srv.URL, HTTP: srv.Client()}
	ctx := context.Background()

	d, err := c.AddDomain(ctx, "acc", "blog", "blog.exemplo.com")
	if err != nil || d.Active() || d.Message != "CNAME record not set" {
		t.Fatalf("add: %+v %v", d, err)
	}
	// Adding again returns the existing domain instead of failing.
	domains["blog.exemplo.com"] = "active"
	d, err = c.AddDomain(ctx, "acc", "blog", "blog.exemplo.com")
	if err != nil || !d.Active() || !d.OnCloudflare {
		t.Fatalf("add again: %+v %v", d, err)
	}
	if err := c.RemoveDomain(ctx, "acc", "blog", "blog.exemplo.com"); err != nil {
		t.Fatal(err)
	}
	if err := c.RemoveDomain(ctx, "acc", "blog", "blog.exemplo.com"); err != nil {
		t.Fatalf("removing a missing domain must be fine: %v", err)
	}
}
