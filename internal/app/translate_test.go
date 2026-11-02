package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	yamlv3 "go.yaml.in/yaml/v3"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

func TestTranslateNote(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	creds, _ := config.LoadCredentials()
	creds.OpenRouterToken = "k"
	_ = SetAIConsent(AIOpenRouter, true)
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

	got, err := TranslateNote(context.Background(), site, "Posts/Bolo.md", "en", AIOpenRouter, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Under its translated title, linked to the original by translationKey.
	if got != "Posts/en/EN(Bolo).md" {
		t.Fatalf("path = %s", got)
	}
	n, _ := obsidian.ReadNote(filepath.Join(root, "Posts", "en", "EN(Bolo).md"))
	o, _ := obsidian.ReadNote(filepath.Join(root, "Posts", "Bolo.md"))
	if o.String("translationKey") != "Posts/Bolo" || n.String("translationKey") != "Posts/Bolo" {
		t.Fatalf("keys = %q %q", o.String("translationKey"), n.String("translationKey"))
	}
	if n.String("title") != "EN(Bolo)" || n.String("description") != "EN(Um bolo)" || strings.TrimSpace(n.Body) != "EN(Texto do bolo.)" || n.Props["audio"] != nil {
		t.Fatalf("translation = %+v", n)
	}
	if n.Props["ai_translated"] != true || n.List("tags")[0] != "EN(doce)" {
		t.Fatalf("ai mark or tags = %+v", n.Props)
	}
	if site.OtherLangs()[0] != "en" {
		t.Fatalf("site languages = %v", site.Translations)
	}

	langs, err := NoteLangs(site, "Posts/en/EN(Bolo).md")
	if err != nil || len(langs) != 3 || !langs[0].Original || langs[0].Path != "Posts/Bolo.md" || langs[1].Path != "Posts/en/EN(Bolo).md" || !langs[1].Exists || langs[1].Stale || langs[2].Exists {
		t.Fatalf("langs = %+v %v", langs, err)
	}
	// The original changes: the translation is out of date.
	write(t, root, "Posts/Bolo.md", "---\ndescription: Um bolo\n---\nTexto novo.\n")
	if langs, _ := NoteLangs(site, "Posts/Bolo.md"); !langs[1].Stale {
		t.Fatalf("after a change = %+v", langs)
	}
	// Translated again: the same file, not a new one.
	if again, err := TranslateNote(context.Background(), site, "Posts/Bolo.md", "en", AIOpenRouter, nil); err != nil || again != got {
		t.Fatalf("again = %s %v", again, err)
	}
	if _, err := os.Stat(filepath.Join(site.Dir(), "hugo.toml")); err != nil {
		t.Fatalf("hugo.toml not written: %v", err)
	}
}

func TestTranslateSite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	creds, _ := config.LoadCredentials()
	creds.OpenRouterToken = "k"
	_ = SetAIConsent(AIOpenRouter, true)
	_ = creds.Save()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "ES"}, "finish_reason": "stop"}}})
	}))
	defer srv.Close()
	openRouterURL = srv.URL
	defer func() { openRouterURL = "" }()

	site := settingsSite(t)
	site.Lang = "pt"
	root := site.ContentDir()
	write(t, root, "A.md", "Um.")
	write(t, root, "Pasta/B.md", "Dois.")
	write(t, root, "AGENTS.md", "guia")
	write(t, root, "en/A.md", "---\ntitle: A\n---\nOne.")

	n, err := TranslateSite(context.Background(), site, "es", AIOpenRouter, nil)
	if err != nil || n.Done != 4 || n.Skipped != 0 { // A, Pasta/B and the test site's _index and Posts/Um
		t.Fatalf("translated %d, %v", n, err)
	}
	if es, _ := filepath.Glob(filepath.Join(root, "Pasta", "es", "*.md")); len(es) != 1 {
		t.Fatalf("Pasta/es = %v", es)
	}
	st, _ := SiteLangs(site)
	for _, s := range st {
		if s.Lang == "es" && (s.Done != 4 || s.Missing != 0 || !s.On) {
			t.Fatalf("es = %+v", s)
		}
	}
	before := calls
	if n, _ := TranslateSite(context.Background(), site, "es", AIOpenRouter, nil); n.Done != 0 || calls != before {
		t.Fatal("up to date notes should not be translated again")
	}
}

