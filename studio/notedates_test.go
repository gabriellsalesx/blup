package main

import "testing"

func TestCreatedProp(t *testing.T) {
	cases := map[string]string{
		"---\ncreated: 2026-04-01\nupdated: 2026-04-02\n---\n\nText": "2026-04-01",
		"---\ncreated: \"2026-04-01 09:47\"\n---\nText":              "2026-04-01 09:47",
		"---\ncreated: 2026-04-01T09:47:00Z\n---\nText":              "",
		"---\ntitle: x\n---\nText":                                   "",
		"No properties":                                              "",
		"---\ncreated: not a date\n---\n":                            "",
	}
	for text, want := range cases {
		got := createdProp(text)
		if want == "" && text == "---\ncreated: 2026-04-01T09:47:00Z\n---\nText" {
			if got == "" {
				t.Errorf("a full timestamp must give a date and time")
			}
			continue
		}
		if got != want {
			t.Errorf("createdProp(%q) = %q, want %q", text, got, want)
		}
	}
}
