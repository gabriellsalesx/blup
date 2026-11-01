package i18n

import "testing"

func TestSystem(t *testing.T) {
	for _, c := range []struct {
		lcAll, lang string
		want        Lang
	}{
		{"", "pt_BR.UTF-8", "pt"},
		{"", "pt_PT.UTF-8", "pt"},
		{"", "es_ES.UTF-8", "en"},
		{"", "en_US.UTF-8", "en"},
		{"C", "pt_BR.UTF-8", "pt"}, // C says nothing: the next one decides
		{"", "", "en"},
	} {
		t.Setenv("LC_ALL", c.lcAll)
		t.Setenv("LC_MESSAGES", "")
		t.Setenv("LANGUAGE", "")
		t.Setenv("LANG", c.lang)
		if got := System(); got != c.want {
			t.Errorf("LC_ALL=%q LANG=%q: %q, want %q", c.lcAll, c.lang, got, c.want)
		}
	}
}
