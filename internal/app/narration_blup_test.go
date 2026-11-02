package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/narrate"
)

func TestBlupVoiceChoice(t *testing.T) {
	// Older choices (a recommended model's voice alone) keep working.
	if m, v := blupChoice("pt-BR-Caio:MAI-Voice-2.1"); m != narrate.DefaultModel || v != "pt-BR-Caio:MAI-Voice-2.1" {
		t.Fatalf("old choice = %q %q", m, v)
	}
	if m, v := blupChoice("openai/gpt-voice|alloy"); m != "openai/gpt-voice" || v != "alloy" {
		t.Fatalf("model choice = %q %q", m, v)
	}
	if got := voiceFor("pt", "blup:pt-BR-Caio:MAI-Voice-2.1"); got != "blup:"+narrate.DefaultModel+"|pt-BR-Caio:MAI-Voice-2.1" {
		t.Fatalf("old voice = %q", got)
	}
	if got := voiceFor("pt", "blup:openai/gpt-voice|alloy"); got != "blup:openai/gpt-voice|alloy" {
		t.Fatalf("another model = %q", got)
	}
	if !NarratesThroughBlup("pt", "blup:openai/gpt-voice|alloy") {
		t.Fatal("not read through Blup Cloud")
	}
}

// A estimativa da narração pelo Blup Cloud não pode ser enganada: um modelo
// fora da lista do OpenRouter (vindo do frontend) ou com a lista fora do ar
// não ganha o preço barato do modelo recomendado; a narração nem começa.
func TestBlupNarrationPriceNotGuessed(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	down := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if down {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`{"data":[{"id":"microsoft/mai-voice-2.1","name":"MAI","pricing":{"prompt":"0.000022"}},{"id":"cara/voz","name":"Cara","pricing":{"prompt":"0.0003"}},{"id":"por/segundo","name":"S","pricing":{"prompt":"0"}}]}`))
	}))
	defer srv.Close()
	openRouterURL = srv.URL
	defer func() { openRouterURL = "" }()
	ctx := context.Background()
	u := blupcloud.AIUsage{PerCharMicros: 22, MonthMicros: 5_000_000, WeekCapMicros: 2_500_000}

	if p, ok := perCharMicros(ctx, u, narrate.DefaultModel); !ok || p != 22 {
		t.Fatalf("default = %v %v", p, ok)
	}
	if p, ok := perCharMicros(ctx, u, "cara/voz"); !ok || p < 299 || p > 301 {
		t.Fatalf("dear model = %v %v", p, ok)
	}
	for _, m := range []string{"inventado/pelo-frontend", "por/segundo"} {
		if p, ok := perCharMicros(ctx, u, m); ok {
			t.Fatalf("%s priced at %v", m, p)
		}
	}
	down = true
	if _, ok := perCharMicros(ctx, u, "cara/voz"); ok {
		t.Fatal("priced without the list")
	}
	// The dear model really uses more of the allowance than the default.
	text := strings.Repeat("a", 10000)
	_, cheap := costOf(u, 22, text)
	_, dear := costOf(u, 300, text)
	if dear.Percent <= cheap.Percent {
		t.Fatalf("cheap %d%%, dear %d%%", cheap.Percent, dear.Percent)
	}
}
