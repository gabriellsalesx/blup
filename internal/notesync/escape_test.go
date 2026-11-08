package notesync

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Ataque: quem escreve no destino (um colaborador do repositório, ou um
// aparelho com a chave) põe "link/evil.md" e, neste computador, "link" é um
// atalho para uma pasta de fora. O sync não escreve através do atalho, e
// nomes com ".." ou absolutos nunca saem da pasta das notas.
func TestRemoteNeverWritesOutsideDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	f, _ := setup(t)
	a := newComputer(t, f, "notas")
	a.write(t, map[string]string{"a.md": "A\n"})
	a.sync(t)

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(a.dir, "link")); err != nil {
		t.Fatal(err)
	}
	f.push("notas", map[string]string{
		"link/evil.md":  "pwned\n",
		"link/sub/x.md": "pwned\n",
		"ok/fine.md":    "fine\n",
		"../escape.md":  "pwned\n",
		"a/../../e2.md": "pwned\n",
	})
	rep := a.sync(t)

	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatalf("written through the link: %v", entries)
	}
	for _, p := range []string{"escape.md", "e2.md"} {
		if _, err := os.Stat(filepath.Join(filepath.Dir(a.dir), p)); err == nil {
			t.Fatalf("%s written outside the notes folder", p)
		}
	}
	if b, err := os.ReadFile(filepath.Join(a.dir, "ok", "fine.md")); err != nil || string(b) != "fine\n" {
		t.Fatalf("a normal file must still arrive: %q %v", b, err)
	}
	for _, p := range []string{"link/evil.md", "link/sub/x.md"} {
		if rep.SkipReasons[p] != SkipInvalidName {
			t.Fatalf("%s: reason %q (report %+v)", p, rep.SkipReasons[p], rep)
		}
	}
}
