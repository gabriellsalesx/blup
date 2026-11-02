package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	yamlv3 "go.yaml.in/yaml/v3"
)

// fakeCloud is OpenRouter and Blup Cloud at once: it answers translations
// (in capitals), says the plan includes AI and records what reached it.
type fakeCloud struct {
	mu    sync.Mutex
	texts []string // what was sent to be translated
	calls []string // method and path of every request
}

func (f *fakeCloud) server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		f.mu.Unlock()
		switch r.URL.Path {
		case "/api/me":
			w.Write([]byte(`{"id":"u","plan":{"name":"pro","limits":{"aiUsage":1}}}`))
		case "/api/ai/translate":
			var req struct{ Text string }
			json.Unmarshal(b, &req)
			f.mu.Lock()
			f.texts = append(f.texts, req.Text)
			f.mu.Unlock()
			json.NewEncoder(w).Encode(map[string]string{"text": strings.ToUpper(req.Text)})
		case "/chat/completions":
			var req struct{ Messages []struct{ Content string } }
			json.Unmarshal(b, &req)
			f.mu.Lock()
			f.texts = append(f.texts, req.Messages[1].Content)
			f.mu.Unlock()
			json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": strings.ToUpper(req.Messages[1].Content)}, "finish_reason": "stop"}}})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	openRouterURL = srv.URL
	t.Cleanup(func() { openRouterURL = "" })
	return srv
}

func (f *fakeCloud) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.texts...)
}

// A IA na nuvem só recebe o texto de uma nota quando a pessoa escolheu
// quem traduz e aceitou mandar para aquele destino. O Studio nunca escolhe
// o Blup Cloud sozinho, mesmo com o plano incluindo IA.
func TestTranslateNeedsChoiceAndConsent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var f fakeCloud
	f.server(t)
	if err := (config.Credentials{OpenRouterToken: "k", BlupToken: "tok"}).Save(); err != nil {
		t.Fatal(err)
	}
	site := settingsSite(t)
	write(t, site.ContentDir(), "Posts/Bolo.md", "Receita secreta da vovó.\n")
	ctx := context.Background()

	// No choice: nothing goes, not even with Blup Cloud's AI in the plan.
	if _, err := TranslateNote(ctx, site, "Posts/Bolo.md", "en", "", nil); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("no choice: %v", err)
	}
	if _, err := TranslateNote(ctx, site, "Posts/Bolo.md", "en", "cloud", nil); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("the older automatic choice: %v", err)
	}
	if _, err := TranslateSite(ctx, site, "en", "", nil); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("whole site without a choice: %v", err)
	}
	// Chosen, not accepted yet: asked for each destination on its own.
	for _, dest := range []string{AIOpenRouter, AIBlup} {
		var ce *ConsentError
		if _, err := TranslateNote(ctx, site, "Posts/Bolo.md", "en", dest, nil); !errors.As(err, &ce) || ce.Dest != dest {
			t.Fatalf("%s without consent: %v", dest, err)
		}
	}
	if s := f.sent(); len(s) != 0 {
		t.Fatalf("text sent before consent: %q", s)
	}
	// Accepting OpenRouter doesn't accept Blup Cloud.
	if err := SetAIConsent(AIOpenRouter, true); err != nil {
		t.Fatal(err)
	}
	var ce *ConsentError
	if _, err := TranslateNote(ctx, site, "Posts/Bolo.md", "en", AIBlup, nil); !errors.As(err, &ce) {
		t.Fatalf("Blup Cloud after accepting OpenRouter: %v", err)
	}
	if _, err := TranslateNote(ctx, site, "Posts/Bolo.md", "en", AIOpenRouter, nil); err != nil {
		t.Fatal(err)
	}
	if err := SetAIConsent(AIBlup, true); err != nil {
		t.Fatal(err)
	}
	if _, err := TranslateNote(ctx, site, "Posts/Bolo.md", "es", AIBlup, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(f.calls, ","), "POST /api/ai/translate") {
		t.Fatalf("Blup Cloud not used: %v", f.calls)
	}
	// Kept on this computer, readable only by its user.
	cfg, _ := config.Dir()
	st, err := os.Stat(filepath.Join(cfg, "ai-consent.json"))
	if err != nil {
		t.Fatal(err)
	}
	if c := LoadAIConsent(); !c.OpenRouter || !c.Blup {
		t.Fatalf("consent = %+v", c)
	}
	if st.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
		t.Fatalf("consent file mode %v", st.Mode())
	}
	if err := SetAIConsent("other", true); err == nil {
		t.Fatal("unknown destination accepted")
	}
}

