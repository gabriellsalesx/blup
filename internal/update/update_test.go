package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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

// release describes the fake GitHub release served by fakeRelease.
type release struct {
	version    string
	body       []byte
	badSum     bool               // checksums.txt lists a wrong hash (still signed)
	noSig      bool               // no checksums.txt.sig asset
	signer     ed25519.PrivateKey // key that signs checksums.txt
	tamperSums bool               // checksums.txt changed after signing
}

// testKey is the key the test client trusts.
var testPub, testPriv, _ = ed25519.GenerateKey(rand.Reader)

// fakeRelease serves /repos/.../releases/latest plus the archive, the
// checksums file and its signature.
func fakeRelease(t *testing.T, r release) *httptest.Server {
	t.Helper()
	if r.signer == nil {
		r.signer = testPriv
	}
	name := AssetName(r.version)
	archive := makeArchive(t, name, r.body)
	sum := sha256.Sum256(archive)
	sumHex := hex.EncodeToString(sum[:])
	if r.badSum {
		sumHex = hex.EncodeToString(make([]byte, 32))
	}
	sums := fmt.Sprintf("%s  %s\n%s  other.tar.gz\n", sumHex, name, hex.EncodeToString(make([]byte, 32)))
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(r.signer, []byte(sums))) + "\n"
	if r.tamperSums {
		evil := sha256.Sum256([]byte("evil"))
		sums = fmt.Sprintf("%s  %s\n", hex.EncodeToString(evil[:]), name)
	}
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		sigAsset := ""
		if !r.noSig {
			sigAsset = fmt.Sprintf(`,{"name":"checksums.txt.sig","browser_download_url":%q}`, srv.URL+"/dl/checksums.txt.sig")
		}
		fmt.Fprintf(w, `{"tag_name":"v%s","body":"- Mochi: novidade","html_url":"https://example.org/r","assets":[
			{"name":%q,"browser_download_url":%q},
			{"name":"checksums.txt","browser_download_url":%q}%s]}`,
			r.version, name, srv.URL+"/dl/"+name, srv.URL+"/dl/checksums.txt", sigAsset)
	})
	mux.HandleFunc("/dl/"+name, func(w http.ResponseWriter, _ *http.Request) { w.Write(archive) })
	mux.HandleFunc("/dl/checksums.txt", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, sums) })
	mux.HandleFunc("/dl/checksums.txt.sig", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, sig) })
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// client is a test client running version 1.2.0 that trusts testPub.
func client(srv *httptest.Server) *Client {
	return &Client{BaseURL: srv.URL, Current: "1.2.0", Keys: []ed25519.PublicKey{testPub}}
}

// applyRefused runs Apply against r and checks it fails with want and
// leaves the executable untouched.
func applyRefused(t *testing.T, r release, current string, want error) {
	t.Helper()
	srv := fakeRelease(t, r)
	c := client(srv)
	c.Current = current
	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(t.TempDir(), "blup")
	os.WriteFile(exe, []byte("OLD"), 0o755)
	if err := c.Apply(context.Background(), rel, exe); !errors.Is(err, want) {
		t.Fatalf("want %v, got %v", want, err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "OLD" {
		t.Fatal("original executable must be untouched")
	}
	if _, err := os.Stat(exe + ".old"); err == nil {
		t.Fatal("no backup should be made when the update is refused")
	}
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
	srv := fakeRelease(t, release{version: "1.3.0", body: []byte("NEW BINARY")})
	c := client(srv)
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
	applyRefused(t, release{version: "1.3.0", body: []byte("EVIL"), badSum: true}, "1.2.0", ErrChecksum)
}

func TestApplyRefusesMissingSignature(t *testing.T) {
	applyRefused(t, release{version: "1.3.0", body: []byte("EVIL"), noSig: true}, "1.2.0", ErrSignature)
}

func TestApplyRefusesOtherKey(t *testing.T) {
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	applyRefused(t, release{version: "1.3.0", body: []byte("EVIL"), signer: other}, "1.2.0", ErrSignature)
}

func TestApplyRefusesTamperedChecksums(t *testing.T) {
	applyRefused(t, release{version: "1.3.0", body: []byte("EVIL"), tamperSums: true}, "1.2.0", ErrSignature)
}

func TestApplyRefusesDowngrade(t *testing.T) {
	applyRefused(t, release{version: "1.1.0", body: []byte("OLD AND BUGGY")}, "1.2.0", ErrNotNewer)
	applyRefused(t, release{version: "1.2.0", body: []byte("SAME")}, "1.2.0", ErrNotNewer)
	// A client that does not say what it runs updates nothing.
	applyRefused(t, release{version: "1.3.0", body: []byte("NEW")}, "", ErrNotNewer)
	applyRefused(t, release{version: "1.3.0", body: []byte("NEW")}, "dev", ErrNotNewer)
}

func TestVerifyChecksums(t *testing.T) {
	sums := []byte("abc  blup_1.0.0_linux_amd64.tar.gz\n")
	good := base64.StdEncoding.EncodeToString(ed25519.Sign(testPriv, sums))
	keys := []ed25519.PublicKey{testPub}
	if err := VerifyChecksums(sums, []byte(good+"\n"), keys); err != nil {
		t.Fatalf("valid signature refused: %v", err)
	}
	for name, sig := range map[string]string{"empty": "", "garbage": "not base64!", "short": "AAAA"} {
		if err := VerifyChecksums(sums, []byte(sig), keys); !errors.Is(err, ErrSignature) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if err := VerifyChecksums(sums, []byte(good), nil); !errors.Is(err, ErrSignature) {
		t.Error("no trusted keys must refuse")
	}
}

func TestChecksumForRefusesDuplicates(t *testing.T) {
	h := hex.EncodeToString(make([]byte, 32))
	if _, err := checksumFor([]byte(h+"  a\n"+h+"  a\n"), "a"); !errors.Is(err, ErrChecksum) {
		t.Fatalf("duplicate accepted: %v", err)
	}
	if _, err := checksumFor([]byte("zz  a\n"), "a"); !errors.Is(err, ErrChecksum) {
		t.Fatalf("bad hash accepted: %v", err)
	}
}

// The key built into the binary must parse: without it no update works.
func TestBuiltInKey(t *testing.T) {
	if len(PublicKeys()) == 0 || len(PublicKeys()) != len(publicKeys) {
		t.Fatal("a built-in release key does not parse")
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

// A release whose files point at plain HTTP (a tampered API answer) is not
// downloaded, and a redirect off HTTPS is refused.
func TestApplyRefusesPlainHTTP(t *testing.T) {
	name := AssetName("9.9.9")
	rel := Release{Version: "9.9.9", assets: map[string]string{
		name:                "http://evil.example/" + name,
		"checksums.txt":     "https://github.com/x/checksums.txt",
		"checksums.txt.sig": "https://github.com/x/checksums.txt.sig",
	}}
	exe := filepath.Join(t.TempDir(), "blup")
	os.WriteFile(exe, []byte("old"), 0o755)
	if err := (&Client{Current: "1.0.0"}).Apply(context.Background(), rel, exe); !errors.Is(err, ErrChecksum) {
		t.Fatalf("err = %v, want ErrChecksum", err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "old" {
		t.Fatal("the executable changed")
	}
	req, _ := http.NewRequest(http.MethodGet, "http://evil.example/x", nil)
	if httpsOnly(req, nil) == nil {
		t.Fatal("redirect to http allowed")
	}
}
