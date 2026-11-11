package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
)

// Only a site the account no longer lists counts as taken down; a failed
// listing (offline, server error) never makes the Studio forget a site.
func TestBlupSiteGone(t *testing.T) {
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if status != http.StatusOK {
			w.WriteHeader(status)
			w.Write([]byte(`{"code":"INTERNAL"}`))
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"limit": 3, "sites": []map[string]any{{"id": "kept", "slug": "ana", "title": "Ana", "url": "https://ana.getblup.site/"}}})
	}))
	defer srv.Close()
	c := &blupcloud.Client{BaseURL: srv.URL, Token: "t", HTTP: srv.Client()}
	if blupSiteGone(t.Context(), c, "kept") {
		t.Fatal("a listed site is not gone")
	}
	if !blupSiteGone(t.Context(), c, "taken-down") {
		t.Fatal("a site the account no longer lists is gone")
	}
	status = http.StatusInternalServerError
	if blupSiteGone(t.Context(), c, "taken-down") {
		t.Fatal("a failed listing must not forget the site")
	}
}
