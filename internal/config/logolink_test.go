package config

import "testing"

func TestValidLogoLink(t *testing.T) {
	for _, s := range []string{"https://empresa.com.br", "http://empresa.com/loja?x=1#a", "/sobre/", "/"} {
		if !ValidLogoLink(s) {
			t.Errorf("%q refused", s)
		}
	}
	for _, s := range []string{"", "javascript:alert(1)", "data:text/html,x", "//evil.com", "empresa.com", "ftp://x.com", "https://", "https://user:pw@x.com", "https://x.com/a b", "/a\nb", `https://x.com/"onmouseover=1`, "mailto:a@b.c", `\\evil`, "/a\x00b"} {
		if ValidLogoLink(s) {
			t.Errorf("%q accepted", s)
		}
	}
}
