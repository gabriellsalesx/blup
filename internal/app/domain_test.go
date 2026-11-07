package app

import "testing"

func TestIsApex(t *testing.T) {
	for d, apex := range map[string]bool{
		"meusite.com.br":        true,
		"www.meusite.com.br":    false,
		"docs.blog.exemplo.com": false,
		"exemplo.com":           true,
	} {
		if isApex(d) != apex {
			t.Errorf("isApex(%s) = %v", d, !apex)
		}
	}
}
