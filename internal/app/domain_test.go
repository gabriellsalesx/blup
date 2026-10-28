package app

import "testing"

func TestNormalizeDomain(t *testing.T) {
	ok := map[string]string{
		"meusite.com.br":                "meusite.com.br",
		" https://www.MeuSite.com.br/ ": "www.meusite.com.br",
		"blog.exemplo.com.":             "blog.exemplo.com",
		"café.com.br":                   "xn--caf-dma.com.br",
	}
	for in, want := range ok {
		if got, err := normalizeDomain(in); err != nil || got != want {
			t.Errorf("normalizeDomain(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "meusite", "site.pages.dev", "eu.github.io", "a b.com", "-x.com", "x.invalidtld"} {
		if got, err := normalizeDomain(in); err == nil {
			t.Errorf("normalizeDomain(%q) = %q, want error", in, got)
		}
	}
}

func TestApexAndLabel(t *testing.T) {
	cases := []struct {
		d     string
		apex  bool
		label string
	}{
		{"meusite.com.br", true, "@"},
		{"www.meusite.com.br", false, "www"},
		{"docs.blog.exemplo.com", false, "docs.blog"},
		{"exemplo.com", true, "@"},
	}
	for _, c := range cases {
		if isApex(c.d) != c.apex || subLabel(c.d) != c.label {
			t.Errorf("%s: apex=%v label=%q", c.d, isApex(c.d), subLabel(c.d))
		}
	}
}
