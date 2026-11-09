package narrate

import (
	"slices"
	"testing"
)

func TestVoiceLabels(t *testing.T) {
	list := []string{"pt-BR-Grant:MAI-Voice-2.1", "pt-PT-Grant:MAI-Voice-2.1", "pt-BR-Luana:MAI-Voice-2.1", "pt-BR-Harper", "pt-PT-Harper", "Sage"}
	if got := VoiceLabels(list, "pt"); !slices.Equal(got, []string{"Grant (Brasil)", "Grant (Portugal)", "Luana", "Harper (Brasil)", "Harper (Portugal)", "Sage"}) {
		t.Fatalf("pt: %v", got)
	}
	if got := VoiceLabels([]string{"en-US-Grant", "en-GB-Grant"}, "en"); !slices.Equal(got, []string{"Grant (US)", "Grant (UK)"}) {
		t.Fatalf("en: %v", got)
	}
	if got := VoiceLabels([]string{"es-MX-Grant", "es-ES-Grant"}, "es"); !slices.Equal(got, []string{"Grant (México)", "Grant (España)"}) {
		t.Fatalf("es: %v", got)
	}
	if got := VoiceLabels([]string{"en-US-Grant", "en-GB-Grant"}, "pt"); !slices.Equal(got, []string{"Grant (EUA)", "Grant (Reino Unido)"}) {
		t.Fatalf("en voices, Studio in pt: %v", got)
	}
	// An unknown region shows its code.
	if got := VoiceLabels([]string{"fr-CA-Ana", "fr-FR-Ana"}, "pt"); !slices.Equal(got, []string{"Ana (CA)", "Ana (FR)"}) {
		t.Fatalf("unknown: %v", got)
	}
	if VoiceRegion("pt-BR-Luana:MAI-Voice-2.1") != "BR" || VoiceRegion("pf_dora") != "" {
		t.Fatal("region")
	}
}
