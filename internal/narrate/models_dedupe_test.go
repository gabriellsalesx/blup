package narrate

import (
	"slices"
	"testing"
)

func TestVoicesForOneVoicePerName(t *testing.T) {
	m := ModelInfo{ID: "x/voice", Multilingual: true, Voices: []string{"pt-BR-Luana", "Grant", "pt-BR-Grant", "pt-BR-Grant", "Harper", "en-US-Iris"}}
	got := VoicesFor(m, "pt")
	if !slices.Equal(got, []string{"pt-BR-Luana", "pt-BR-Grant", "Harper"}) {
		t.Fatalf("got %v", got)
	}
}
