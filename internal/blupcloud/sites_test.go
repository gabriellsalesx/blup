package blupcloud

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPackSite(t *testing.T) {
	dir := t.TempDir()
	page := strings.Repeat("<p>olá</p>", 200)
	os.MkdirAll(filepath.Join(dir, "posts", "oi"), 0o755)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(page), 0o644)
	os.WriteFile(filepath.Join(dir, "posts", "oi", "index.html"), []byte("<p>x</p>"), 0o644)
	os.WriteFile(filepath.Join(dir, "foto.webp"), bytes.Repeat([]byte{1, 2, 3}, 500), 0o644)
	os.WriteFile(filepath.Join(dir, "_headers"), []byte("/*\n  X: y"), 0o644)
	os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("x"), 0o644)

	files, err := PackSite(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]SiteFile{}
	for _, f := range files {
		got[f.Path] = f
	}
	if len(got) != 3 || got["/_headers"].Path != "" {
		t.Fatalf("files = %v", files)
	}
	home := got["/index.html"]
	if !home.GZ || home.Type != "text/html; charset=utf-8" {
		t.Fatalf("home = %+v", home)
	}
	// The hash is of what is stored (gzipped), which opens to the page.
	sum := sha256.Sum256(home.data)
	r, _ := gzip.NewReader(bytes.NewReader(home.data))
	plain, _ := io.ReadAll(r)
	if hex.EncodeToString(sum[:]) != home.Hash || string(plain) != page {
		t.Fatal("stored copy doesn't match")
	}
	if small := got["/posts/oi/index.html"]; small.GZ {
		t.Fatal("a tiny file stays as it is")
	}
	if img := got["/foto.webp"]; img.GZ || img.Type != "image/webp" {
		t.Fatalf("image = %+v", img)
	}
}

func TestDeploySite(t *testing.T) {
	var mu sync.Mutex
	stored := map[string][]byte{"already": nil}
	var deployed []SiteFile
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/missing"):
			var in struct{ Hashes []string }
			json.NewDecoder(r.Body).Decode(&in)
			missing := []string{}
			for _, h := range in.Hashes {
				if _, ok := stored[h]; !ok {
					missing = append(missing, h)
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"missing": missing})
		case r.Method == http.MethodPut:
			b, _ := io.ReadAll(r.Body)
			sum := sha256.Sum256(b)
			if h := filepath.Base(r.URL.Path); h != hex.EncodeToString(sum[:]) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			stored[filepath.Base(r.URL.Path)] = b
		case strings.HasSuffix(r.URL.Path, "/deploys"):
			var in struct{ Files []SiteFile }
			json.NewDecoder(r.Body).Decode(&in)
			deployed = in.Files
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"url": "https://ana.getblup.site/"})
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(strings.Repeat("<p>oi</p>", 100)), 0o644)
	os.WriteFile(filepath.Join(dir, "a.css"), []byte("body{color:red}"), 0o644)
	files, err := PackSite(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := &Client{BaseURL: srv.URL, Token: "t", HTTP: srv.Client()}
	res, err := c.DeploySite(context.Background(), "s1", files)
	if err != nil {
		t.Fatal(err)
	}
	if res.URL != "https://ana.getblup.site/" || res.Uploaded != 2 || len(deployed) != 2 || !deployed[0].GZ && !deployed[1].GZ {
		t.Fatalf("res = %+v, deployed = %+v", res, deployed)
	}
	// Again: nothing new goes up.
	if res, _ = c.DeploySite(context.Background(), "s1", files); res.Uploaded != 0 || res.Skipped != 2 {
		t.Fatalf("second = %+v", res)
	}
}
