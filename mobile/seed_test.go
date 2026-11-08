package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A new phone starts with one welcome note, in the app's language, and
// nothing else.
func TestSeedWritesOneWelcomeNoteInTheLanguage(t *testing.T) {
	defer useLang(appLang())
	for lang, want := range map[string]string{"pt": "Bem-vindo ao Blup.md", "en": "Welcome to Blup.md", "es": "Bienvenido a Blup.md"} {
		useLang(lang)
		dir := filepath.Join(t.TempDir(), "cofre")
		if err := seed(dir); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != want {
			t.Fatalf("%s: got %v, want only %q", lang, entries, want)
		}
		body, _ := os.ReadFile(filepath.Join(dir, want))
		if !strings.HasPrefix(string(body), "---\ncreated: ") || strings.ContainsAny(string(body), "–—") {
			t.Fatalf("%s: unexpected note:\n%s", lang, body)
		}
	}
}
