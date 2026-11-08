package main

import (
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
)

// Every language has every text, and the reminders speak the one in use.
func TestTexts(t *testing.T) {
	for lang, keys := range texts {
		for key := range texts["pt"] {
			if keys[key] == "" {
				t.Errorf("%s has no %q", lang, key)
			}
		}
	}
	defer useLang("pt")
	at := time.Date(2026, 11, 7, 8, 0, 0, 0, time.UTC)
	day := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	for lang, want := range map[string]string{"pt": "Amanhã às 10:00", "en": "Tomorrow at 10:00", "es": "Mañana a las 10:00"} {
		useLang(lang)
		if got := whenText(app.Event{Time: "10:00"}, day, at); got != want {
			t.Errorf("%s: whenText = %q, want %q", lang, got, want)
		}
	}
}
