package translate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSplitKeepsCodeWhole(t *testing.T) {
	text := strings.Repeat("Paragraph one.\n\n", 40) + "```\ncode\n\nmore code\n```\n\nEnd.\n"
	parts := Split(text, 200)
	if strings.Join(parts, "") != text {
		t.Fatal("parts don't add up to the text")
	}
	for _, p := range parts {
		if strings.Count(p, "```")%2 != 0 {
			t.Fatalf("a code block was cut: %q", p)
		}
	}
	if len(parts) < 3 {
		t.Fatalf("parts = %d", len(parts))
	}
}

func TestTextThroughOpenRouter(t *testing.T) {
	var got struct {
		Model    string   `json:"model"`
		Models   []string `json:"models"`
		Messages []struct{ Role, Content string }
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer k" {
			http.Error(w, "bad", 400)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "Hello **world**."}, "finish_reason": "stop"}}})
	}))
	defer srv.Close()
	out, err := (&Client{Key: "k", BaseURL: srv.URL}).Text(context.Background(), "\nOlá **mundo**.\n\n", "pt", "en", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "\nHello **world**.\n\n" {
		t.Fatalf("out = %q", out)
	}
	if got.Model != Model || got.Messages[1].Content != "Olá **mundo**." || !strings.Contains(got.Messages[0].Content, "Brazilian Portuguese to English") {
		t.Fatalf("request = %+v", got)
	}
	// The staff's list: OpenRouter tries them in order.
	got.Model, got.Models = "", nil
	if _, err := (&Client{Key: "k", BaseURL: srv.URL, Models: []string{"a/one", "b/two"}}).Text(context.Background(), "Olá", "pt", "en", nil); err != nil {
		t.Fatal(err)
	}
	if got.Model != "" || strings.Join(got.Models, ",") != "a/one,b/two" {
		t.Fatalf("models = %q %v", got.Model, got.Models)
	}
}

func TestBlupLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":{"message":"acabou","code":"AI_MONTH_LIMIT"}}`))
	}))
	defer srv.Close()
	_, err := (&Client{Key: "s", BaseURL: srv.URL, Blup: true}).Text(context.Background(), "Olá", "pt", "es", nil)
	if err == nil || !strings.Contains(err.Error(), "acabou") || !errors.Is(err, ErrLimit) {
		t.Fatalf("err = %v", err)
	}
}

// Blup Cloud takes at most 16000 characters a part: paragraphs are packed
// up to MaxPart, never one past it (only a paragraph alone may be longer).
func TestSplitStaysUnderMax(t *testing.T) {
	para := strings.Repeat("palavra ", 875) + "\n\n" // 7002 bytes
	text := strings.Repeat("x", 8990) + "\n\n" + para + para + para
	parts := Split(text, MaxPart)
	if strings.Join(parts, "") != text {
		t.Fatal("parts don't add up to the text")
	}
	for _, p := range parts {
		if len(p) > MaxPart {
			t.Fatalf("part of %d bytes", len(p))
		}
	}
}

func TestOpenRouterAttribution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-OpenRouter-Title") != "Blup Studio" || r.Header.Get("HTTP-Referer") == "" {
			http.Error(w, "no attribution", 400)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Hi"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()
	if out, err := (&Client{Key: "k", BaseURL: srv.URL}).Text(context.Background(), "Oi", "pt", "en", nil); err != nil || out != "Hi" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
}

// TestCutGoesInHalves: an answer cut for lack of room comes back as CUT;
// the part goes again in halves (never the same text twice), and the
// halves come back joined with the blank lines they had.
func TestCutGoesInHalves(t *testing.T) {
	var sent []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Text string }
		_ = json.NewDecoder(r.Body).Decode(&b)
		sent = append(sent, b.Text)
		if strings.Count(b.Text, "\n\n") >= 2 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":{"message":"longa","code":"CUT"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"text": strings.ToUpper(b.Text)})
	}))
	defer srv.Close()
	out, err := (&Client{Key: "s", BaseURL: srv.URL, Blup: true}).Text(context.Background(), "um\n\ndois\n\ntrês\n\nquatro", "pt", "en", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "UM\n\nDOIS\n\nTRÊS\n\nQUATRO" {
		t.Fatalf("out = %q", out)
	}
	if sent[0] != "um\n\ndois\n\ntrês\n\nquatro" || len(sent) < 3 {
		t.Fatalf("sent = %q", sent)
	}
	for _, s := range sent[1:] {
		if s == sent[0] {
			t.Fatal("the cut text went again whole")
		}
	}
}

// TestCutAloneFails: a single paragraph cut has nothing smaller to go in,
// so the error comes back (once, no retries of the same text).
func TestCutAloneFails(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"message":"longa","code":"CUT"}}`))
	}))
	defer srv.Close()
	_, err := (&Client{Key: "s", BaseURL: srv.URL, Blup: true}).Text(context.Background(), "um parágrafo só", "pt", "en", nil)
	if !errors.Is(err, ErrCut) || n != 1 {
		t.Fatalf("err = %v, calls = %d", err, n)
	}
}

// TestFieldsInOneRequest: the short texts go together, each after its
// mark, and come back in their places; an answer that loses a mark falls
// back to one request per text.
func TestFieldsInOneRequest(t *testing.T) {
	broken := false
	var sent []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Text string }
		_ = json.NewDecoder(r.Body).Decode(&b)
		sent = append(sent, b.Text)
		out := strings.ToUpper(b.Text)
		out = strings.ReplaceAll(out, "<!--F", "<!--f")
		if broken && strings.Contains(out, "<!--f1-->") {
			out = strings.Replace(out, "<!--f1-->", "", 1)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"text": out})
	}))
	defer srv.Close()
	c := &Client{Key: "s", BaseURL: srv.URL, Blup: true}
	got, err := c.Fields(context.Background(), []string{"título", "", "descrição", "tag"}, "pt", "en")
	if err != nil || strings.Join(got, "|") != "TÍTULO||DESCRIÇÃO|TAG" || len(sent) != 1 {
		t.Fatalf("got %q, %v, sent %q", got, err, sent)
	}
	broken, sent = true, nil
	got, err = c.Fields(context.Background(), []string{"um", "dois"}, "pt", "en")
	if err != nil || strings.Join(got, "|") != "UM|DOIS" || len(sent) != 3 {
		t.Fatalf("fallback: got %q, %v, sent %q", got, err, sent)
	}
}

// TestWaitsWhatTheServerAsks: a 429 with Retry-After waits that long and
// tries again, instead of failing the note.
func TestWaitsWhatTheServerAsks(t *testing.T) {
	n := 0
	var first time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			first = time.Now()
			w.Header().Set("Retry-After", "3")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"Muitos pedidos","code":"TOO_MANY_REQUESTS"}}`))
			return
		}
		if time.Since(first) < 3*time.Second {
			t.Errorf("tried again after %v", time.Since(first))
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "ok"})
	}))
	defer srv.Close()
	out, err := (&Client{Key: "s", BaseURL: srv.URL, Blup: true}).Text(context.Background(), "olá", "pt", "en", nil)
	if err != nil || out != "ok" || n != 2 {
		t.Fatalf("out %q, err %v, calls %d", out, err, n)
	}
}
