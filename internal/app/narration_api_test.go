package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

// fakeMP3 is one silent MPEG-2 Layer III frame (24 kHz, 48 kbit/s, mono).
func fakeMP3() []byte {
	return append([]byte{0xFF, 0xF3, 0x64, 0xC0}, bytes.Repeat([]byte{0}, 140)...)
}

func TestNarrate(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir()) // no local voice
	var inputs, voices []string
	var models []byte = []byte(`{"data":[{"id":"microsoft/mai-voice-2.1","name":"MAI","pricing":{"prompt":"0.000022"},"supported_voices":["pt-BR-Luana:MAI-Voice-2.1","pt-BR-Caio:MAI-Voice-2.1","en-US-Harper:MAI-Voice-2.1"]},{"id":"elevenlabs/eleven","name":"Eleven","description":"90+ languages","pricing":{"prompt":"0.00004"},"supported_voices":["sarah"]}]}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.Write(models)
			return
		}
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		inputs = append(inputs, req["input"])
		voices = append(voices, req["model"]+"|"+req["voice"])
		w.Write(fakeMP3())
	}))
	defer srv.Close()
	openRouterURL = srv.URL
	defer func() { openRouterURL = "" }()

	site := settingsSite(t)
	site.Lang = "pt"
	root := site.ContentDir()
	write(t, root, "Post.md", "---\ntitle: Meu post\ndescription: Sobre ler em voz alta\n---\nUm texto com **várias** palavras para ler em voz alta.\n")

	if _, err := Narrate(context.Background(), site, "Post.md", "nova", nil); !errors.Is(err, ErrNoAIKey) {
		t.Fatalf("without a key: %v", err)
	}
	if err := (config.Credentials{OpenRouterToken: "k"}).Save(); err != nil {
		t.Fatal(err)
	}
	// Nothing goes to OpenRouter before the person accepts, and no choice
	// is the local voice (not installed here), never the cloud.
	var ce *ConsentError
	if _, err := Narrate(context.Background(), site, "Post.md", "nova", nil); !errors.As(err, &ce) || ce.Dest != AIOpenRouter {
		t.Fatalf("without consent: %v", err)
	}
	if _, err := Narrate(context.Background(), site, "Post.md", "", nil); err == nil || errors.As(err, &ce) {
		t.Fatalf("no voice chosen: %v", err)
	}
	if _, err := Narrate(context.Background(), site, "Post.md", "blup:pt-BR-Caio:MAI-Voice-2.1", nil); !errors.As(err, &ce) || ce.Dest != AIBlup {
		t.Fatalf("Blup Cloud without consent: %v", err)
	}
	if len(inputs) != 0 {
		t.Fatalf("text sent before consent: %q", inputs)
	}
	if err := SetAIConsent(AIOpenRouter, true); err != nil {
		t.Fatal(err)
	}
	if c := LoadAIConsent(); !c.OpenRouter || c.Blup {
		t.Fatalf("consent = %+v", c)
	}
	var steps []int
	n, err := Narrate(context.Background(), site, "Post.md", "nova", func(done, total int) { steps = append(steps, done) })
	if err != nil {
		t.Fatal(err)
	}
	if n.Link != "[[Post.mp3]]" || n.Bytes != 144 || len(steps) != 2 {
		t.Fatalf("narration: %+v steps %v", n, steps)
	}
	if len(inputs) != 1 || inputs[0] != "Meu post.\n\nSobre ler em voz alta.\n\nUm texto com várias palavras para ler em voz alta." {
		t.Fatalf("read aloud: %q", inputs)
	}
	if _, err := os.Stat(filepath.Join(root, attachmentsFolder(site), "Post.mp3")); err != nil {
		t.Fatal(err)
	}

	// The hash follows the text; the same note keeps its file.
	h, _ := NarrationHash(site, "Post.md")
	if h != n.Hash {
		t.Fatalf("hash %s != %s", h, n.Hash)
	}
	write(t, root, "Post.md", "---\ntitle: Meu post\naudio: \"[[Post.mp3]]\"\n---\nOutro texto, agora mudado, com mais palavras.\n")
	if h2, _ := NarrationHash(site, "Post.md"); h2 == h {
		t.Fatal("hash did not change with the text")
	}
	n2, err := Narrate(context.Background(), site, "Post.md", "nova", nil)
	if err != nil || n2.Link != "[[Post.mp3]]" {
		t.Fatalf("again: %+v %v", n2, err)
	}

	// Another note with the same name elsewhere gets a free name.
	write(t, root, "Pasta/Post.md", "Um texto diferente para outra nota qualquer.\n")
	if n3, err := Narrate(context.Background(), site, "Pasta/Post.md", "nova", nil); err != nil || n3.Link != "[[Post 2.mp3]]" {
		t.Fatalf("taken name: %+v %v", n3, err)
	}

	if voices[0] != "microsoft/mai-voice-2.1|pt-BR-Luana:MAI-Voice-2.1" {
		t.Fatalf("default voice: %v", voices)
	}

	// A chosen model that left OpenRouter doesn't stop the narration: the
	// recommended voice reads it, and the result says so.
	voices = nil
	if n, err := Narrate(context.Background(), site, "Post.md", "gone/model|x", nil); err != nil || n.Used == "" || voices[0] != "microsoft/mai-voice-2.1|pt-BR-Luana:MAI-Voice-2.1" {
		t.Fatalf("fallback: %+v %v %v", n, err, voices)
	}
	// Another model works when chosen, with its voice.
	voices = nil
	if n, err := Narrate(context.Background(), site, "Post.md", "elevenlabs/eleven|sarah", nil); err != nil || n.Used != "" || voices[0] != "elevenlabs/eleven|sarah" {
		t.Fatalf("other model: %+v %v %v", n, err, voices)
	}

	write(t, root, "Secreto.md", "---\npassword: sim\n---\nTexto secreto que não pode virar áudio.\n")
	if _, err := Narrate(context.Background(), site, "Secreto.md", "nova", nil); err == nil {
		t.Fatal("a locked post was narrated")
	}
}

func TestVoiceFor(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir()) // no local voice installed
	cases := []struct{ lang, chosen, want string }{
		{"pt", "", "kokoro:pf_dora"}, // no choice: the local voice, never the cloud
		{"es", "", "kokoro:ef_dora"},
		{"pt", "pt-BR-Caio:MAI-Voice-2.1", "microsoft/mai-voice-2.1|pt-BR-Caio:MAI-Voice-2.1"},
		{"en", "pt-BR-Caio:MAI-Voice-2.1", "microsoft/mai-voice-2.1|en-US-Harper:MAI-Voice-2.1"},
		{"pt", "elevenlabs/eleven|sarah", "elevenlabs/eleven|sarah"},
		{"pt", "kokoro:pf_dora", "kokoro:pf_dora"},
		{"en", "kokoro:pf_dora", "kokoro:af_heart"},                         // another language: still local
		{"pt", "nova", "microsoft/mai-voice-2.1|pt-BR-Luana:MAI-Voice-2.1"}, // an older OpenRouter choice stays there
		{"en", "kokoro:bm_george", "kokoro:bm_george"},
	}
	for _, c := range cases {
		if got := voiceFor(c.lang, c.chosen); got != c.want {
			t.Errorf("voiceFor(%q, %q) = %q, want %q", c.lang, c.chosen, got, c.want)
		}
	}
}
