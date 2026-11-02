package blupcloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const testSite = "6f1c1f9e-1b2a-4c3d-9e8f-0a1b2c3d4e5f"

func TestValidHostname(t *testing.T) {
	good := []string{"meusite.com.br", "blog.meusite.com", "xn--caf-dma.com", "a-b.io", "www.example.org"}
	bad := []string{
		"", "localhost", "meusite", "https://meusite.com", "meusite.com/", "meusite.com:443", "user@meusite.com",
		"MEUSITE.com", "meu site.com", "-x.com", "x-.com", "x..com", ".x.com", "x.com.", "127.0.0.1", "[::1]", "::1",
		"1.2.3.4", "getblup.site", "nome.getblup.site", "a.b.getblup.io", "getblup.io", "x.localhost", "printer.local",
		"svc.internal", "café.com", strings.Repeat("a", 64) + ".com", strings.Repeat("a.", 127) + "com", "x.com?y", "x.com#y",
		"a.123",
	}
	for _, h := range good {
		if !ValidHostname(h) {
			t.Errorf("%q refused", h)
		}
	}
	for _, h := range bad {
		if ValidHostname(h) {
			t.Errorf("%q accepted", h)
		}
	}
}

// The calls follow the cloud's contract: path, method, body, Bearer token,
// and each error code becomes its error.
func TestDomainAPI(t *testing.T) {
	var linked *DomainInfo
	var hits atomic.Int32
	code, status := "", 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path != "/api/sites/"+testSite+"/domain" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if code != "" {
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]string{"code": code, "message": "x"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(linked)
		case http.MethodPost:
			var in map[string]any
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in) != 1 {
				t.Errorf("body %v", in)
			}
			linked = &DomainInfo{Hostname: in["hostname"].(string), Status: "pending_ownership", SSL: "pending", Records: []DomainRecord{
				{Type: "TXT", Name: "_blup." + in["hostname"].(string), Value: "blup-verify=abc"},
				{Type: "CNAME", Name: in["hostname"].(string), Value: "fallback.getblup.site"},
				{Type: "A", Name: "@", Value: "1.2.3.4"}, // not in the contract: dropped
			}}
			json.NewEncoder(w).Encode(linked)
		case http.MethodDelete:
			linked = nil
			json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		}
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}
	ctx := context.Background()

	if d, err := c.Domain(ctx, testSite); err != nil || d != nil {
		t.Fatalf("no domain: %v %v", d, err)
	}
	d, err := c.SetDomain(ctx, testSite, "blog.meusite.com.br")
	if err != nil || d.Status != "pending_ownership" || len(d.Records) != 2 || d.Records[0].Type != "TXT" {
		t.Fatalf("set: %+v %v", d, err)
	}
	if got, err := c.Domain(ctx, testSite); err != nil || got == nil || got.Hostname != "blog.meusite.com.br" {
		t.Fatalf("get: %+v %v", got, err)
	}
	if err := c.RemoveDomain(ctx, testSite); err != nil || linked != nil {
		t.Fatalf("remove: %v", err)
	}

	// Refused here: nothing is sent.
	before := hits.Load()
	for _, h := range []string{"https://x.com", "nome.getblup.site", "localhost", "10.0.0.1"} {
		if _, err := c.SetDomain(ctx, testSite, h); !errors.Is(err, ErrBadHostname) {
			t.Errorf("%q: %v", h, err)
		}
	}
	if _, err := c.SetDomain(ctx, "../../admin", "x.com"); !errors.Is(err, ErrDomainSiteMissing) {
		t.Errorf("bad site id: %v", err)
	}
	if hits.Load() != before {
		t.Fatal("a refused hostname or site id went out")
	}

	for _, tc := range []struct {
		code   string
		status int
		want   error
	}{
		{"INVALID", 400, ErrDomainInvalid},
		{"PLAN", 403, ErrDomainPlan},
		{"NOT_FOUND", 404, ErrDomainSiteMissing},
		{"DOMAIN_TAKEN", 409, ErrDomainTaken},
		{"DOMAIN_EXISTS", 409, ErrDomainExists},
		{"TOO_MANY_REQUESTS", 429, ErrDomainRate},
		{"DOMAIN_PROVIDER", 502, ErrDomainProvider},
		{"DOMAIN_UNAVAILABLE", 503, ErrDomainUnavailable},
	} {
		code, status = tc.code, tc.status
		if _, err := c.SetDomain(ctx, testSite, "x.com"); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v", tc.code, err)
		}
	}
	code = ""
	c.Token = "other"
	if _, err := c.Domain(ctx, testSite); !errors.Is(err, ErrSignedOut) {
		t.Errorf("401: %v", err)
	}
}

// An answer that is not the contract's (unknown status, a hostname that
// could not be one) is not shown.
func TestDomainAnswerChecked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"hostname":"<img src=x>.com","status":"active","ssl":"active","records":[]}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}
	if _, err := c.Domain(context.Background(), testSite); err == nil {
		t.Fatal("odd answer accepted")
	}
}
