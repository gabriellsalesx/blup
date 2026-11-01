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

// The mark counts only in the properties (as for the theme): a line
// "blup: docs" in the text, as in a note that explains the marks, does not
// make the folder a section. Properties written on Windows (BOM, CRLF) do.
func TestSectionsOnlyProperties(t *testing.T) {
	dir := t.TempDir()
	notes := map[string]string{
		"Guia":    "---\ntitle: Guia\n---\nPara criar a documentação, escreva:\n\nblup: docs\n",
		"Pedidos": "\xef\xbb\xbf---\r\ntitle: Pedidos\r\nblup: feedback\r\n---\r\n",
		"Solta":   "blup: roadmap\n",
	}
	for folder, text := range notes {
		if err := os.MkdirAll(filepath.Join(dir, folder), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, folder, "_index.md"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := Sections(dir)
	if got["feedback"] != "Pedidos" || len(got) != 1 {
		t.Errorf("sections = %v", got)
	}
}
