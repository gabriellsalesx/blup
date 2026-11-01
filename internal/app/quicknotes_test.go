package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestFolderTags(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	write(t, root, "Livros/A.md", "---\ntags: [livro, leitura]\n---\n")
	write(t, root, "Livros/B.md", "---\ntags: [livro]\n---\n")
	write(t, root, "Livros/C.md", "sem tags")
	write(t, root, "Solo/Uma.md", "---\ntags: [x]\n---\n")
	if got := FolderTags(site, "Livros"); !slices.Equal(got, []string{"livro"}) {
		t.Fatalf("Livros = %v (livro is in 2 of 3, leitura in 1 of 3)", got)
	}
	if got := FolderTags(site, "Solo"); got != nil {
		t.Fatalf("one note alone decided: %v", got)
	}
	if _, err := CreateDatabase(site, "", "Clientes", DBSchema{}); err != nil {
		t.Fatal(err)
	}
	if got := FolderTags(site, "Bancos de dados/Clientes"); !slices.Equal(got, []string{"clientes"}) {
		t.Fatalf("records = %v", got)
	}
}

func TestDailyNote(t *testing.T) {
	site := settingsSite(t)
	at := time.Date(2026, 11, 1, 9, 5, 0, 0, time.Local)
	rel, err := DailyNote(site, "Diário", "diario", at)
	if err != nil || rel != "Diário/2026-11-01.md" {
		t.Fatalf("daily = %q, %v", rel, err)
	}
	if _, err := DailyNote(site, "Diário", "diario", at.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(site.ContentDir(), rel))
	s := string(b)
	if !strings.Contains(s, "created: 2026-11-01") || !strings.Contains(s, "diario") || !strings.Contains(s, "## 09:05") || !strings.HasSuffix(s, "## 11:05\n\n") {
		t.Fatalf("daily note = %q", s)
	}
}

func TestVaultTags(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	write(t, root, "A.md", "---\ntags: [estudos, \"#hub\"]\n---\n")
	write(t, root, "B.md", "---\ntags: estudos, go\n---\n")
	write(t, root, ".escondida/C.md", "---\ntags: [segredo]\n---\n")
	got, err := VaultTags(site)
	if err != nil || !slices.Equal(got, []string{"estudos", "go", "hub"}) {
		t.Fatalf("tags = %v, %v", got, err)
	}
}