// Posts com senha nunca vão para a IA na nuvem: nem um a um, nem na
// tradução do site inteiro, que pula os trancados e conta quantos.
func TestLockedNotesNeverGoToCloud(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var f fakeCloud
	f.server(t)
	if err := (config.Credentials{OpenRouterToken: "k"}).Save(); err != nil {
		t.Fatal(err)
	}
	if err := SetAIConsent(AIOpenRouter, true); err != nil {
		t.Fatal(err)
	}
	site := settingsSite(t)
	root := site.ContentDir()
	write(t, root, "Posts/Segredo.md", "---\npassword: abacaxi\n---\nTexto trancado do diario.\n")
	write(t, root, "Posts/Geral.md", "---\npassword: sim\n---\nOutro texto trancado.\n")
	write(t, root, "Posts/Aberto.md", "---\npassword: false\n---\nTexto aberto.\n")
	ctx := context.Background()

	if _, err := TranslateNote(ctx, site, "Posts/Segredo.md", "en", AIOpenRouter, nil); err == nil {
		t.Fatal("a locked post was translated in the cloud")
	}
	res, err := TranslateSite(ctx, site, "en", AIOpenRouter, nil)
	if err != nil {
		t.Fatal(err)
	}
	// _index, Posts/Um and Posts/Aberto; the two locked ones left out.
	if res.Done != 3 || res.Skipped != 2 {
		t.Fatalf("site = %+v", res)
	}
	for _, s := range f.sent() {
		if strings.Contains(s, "trancado") {
			t.Fatalf("locked text sent: %q", s)
		}
	}
	if !strings.Contains(strings.Join(f.sent(), "\n"), "Texto aberto.") {
		t.Fatalf("open post not translated: %q", f.sent())
	}
	// The local translation still lists them (it never leaves the computer).
	todo, _ := TranslationTodo(site, "en")
	if !strings.Contains(strings.Join(todo, ","), "Posts/Segredo.md") {
		t.Fatalf("todo = %v", todo)
	}
}

func TestNoteLocked(t *testing.T) {
	cases := map[string]bool{
		"password: abc":       true,
		"password: sim":       true,
		"password: true":      true,
		"password: 1234":      true,
		"password: \"false\"": true, // the theme locks it, with "false" as the password
		"password: false":     false,
		"password: \"\"":      false,
		"password:":           false,
		"title: x":            false,
	}
	for yaml, want := range cases {
		var props map[string]any
		if err := yamlv3.Unmarshal([]byte(yaml), &props); err != nil {
			t.Fatal(err)
		}
		if got := noteLocked(obsidian.Note{Props: props}); got != want {
			t.Errorf("%s: locked = %v, want %v", yaml, got, want)
		}
	}
}

// Um arquivo de aceite estragado vale como nenhum aceite (na dúvida,
// pergunta de novo).
func TestAIConsentFailsClosed(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := SetAIConsent(AIOpenRouter, true); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Dir()
	f := filepath.Join(cfg, "ai-consent.json")
	for _, bad := range []string{`{"openrouter":true,"blup":"invalido"}`, `{"openrouter":true,"blup":tr`, `null`, `[true]`, `"x"`, ``} {
		if err := os.WriteFile(f, []byte(bad), 0o600); err != nil {
			t.Fatal(err)
		}
		if c := LoadAIConsent(); c.OpenRouter || c.Blup {
			t.Errorf("%q: consent = %+v", bad, c)
		}
		if err := needConsent(AIOpenRouter); err == nil {
			t.Errorf("%q: sending allowed", bad)
		}
	}
	os.WriteFile(f, []byte(`{"openrouter":true}`), 0o600)
	if c := LoadAIConsent(); !c.OpenRouter || c.Blup {
		t.Fatalf("good file: %+v", c)
	}
}