// The home note's menu labels and the site's texts are translated; a
// section's note keeps its name.
func TestTranslateHomeAndSection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	creds, _ := config.LoadCredentials()
	creds.OpenRouterToken = "k"
	_ = SetAIConsent(AIOpenRouter, true)
	_ = creds.Save()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Messages []struct{ Content string } }
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "ES(" + req.Messages[1].Content + ")"}, "finish_reason": "stop"}}})
	}))
	defer srv.Close()
	openRouterURL = srv.URL
	defer func() { openRouterURL = "" }()

	site := settingsSite(t)
	root := site.ContentDir()
	write(t, root, "_index.md", "---\ntitle: Início\nsite_description: Notas sobre bolos\nbutton_text: Assine\nmenu:\n  - Posts\n  - Loja | https://x.dev\n---\nOi\n")
	write(t, root, "Posts/_index.md", "---\ntitle: Receitas\nlinkTitle: Doces\n---\n")
	if p, err := TranslateNote(context.Background(), site, "_index.md", "es", AIOpenRouter, nil); err != nil || p != "es/_index.md" {
		t.Fatalf("home = %s %v", p, err)
	}
	h, _ := obsidian.ReadNote(filepath.Join(root, "es", "_index.md"))
	if m := h.List("menu"); len(m) != 2 || m[0] != "Posts" || m[1] != "ES(Loja) | https://x.dev" || h.String("site_description") != "ES(Notas sobre bolos)" || h.String("button_text") != "ES(Assine)" {
		t.Fatalf("home props = %+v", h.Props)
	}
	if p, err := TranslateNote(context.Background(), site, "Posts/_index.md", "es", AIOpenRouter, nil); err != nil || p != "Posts/es/_index.md" {
		t.Fatalf("section = %s %v", p, err)
	}
	// An author's profile keeps the author's name, its role translated.
	write(t, root, "Autores/_index.md", "---\ntitle: Autores\nblup: authors\n---\n")
	write(t, root, "Autores/Ana.md", "---\nrole: Confeiteira\n---\nFaz bolos.\n")
	if p, err := TranslateNote(context.Background(), site, "Autores/Ana.md", "es", AIOpenRouter, nil); err != nil || p != "Autores/es/Ana.md" {
		t.Fatalf("author = %s %v", p, err)
	}
	if a, _ := obsidian.ReadNote(filepath.Join(root, "Autores", "es", "Ana.md")); a.String("role") != "ES(Confeiteira)" {
		t.Fatalf("author role = %+v", a.Props)
	}
	if todo, _ := TranslationTodo(site, "es"); slices.Contains(todo, "Autores/Ana.md") || !slices.Contains(todo, "Posts/Um.md") {
		t.Fatalf("todo = %v", todo)
	}
	// A section's short name (its cards, its menu item) is translated too.
	if sec, _ := obsidian.ReadNote(filepath.Join(root, "Posts", "es", "_index.md")); sec.String("linkTitle") != "ES(Doces)" {
		t.Fatalf("section linkTitle = %+v", sec.Props)
	}
	if err := MarkReviewed(site, "Posts/es/_index.md", true); err != nil {
		t.Fatal(err)
	}
	if langs, _ := NoteLangs(site, "Posts/_index.md"); langs[2].AI || !langs[2].Exists {
		t.Fatalf("reviewed = %+v", langs)
	}
}

