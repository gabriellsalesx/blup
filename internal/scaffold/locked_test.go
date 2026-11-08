package scaffold

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/hugo"
	"github.com/gabriellsalesx/blup/internal/lock"
)

// TestBuildLockedTranslated: a protected post with a translation that has
// its own password, a description and a narration, and a protected plain
// page. After encrypting, no
// file of the site has the text, the description, the narration or a
// password, and each language opens with its own password.
func TestBuildLockedTranslated(t *testing.T) {
	bin, ok := hugo.Find(context.Background())
	if !ok {
		t.Skip("hugo not installed")
	}
	mp3 := []byte("ID3NARRACAO-SECRETA")
	for _, model := range []string{"mochi", "nori", "bento"} {
		t.Run(model, func(t *testing.T) {
			vault := t.TempDir()
			folder := "Blog"
			if model == "bento" {
				folder = "Docs"
			}
			files := map[string]string{
				folder + "/Segredo.md":    "---\ntitle: Titulo publico\npassword: senha-pt\ndescription: DESCRICAO-SECRETA\naudio: \"[[narra.mp3]]\"\n---\nTEXTO-SECRETO-PT\n",
				folder + "/en/Segredo.md": "---\ntitle: Public title\npassword: senha-en\ndescription: SECRET-DESCRIPTION\n---\nTEXTO-SECRETO-EN\n",
				folder + "/Aberto.md":     "---\ntitle: Aberto\n---\nTexto aberto.\n",
				// A plain page (the privacy policy) can be protected too.
				folder + "/Privacidade.md": "---\ntitle: Privacidade\nblup: privacy\npassword: senha-pt\ndescription: DESCRICAO-SECRETA\n---\nTEXTO-SECRETO-PT\n",
				"Anexos/narra.mp3":         string(mp3),
			}
			for rel, text := range files {
				p := filepath.Join(vault, filepath.FromSlash(rel))
				os.MkdirAll(filepath.Dir(p), 0o755)
				if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			dir := t.TempDir()
			if d := os.Getenv("BLUP_KEEP"); d != "" {
				dir = filepath.Join(d, model)
			}
			s := config.NewSite(dir)
			s.Title, s.Lang, s.Model, s.Vault = "Site", "pt", model, vault
			if model != "bento" {
				s.Folder = folder
			}
			s.Translations = []string{"en"}
			if err := Write(s); err != nil {
				t.Fatal(err)
			}
			cmd := bin.Command(context.Background(), dir, "--minify", "--baseURL", "https://example.org/", "--logLevel", "warn")
			cmd.Env = append(os.Environ(), "HUGO_BLUP_MODE=publish")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("hugo: %v\n%s", err, out)
			}
			public := filepath.Join(dir, "public")
			n, err := lock.Process(public, "")
			if err != nil || n != 3 {
				t.Fatalf("lock: %d %v", n, err)
			}
			if err := lock.Check(public); err != nil {
				t.Fatal(err)
			}
			if _, err := lock.Prune(public); err != nil {
				t.Fatal(err)
			}
			cipher := regexp.MustCompile(`data-salt="?([^" >]+)"? data-iv="?([^" >]+)"? data-iter="?\d+"? data-ct="?([^" >]+)`)
			opened := map[string]bool{}
			filepath.WalkDir(public, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				b, _ := os.ReadFile(p)
				for _, secret := range []string{"TEXTO-SECRETO", "DESCRICAO-SECRETA", "SECRET-DESCRIPTION", "NARRACAO-SECRETA", "senha-pt", "senha-en", "blup-begin"} {
					if strings.Contains(string(b), secret) {
						t.Errorf("%s leaks %q", p, secret)
					}
				}
				for _, m := range cipher.FindAllStringSubmatch(string(b), -1) {
					dec := func(s string) []byte { v, _ := base64.StdEncoding.DecodeString(s); return v }
					sealed := lock.Sealed{Salt: dec(m[1]), IV: dec(m[2]), Ciphertext: dec(m[3]), Iterations: lock.Iterations}
					for _, pw := range []string{"senha-pt", "senha-en"} {
						if plain, err := lock.Decrypt(pw, sealed); err == nil {
							if want := map[string]string{"senha-pt": "TEXTO-SECRETO-PT", "senha-en": "TEXTO-SECRETO-EN"}[pw]; !strings.Contains(string(plain), want) {
								t.Errorf("%s: %s opens the wrong text", p, pw)
							}
							opened[pw] = true
						}
					}
				}
				return nil
			})
			if !opened["senha-pt"] || !opened["senha-en"] {
				t.Errorf("each language must open with its own password: %v", opened)
			}
		})
	}
}
