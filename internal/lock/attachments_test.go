package lock

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func write(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A protected post's attachments go inside it (quoted or bare attributes,
// as --minify writes them, and the site's own path), and Prune deletes the
// files only it used; a file a public page uses stays.
func TestInlineAndPrune(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "vault/Anexos/so-trancado_hu_1.webp", "W480")
	write(t, dir, "vault/Anexos/so-trancado_hu_2.webp", "W800")
	write(t, dir, "vault/Anexos/doc secreto.pdf", "PDF")
	write(t, dir, "vault/Anexos/publica.png", "PUB")
	write(t, dir, "vault/Comentario/so-comentario.gif", "GIF")
	write(t, dir, "vault/Anexos/notas.md", "not an attachment: never touched")
	write(t, dir, "posts/aberto/index.html", `<p><img src=/repo/vault/Anexos/publica.png alt=x></p>`)
	write(t, dir, Manifest, `{"abc":"senha"}`)
	write(t, dir, "posts/trancado/index.html", `<main><blup-begin data-id=abc></blup-begin>`+
		`<img src=/repo/vault/Anexos/so-trancado_hu_1.webp srcset="/repo/vault/Anexos/so-trancado_hu_1.webp 480w, /repo/vault/Anexos/so-trancado_hu_2.webp 800w" sizes="100vw" alt=y>`+
		`<a href="https://example.org/repo/vault/Anexos/doc%20secreto.pdf">doc</a>`+
		`<img src='/repo/vault/Anexos/publica.png'>`+
		`<blup-end></blup-end></main>`)

	if n, err := Process(dir, ""); err != nil || n != 1 {
		t.Fatalf("process: %d %v", n, err)
	}
	page, _ := os.ReadFile(filepath.Join(dir, "posts/trancado/index.html"))
	plain := decryptRegion(t, string(page), "senha")
	for _, want := range []string{
		`src="data:image/webp;base64,` + b64("W800") + `"`,
		`href="data:application/pdf;base64,` + b64("PDF") + `" download="doc secreto.pdf"`,
		`src="data:image/png;base64,` + b64("PUB") + `"`,
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing %s in\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "srcset") || strings.Contains(plain, "/vault/") {
		t.Errorf("left references:\n%s", plain)
	}

	if n, err := Prune(dir); err != nil || n != 4 {
		t.Fatalf("prune: %d %v", n, err)
	}
	for rel, kept := range map[string]bool{
		"vault/Anexos/publica.png": true, "vault/Anexos/notas.md": true,
		"vault/Anexos/so-trancado_hu_1.webp": false, "vault/Anexos/doc secreto.pdf": false,
		"vault/Comentario": false,
	} {
		_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if (err == nil) != kept {
			t.Errorf("%s: kept=%v, want %v", rel, err == nil, kept)
		}
	}
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// decryptRegion opens the encrypted region of a page, as lock.js does.
func decryptRegion(t *testing.T, page, password string) string {
	t.Helper()
	attr := func(name string) []byte {
		m := regexp.MustCompile(`data-` + name + `="([^"]+)"`).FindStringSubmatch(page)
		if m == nil {
			t.Fatalf("no data-%s in %s", name, page)
		}
		b, err := base64.StdEncoding.DecodeString(m[1])
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	plain, err := Decrypt(password, Sealed{Salt: attr("salt"), IV: attr("iv"), Ciphertext: attr("ct"), Iterations: Iterations})
	if err != nil {
		t.Fatal(err)
	}
	return string(plain)
}

// Fail closed: a page with protected markers is never published in clear,
// whatever the manifest says (missing, empty, null, not an object).
func TestMarkersWithoutManifest(t *testing.T) {
	body := `<p>x</p><blup-begin data-id="abc"></blup-begin><p>SEGREDO</p><blup-end></blup-end>`
	for name, manifest := range map[string]string{"missing": "", "empty object": "{}", "null": "null", "list": `["abc"]`, "number": `{"abc": 1}`, "lone end": "-"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			page := body
			if manifest == "-" {
				page = `<p>SEGREDO</p><blup-end></blup-end>`
				manifest = "{}"
			}
			write(t, dir, "p/index.html", page)
			if manifest != "" {
				write(t, dir, Manifest, manifest)
			}
			posts, _, perr := Pending(dir)
			_, err := Process(dir, "")
			if err == nil {
				t.Fatalf("process accepted it (pending %d, %v)", posts, perr)
			}
			if manifest == "{}" && !errors.Is(err, ErrLeftover) {
				t.Errorf("want ErrLeftover, got %v", err)
			}
			if err := Check(dir); !errors.Is(err, ErrLeftover) {
				t.Errorf("check: %v", err)
			}
		})
	}
	// null, a list or other values are not "nothing to encrypt".
	for _, bad := range []string{"null", "[]", `{"a":1}`} {
		dir := t.TempDir()
		write(t, dir, Manifest, bad)
		if _, _, err := Pending(dir); err == nil {
			t.Errorf("Pending accepted %s", bad)
		}
	}
	clean := t.TempDir()
	write(t, clean, "p/index.html", "<p>ok</p>")
	if err := Check(clean); err != nil {
		t.Fatal(err)
	}
}