// The feedback board's example requests are translated with the note:
// each one's title, text and comments; votes, kind, status and dates stay.
func TestTranslateFeedbackExamples(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	creds, _ := config.LoadCredentials()
	creds.OpenRouterToken = "k"
	_ = SetAIConsent(AIOpenRouter, true)
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
	write(t, root, "Feedback/_index.md", "---\ntitle: Feedback\nblup: feedback\nexamples:\n  - title: Modo escuro\n    kind: feature\n    status: planned\n    votes: 42\n    days: 2\n    body: Seguir o sistema.\n    replies:\n      - author: marina\n        days: 1\n        text: Seria ótimo.\n---\n")

	before, _ := NoteLangs(site, "Feedback/_index.md")
	got, err := TranslateNote(context.Background(), site, "Feedback/_index.md", "en", AIOpenRouter, nil)
	if err != nil {
		t.Fatal(err)
	}
	n, err := obsidian.ReadNote(filepath.Join(root, filepath.FromSlash(got)))
	if err != nil {
		t.Fatal(err)
	}
	ex, _ := n.Props["examples"].([]any)
	if len(ex) != 1 {
		t.Fatalf("examples = %#v", n.Props["examples"])
	}
	e := ex[0].(map[string]any)
	reply := e["replies"].([]any)[0].(map[string]any)
	if e["title"] != "EN(Modo escuro)" || e["body"] != "EN(Seguir o sistema.)" || reply["text"] != "EN(Seria ótimo.)" {
		t.Fatalf("texts = %#v", e)
	}
	if e["kind"] != "feature" || e["status"] != "planned" || e["votes"] != 42 || reply["author"] != "marina" || reply["days"] != 1 {
		t.Fatalf("the rest changed = %#v", e)
	}
	if before[1].Exists {
		t.Fatal("no translation before")
	}
	// An example changed in the original: the translation is out of date.
	write(t, root, "Feedback/_index.md", "---\ntitle: Feedback\nblup: feedback\nexamples:\n  - title: Modo claro\n---\n")
	if langs, _ := NoteLangs(site, "Feedback/_index.md"); !langs[1].Stale {
		t.Fatalf("after an example changed = %+v", langs)
	}
}

func TestFeedbackTranslationRejectsMalformedTextsBeforeWriting(t *testing.T) {
	for _, props := range []map[string]any{{examplesKey: []string{"only one"}}, {examplesKey: []string{"a", "b", "extra"}}, {examplesKey: []any{"a", 42}}, {examplesKey: nil}, nil} {
		t.Run("invalid texts", func(t *testing.T) {
			site := settingsSite(t)
			root := site.ContentDir()
			const src = "---\ntitle: Feedback\nexamples:\n  - title: Original\n    body: Texto\n---\n"
			write(t, root, "Feedback/_index.md", src)
			_, err := WriteTranslation(site, "Feedback/_index.md", "en", "Feedback", "", "", props)
			if err == nil {
				t.Fatal("malformed translated texts accepted")
			}
			got, _ := os.ReadFile(filepath.Join(root, "Feedback/_index.md"))
			if string(got) != src {
				t.Fatal("original modified before validation")
			}
			if _, err := os.Stat(filepath.Join(root, "Feedback/en/_index.md")); !os.IsNotExist(err) {
				t.Fatal("translation written on invalid input")
			}
		})
	}
}

func TestFeedbackExamplesUnexpectedYAMLTypes(t *testing.T) {
	for _, yaml := range []string{"null", "42", "{title: wrong}", "[null, 1, {title: [bad], body: ok, replies: [null, {text: yes}, {text: [bad]}]}]"} {
		var v any
		if err := yamlv3.Unmarshal([]byte(yaml), &v); err != nil {
			t.Fatal(err)
		}
		texts := exampleTexts(v)
		i := 0
		_ = walkExamples(v, func(s string) string { i++; return s })
		if i != len(texts) {
			t.Fatalf("different traversal count: %d / %d", i, len(texts))
		}
	}
}
