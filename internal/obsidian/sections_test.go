package obsidian

import (
	"os"
	"path/filepath"
	"testing"
)

// Every section the Studio can add is found by its blup: mark, whatever
// its folder is called (feedback came after the others and was missed).
func TestSections(t *testing.T) {
	dir := t.TempDir()
	for folder, kind := range map[string]string{"Documentação": "docs", "Pedidos": "feedback", "Notas": ""} {
		if err := os.MkdirAll(filepath.Join(dir, folder), 0o755); err != nil {
			t.Fatal(err)
		}
		text := "---\ntitle: " + folder + "\n---\n"
		if kind != "" {
			text = "---\ntitle: " + folder + "\nblup: " + kind + "\n---\n"
		}
		if err := os.WriteFile(filepath.Join(dir, folder, "_index.md"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := Sections(dir)
	if got["docs"] != "Documentação" || got["feedback"] != "Pedidos" || len(got) != 2 {
		t.Errorf("sections = %v", got)
	}
}
