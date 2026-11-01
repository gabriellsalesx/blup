package translate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
		Model    string `json:"model"`
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
