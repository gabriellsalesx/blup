package app

import (
	"testing"

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
