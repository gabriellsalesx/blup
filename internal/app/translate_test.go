package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

func TestTranslateNote(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	creds, _ := config.LoadCredentials()
	creds.OpenRouterToken = "k"
	if err := creds.Save(); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Messages []struct{ Content string } }
		_ = json.NewDecoder(r.Body).Decode(&req)
		text := "EN(" + req.Messages[1].Content + ")"
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": text}, "finish_reason": "stop"}}})
	}))
	defer srv.Close()
	openRouterURL = srv.URL
	defer func() { openRouterURL = "" }()

	site := settingsSite(t)
	site.Lang = "pt"
	root := site.ContentDir()
	write(t, root, "Posts/Bolo.md", "---\ndescription: Um bolo\naudio: \"[[bolo.mp3]]\"\ntags: [doce]\n---\nTexto do bolo.\n")

	got, err := TranslateNote(context.Background(), site, "Posts/Bolo.md", "en", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Posts/en/Bolo.md" {
		t.Fatalf("path = %s", got)
	}
	n, _ := obsidian.ReadNote(filepath.Join(root, "Posts", "en", "Bolo.md"))
	if n.String("title") != "EN(Bolo)" || n.String("description") != "EN(Um bolo)" || strings.TrimSpace(n.Body) != "EN(Texto do bolo.)" || n.Props["audio"] != nil || len(n.List("tags")) != 1 {
		t.Fatalf("translation = %+v", n)
	}
	if site.OtherLangs()[0] != "en" {
		t.Fatalf("site languages = %v", site.Translations)
	}

	langs, err := NoteLangs(site, "Posts/en/Bolo.md")
	if err != nil || len(langs) != 3 || !langs[0].Original || langs[0].Path != "Posts/Bolo.md" || !langs[1].Exists || langs[1].Stale || langs[2].Exists {
		t.Fatalf("langs = %+v %v", langs, err)
	}
	// The original changes: the translation is out of date.
	write(t, root, "Posts/Bolo.md", "---\ndescription: Um bolo\n---\nTexto novo.\n")
	if langs, _ := NoteLangs(site, "Posts/Bolo.md"); !langs[1].Stale {
		t.Fatalf("after a change = %+v", langs)
	}
	if _, err := os.Stat(filepath.Join(site.Dir(), "hugo.toml")); err != nil {
		t.Fatalf("hugo.toml not written: %v", err)
	}
}
