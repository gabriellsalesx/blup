package narrate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSpeakable(t *testing.T) {
	body := "Primeiro parágrafo com **negrito**, *itálico*, `código` e [um link](https://x.com).\nContinua na linha seguinte.\n\n" +
		"## Um título\n\n" +
		"- item um\n- item dois!\n\n" +
		"> [!tip] Dica boa\n> Texto da dica.\n\n" +
		"```go\nfmt.Println(\"não lê\")\n```\n\n" +
		"| a | b |\n| --- | --- |\n| 1 | 2 |\n\n" +
		"%%comentário escondido%%\n\n" +
		"![foto](Anexos/x.png) Veja [[Outra nota|a outra nota]] e [[Mais uma]]. Fórmula $x^2$ aqui.[^1]\n\n" +
		"[^1]: nota de rodapé\n\n" +
		"Alinhado ao centro.\n{.text-center}\n"
	got := Speakable("Meu post", body)
	want := []string{
		"Meu post.",
		"Primeiro parágrafo com negrito, itálico, código e um link. Continua na linha seguinte.",
		"Um título.",
		"item um.",
		"item dois!",
		"Dica boa.",
		"Texto da dica.",
		"Veja a outra nota e Mais uma. Fórmula aqui.",
		"Alinhado ao centro.",
	}
	if got != strings.Join(want, "\n\n") {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, strings.Join(want, "\n\n"))
	}
	for _, bad := range []string{"Println", "| a", "comentário", "rodapé", "x.png", "**", "{.text"} {
		if strings.Contains(got, bad) {
			t.Errorf("%q should not be read", bad)
		}
	}
}

func TestSplit(t *testing.T) {
	text := strings.Repeat("Uma frase curta aqui. ", 30) + "\n\n" + strings.Repeat("palavra ", 50) + "\n\nFim."
	parts := Split(text, 120)
	if len(parts) < 3 {
		t.Fatalf("expected several parts, got %d", len(parts))
	}
	for _, p := range parts {
		if len(p) > 120 {
			t.Errorf("part too long (%d): %q", len(p), p)
		}
	}
	joined := strings.Join(parts, " ")
	if strings.Fields(joined)[0] != "Uma" || !strings.HasSuffix(joined, "Fim.") {
		t.Errorf("text lost: %q", joined)
	}
	if got := Split("Curto.", 100); len(got) != 1 || got[0] != "Curto." {
		t.Errorf("short text: %q", got)
	}
}

// mp3File builds an MP3 file of n silent MPEG-2 Layer III frames (24 kHz,
// 48 kbit/s, mono: 144 bytes each), with an ID3v2 tag and an Info frame.
func mp3File(n int, fill byte) []byte {
	head := []byte{0xFF, 0xF3, 0x64, 0xC0}
	fr := func(b byte) []byte {
		f := append([]byte{}, head...)
		return append(f, bytes.Repeat([]byte{b}, 140)...)
	}
	var out bytes.Buffer
	out.Write([]byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0, 5})
	out.Write([]byte("abcde"))
	info := fr(0)
	copy(info[13:], "Info")
	out.Write(info)
	for i := 0; i < n; i++ {
		out.Write(fr(fill))
	}
	return out.Bytes()
}

func TestJoinMP3(t *testing.T) {
	out, secs, err := JoinMP3([][]byte{mp3File(3, 1), mp3File(2, 2)})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 5*144 {
		t.Fatalf("got %d bytes, want %d (5 frames, no tags or Info frames)", len(out), 5*144)
	}
	if bytes.Contains(out, []byte("Info")) || bytes.Contains(out, []byte("ID3")) {
		t.Error("tags or Info frames left in")
	}
	if out[4] != 1 || out[4*144+4] != 2 {
		t.Error("frames out of order")
	}
	if want := 5 * 576 / 24000.0; secs != want {
		t.Errorf("seconds = %v, want %v", secs, want)
	}
	if _, _, err := JoinMP3([][]byte{[]byte("not audio")}); !errors.Is(err, ErrNotMP3) {
		t.Errorf("want ErrNotMP3, got %v", err)
	}
}

func TestSpeak(t *testing.T) {
	var got map[string]any
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer k1" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"No auth credentials found"}}`))
			return
		}
		switch r.URL.Path {
		case "/key":
			w.Write([]byte(`{"data":{"label":"x"}}`))
		case "/audio/speech":
			if calls == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			json.NewDecoder(r.Body).Decode(&got)
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write(mp3File(1, 9))
		}
	}))
	defer srv.Close()
	c := &Client{Key: "k1", BaseURL: srv.URL}
	audio, err := c.Speak(context.Background(), "Olá", "nova", "Leia em português.")
	if err != nil {
		t.Fatal(err)
	}
	if len(audio) == 0 || calls != 2 {
		t.Errorf("audio %d bytes after %d calls (one retry expected)", len(audio), calls)
	}
	if got["model"] != Model || got["input"] != "Olá" || got["voice"] != "nova" || got["response_format"] != "mp3" {
		t.Errorf("request body: %v", got)
	}
	if err := c.CheckKey(context.Background()); err != nil {
		t.Errorf("CheckKey: %v", err)
	}
	bad := &Client{Key: "nope", BaseURL: srv.URL}
	if err := bad.CheckKey(context.Background()); !errors.Is(err, ErrInvalidKey) {
		t.Errorf("want ErrInvalidKey, got %v", err)
	}
}
