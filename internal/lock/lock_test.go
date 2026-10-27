package lock

import (
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestEncryptRoundTrip(t *testing.T) {
	s, err := Encrypt("girassol", []byte("<p>segredo</p>"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decrypt("girassol", s)
	if err != nil || string(got) != "<p>segredo</p>" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := Decrypt("errada", s); err == nil {
		t.Fatal("wrong password must fail")
	}
}

// TestBrowserCompatible decrypts our output with WebCrypto in Node, the same
// API the browser uses in theme/assets/js/lock.js.
func TestBrowserCompatible(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	s, err := Encrypt("Ação-123", []byte("<h2>Olá</h2><p>conteúdo protegido</p>"))
	if err != nil {
		t.Fatal(err)
	}
	enc := base64.StdEncoding.EncodeToString
	script := `
const [pw, salt, iv, iter, ct] = process.argv.slice(1);
const b = (s) => Uint8Array.from(atob(s), (c) => c.charCodeAt(0));
const base = await crypto.subtle.importKey("raw", new TextEncoder().encode(pw), "PBKDF2", false, ["deriveKey"]);
const key = await crypto.subtle.deriveKey({ name: "PBKDF2", salt: b(salt), iterations: Number(iter), hash: "SHA-256" }, base, { name: "AES-GCM", length: 256 }, false, ["decrypt"]);
const out = await crypto.subtle.decrypt({ name: "AES-GCM", iv: b(iv) }, key, b(ct));
process.stdout.write(new TextDecoder().decode(out));`
	out, err := exec.Command(node, "--input-type=module", "-e", script, "Ação-123",
		enc(s.Salt), enc(s.IV), "600000", enc(s.Ciphertext)).CombinedOutput()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, out)
	}
	if string(out) != "<h2>Olá</h2><p>conteúdo protegido</p>" {
		t.Fatalf("node decrypted %q", out)
	}
}

func TestProcess(t *testing.T) {
	dir := t.TempDir()
	page := `<main><div data-blup-lock><form></form><blup-begin data-id=abc1></blup-begin><p>SEGREDO-1</p><blup-end></blup-end></div></main>`
	page2 := `<blup-begin data-id="def2"></blup-begin><p>SEGREDO-2</p><blup-end></blup-end>`
	write := func(rel, s string) {
		p := filepath.Join(dir, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(s), 0o644)
	}
	write("a/index.html", page)
	write("b/index.html", page2)
	write("c/index.html", "<p>public</p>")
	write(Manifest, `{"abc1":"senha","def2":"__global__"}`)

	n, needs, err := Pending(dir)
	if err != nil || n != 2 || !needs {
		t.Fatalf("pending: %d %v %v", n, needs, err)
	}
	if _, err := Process(dir, ""); !errors.Is(err, ErrNoGlobalPassword) {
		t.Fatalf("want ErrNoGlobalPassword, got %v", err)
	}
	write(Manifest, `{"abc1":"senha","def2":"__global__"}`)
	n, err = Process(dir, "geral")
	if err != nil || n != 2 {
		t.Fatalf("process: %d %v", n, err)
	}
	if _, err := os.Stat(filepath.Join(dir, Manifest)); !os.IsNotExist(err) {
		t.Fatal("manifest must be deleted")
	}
	for rel, pw := range map[string]string{"a/index.html": "senha", "b/index.html": "geral"} {
		b, _ := os.ReadFile(filepath.Join(dir, rel))
		if strings.Contains(string(b), "SEGREDO") || strings.Contains(string(b), "blup-begin") {
			t.Fatalf("%s still has clear text: %s", rel, b)
		}
		s := parse(t, string(b))
		plain, err := Decrypt(pw, s)
		if err != nil || !strings.Contains(string(plain), "SEGREDO") {
			t.Fatalf("%s: %q %v", rel, plain, err)
		}
	}
}

func TestProcessUnknownID(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.html"), []byte(`<blup-begin data-id="zzz"></blup-begin>S<blup-end></blup-end>`), 0o644)
	os.WriteFile(filepath.Join(dir, Manifest), []byte(`{"abc":"x"}`), 0o644)
	if _, err := Process(dir, ""); !errors.Is(err, ErrLeftover) {
		t.Fatalf("want ErrLeftover, got %v", err)
	}
}

var attr = regexp.MustCompile(`data-(salt|iv|ct)="([^"]+)"`)

func parse(t *testing.T, html string) Sealed {
	t.Helper()
	s := Sealed{Iterations: Iterations}
	for _, m := range attr.FindAllStringSubmatch(html, -1) {
		b, err := base64.StdEncoding.DecodeString(m[2])
		if err != nil {
			t.Fatal(err)
		}
		switch m[1] {
		case "salt":
			s.Salt = b
		case "iv":
			s.IV = b
		case "ct":
			s.Ciphertext = b
		}
	}
	return s
}
