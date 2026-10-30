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
	var inputs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		inputs = append(inputs, req["input"])
		if req["voice"] != "pt-BR-Luana:MAI-Voice-2.1" {
			t.Errorf("voice not in the site's language: %q", req["voice"])
		}
		w.Write(fakeMP3())
	}))
	defer srv.Close()
	openRouterURL = srv.URL
	defer func() { openRouterURL = "" }()

	site := settingsSite(t)
	site.Lang = "pt"
	root := site.ContentDir()
	write(t, root, "Post.md", "---\ntitle: Meu post\n---\nUm texto com **várias** palavras para ler em voz alta.\n")

	if _, err := Narrate(context.Background(), site, "Post.md", "nova", nil); !errors.Is(err, ErrNoAIKey) {
		t.Fatalf("without a key: %v", err)
	}
	if err := (config.Credentials{OpenRouterToken: "k"}).Save(); err != nil {
		t.Fatal(err)
	}
	var steps []int
	n, err := Narrate(context.Background(), site, "Post.md", "nova", func(done, total int) { steps = append(steps, done) })
	if err != nil {
		t.Fatal(err)
	}
	if n.Link != "[[Post.mp3]]" || n.Bytes != 144 || len(steps) != 2 {
		t.Fatalf("narration: %+v steps %v", n, steps)
	}
	if len(inputs) != 1 || inputs[0] != "Meu post.\n\nUm texto com várias palavras para ler em voz alta." {
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

	write(t, root, "Secreto.md", "---\npassword: sim\n---\nTexto secreto que não pode virar áudio.\n")
	if _, err := Narrate(context.Background(), site, "Secreto.md", "nova", nil); err == nil {
		t.Fatal("a locked post was narrated")
	}
}
