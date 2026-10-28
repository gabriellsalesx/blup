package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.3.0", "1.2.9", true},
		{"v1.10.0", "1.9.0", true},
		{"1.2.0", "1.2.0", false},
		{"1.2.0", "1.3.0", false},
		{"2.0.0", "dev", false},
		{"1.2.0-rc1", "1.1.0", true},
	}
	for _, c := range cases {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

// fakeRelease serves /repos/.../releases/latest plus the archive and the
// checksums file for version, containing an executable with content body.
func fakeRelease(t *testing.T, version string, body []byte, badSum bool) *httptest.Server {
	t.Helper()
	name := AssetName(version)
	archive := makeArchive(t, name, body)
	sum := sha256.Sum256(archive)
	sumHex := hex.EncodeToString(sum[:])
	if badSum {
		sumHex = hex.EncodeToString(make([]byte, 32))
	}
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v%s","body":"- Mochi: novidade","html_url":"https://example.org/r","assets":[
			{"name":%q,"browser_download_url":%q},
			{"name":"checksums.txt","browser_download_url":%q}]}`,
			version, name, srv.URL+"/dl/"+name, srv.URL+"/dl/checksums.txt")
	})
	mux.HandleFunc("/dl/"+name, func(w http.ResponseWriter, r *http.Request) { w.Write(archive) })
	mux.HandleFunc("/dl/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  %s\n%s  other.tar.gz\n", sumHex, name, hex.EncodeToString(make([]byte, 32)))
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func makeArchive(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	exe := "blup"
	if runtime.GOOS == "windows" {
		exe = "blup.exe"
	}
	var buf bytes.Buffer
	if filepath.Ext(name) == ".zip" {
		zw := zip.NewWriter(&buf)
		w, _ := zw.Create(exe)
		w.Write(body)
		zw.Close()
		return buf.Bytes()
	}
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "README.md", Mode: 0o644, Size: 2, Typeflag: tar.TypeReg})
	tw.Write([]byte("hi"))
	tw.WriteHeader(&tar.Header{Name: exe, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
	tw.Write(body)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestApplyAndRollback(t *testing.T) {
	srv := fakeRelease(t, "1.3.0", []byte("NEW BINARY"), false)
	c := &Client{BaseURL: srv.URL}
	rel, err := c.Latest(context.Background())
	if err != nil || rel.Version != "1.3.0" || rel.Notes != "- Mochi: novidade" {
		t.Fatalf("latest: %+v %v", rel, err)
	}

	exe := filepath.Join(t.TempDir(), "blup")
	os.WriteFile(exe, []byte("OLD BINARY"), 0o755)
	if err := c.Apply(context.Background(), rel, exe); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "NEW BINARY" {
		t.Fatalf("exe = %q", b)
	}
	if st, _ := os.Stat(exe); runtime.GOOS != "windows" && st.Mode().Perm()&0o100 == 0 {
		t.Error("new executable is not executable")
	}
	if b, _ := os.ReadFile(exe + ".old"); string(b) != "OLD BINARY" {
		t.Fatalf("backup = %q", b)
	}
	if err := Rollback(exe); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "OLD BINARY" {
		t.Fatalf("after rollback exe = %q", b)
	}
}

func TestApplyBadChecksumKeepsOriginal(t *testing.T) {
	srv := fakeRelease(t, "1.3.0", []byte("EVIL"), true)
	c := &Client{BaseURL: srv.URL}
	rel, _ := c.Latest(context.Background())
	exe := filepath.Join(t.TempDir(), "blup")
	os.WriteFile(exe, []byte("OLD"), 0o755)
	if err := c.Apply(context.Background(), rel, exe); !errors.Is(err, ErrChecksum) {
		t.Fatalf("want ErrChecksum, got %v", err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "OLD" {
		t.Fatal("original executable must be untouched")
	}
}

func TestRollbackWithoutBackup(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "blup")
	os.WriteFile(exe, []byte("X"), 0o755)
	if err := Rollback(exe); !errors.Is(err, ErrNoBackup) {
		t.Fatalf("want ErrNoBackup, got %v", err)
	}
}

func TestCacheRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir()) // macOS cache dir
	c := Cache{Latest: "1.3.0", Skipped: "1.2.0"}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	if got := LoadCache(); got.Latest != "1.3.0" || got.Skipped != "1.2.0" {
		t.Fatalf("cache = %+v", got)
	}
}
