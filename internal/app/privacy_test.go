package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
)

func TestPrivacyNote(t *testing.T) {
	site := config.NewSite(t.TempDir())
	site.Vault, site.Folder, site.Lang, site.Author = t.TempDir(), "Site", "pt", "Ana"
	site.Cookies = config.Cookies{Banner: true, GoogleAnalytics: "G-AB12CD"}
	site.Deploy.Provider = "cloudflare"

	path, err := writePrivacyNote(site, "ana@x.dev")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "Política de privacidade.md" || privacyNote(site) != path {
		t.Fatalf("note = %s, found %s", path, privacyNote(site))
	}
	b, _ := os.ReadFile(path)
	text := string(b)
	for _, want := range []string{"blup: privacy", "hidden: true", "ana@x.dev", "Ana", "Google Analytics", "YouTube", "Cloudflare Pages", "link **Cookies**"} {
		if !strings.Contains(text, want) {
			t.Errorf("policy misses %q", want)
		}
	}
	for _, not := range []string{"Meta Pixel", "Umami", "Plausible", "GitHub para comentar", "—", "–"} {
		if strings.Contains(text, not) {
			t.Errorf("policy must not mention %q", not)
		}
	}

	// The note is the user's: never rewritten.
	os.WriteFile(path, []byte("---\nblup: privacy\n---\nminha"), 0o644)
	if again, _ := writePrivacyNote(site, ""); again != path {
		t.Fatalf("second write = %s", again)
	}
	if b, _ := os.ReadFile(path); !strings.HasSuffix(string(b), "minha") {
		t.Fatal("an existing policy must not be overwritten")
	}
}

func TestPrivacyBodyWithoutTools(t *testing.T) {
	site := config.NewSite("")
	site.Lang, site.Title = "en", "My Blog"
	text := privacyBody(site, "", time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC))
	for _, want := range []string{"run by My Blog", "[your contact email]", "uses no statistics or ads tools", "[Say where the site is hosted", "October 29, 2026"} {
		if !strings.Contains(text, want) {
			t.Errorf("policy misses %q", want)
		}
	}
	if strings.Contains(text, "**Cookies** link") {
		t.Error("no banner: no Cookies link")
	}
}
