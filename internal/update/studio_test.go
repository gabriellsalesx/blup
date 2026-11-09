package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// studioRelease describes the fake release served by fakeStudioRelease.
type studioRelease struct {
	version string
	body    []byte
	signer  ed25519.PrivateKey
	noSig   bool
	badSum  bool // the signed list carries another hash
	tamper  bool // the file served is not the one listed
}

func fakeStudioRelease(t *testing.T, r studioRelease) *httptest.Server {
	t.Helper()
	if r.signer == nil {
		r.signer = testPriv
	}
	name, _ := StudioAsset(StudioAppImage, r.version)
	sum := sha256.Sum256(r.body)
	sumHex := hex.EncodeToString(sum[:])
	if r.badSum {
		sumHex = hex.EncodeToString(make([]byte, 32))
	}
	sums := fmt.Sprintf("%s  %s\n", sumHex, name)
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(r.signer, []byte(sums)))
	served := r.body
	if r.tamper {
		served = append([]byte("evil"), r.body...)
	}
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		sigAsset := ""
		if !r.noSig {
			sigAsset = fmt.Sprintf(`,{"name":%q,"browser_download_url":%q}`, studioSigName, srv.URL+"/dl/sig")
		}
		fmt.Fprintf(w, `{"tag_name":"v%s","assets":[{"name":%q,"browser_download_url":%q},{"name":%q,"browser_download_url":%q}%s]}`,
			r.version, name, srv.URL+"/dl/file", studioSumsName, srv.URL+"/dl/sums", sigAsset)
	})
	mux.HandleFunc("/dl/file", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(served)))
		w.Write(served)
	})
	mux.HandleFunc("/dl/sums", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, sums) })
	mux.HandleFunc("/dl/sig", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, sig) })
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func downloadStudio(t *testing.T, r studioRelease, current string) (string, string, error) {
	t.Helper()
	srv := fakeStudioRelease(t, r)
	c := client(srv)
	c.Current = current
	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path, err := c.DownloadStudio(context.Background(), rel, StudioAppImage, dir, nil)
	return dir, path, err
}

func TestDownloadStudio(t *testing.T) {
	body := []byte("new studio")
	srv := fakeStudioRelease(t, studioRelease{version: "1.3.0", body: body})
	c := client(srv)
	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var last, total int64
	path, err := c.DownloadStudio(context.Background(), rel, StudioAppImage, dir, func(d, tot int64) { last, total = d, tot })
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "blup-studio_1.3.0_x86_64.AppImage" {
		t.Fatalf("file name %s", path)
	}
	if b, _ := os.ReadFile(path); string(b) != string(body) {
		t.Fatal("downloaded file differs")
	}
	if last != int64(len(body)) || total != int64(len(body)) {
		t.Fatalf("progress %d of %d", last, total)
	}
	if left, _ := filepath.Glob(filepath.Join(dir, ".blup-studio-*")); len(left) != 0 {
		t.Fatalf("partial files left: %v", left)
	}
}

func TestDownloadStudioRefusals(t *testing.T) {
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	cases := map[string]struct {
		r       studioRelease
		current string
		want    error
	}{
		"no signature":   {studioRelease{version: "1.3.0", noSig: true}, "1.2.0", ErrSignature},
		"other key":      {studioRelease{version: "1.3.0", signer: other}, "1.2.0", ErrSignature},
		"wrong checksum": {studioRelease{version: "1.3.0", badSum: true}, "1.2.0", ErrChecksum},
		"tampered file":  {studioRelease{version: "1.3.0", tamper: true}, "1.2.0", ErrChecksum},
		"not newer":      {studioRelease{version: "1.2.0"}, "1.2.0", ErrNotNewer},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.r.body = []byte("new studio")
			dir, path, err := downloadStudio(t, tc.r, tc.current)
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
			if path != "" {
				t.Fatal("a refused download returns no file")
			}
			if left, _ := os.ReadDir(dir); len(left) != 0 {
				t.Fatalf("a refused download leaves nothing behind: %v", left)
			}
		})
	}
}

func TestStudioAsset(t *testing.T) {
	for kind, want := range map[StudioKind]string{
		StudioAppImage: "blup-studio_1.0.5_x86_64.AppImage",
		StudioWindows:  "blup-studio_1.0.5_windows_x64-setup.exe",
		StudioMac:      "blup-studio_1.0.5_macos_universal.dmg",
	} {
		if got, err := StudioAsset(kind, "1.0.5"); err != nil || got != want {
			t.Errorf("%s: %s, %v", kind, got, err)
		}
	}
	if _, err := StudioAsset("", "1.0.5"); !errors.Is(err, ErrNoAsset) {
		t.Fatal("an install that cannot update itself has no asset")
	}
}
