package app

import "testing"

func TestSplitFront(t *testing.T) {
	cases := []struct{ in, fm, body string }{
		{"---\ntags: [a]\n---\ntexto", "tags: [a]\n", "texto"},
		{"\ufeff---\r\ntitle: x\r\n---\r\nok", "title: x\n", "ok"},
		{"---\n---\nvazio", "", "vazio"},
		{"---\na: 1\n----\nb: 2\n---\nfim", "a: 1\n----\nb: 2\n", "fim"},
		{"---\na: 1\n---", "a: 1\n", ""},
		{"sem cabeçalho", "", "sem cabeçalho"},
		{"---\nnunca fecha", "", "---\nnunca fecha"},
	}
	for _, c := range cases {
		if fm, body := splitFront(c.in); fm != c.fm || body != c.body {
			t.Errorf("splitFront(%q) = %q, %q; want %q, %q", c.in, fm, body, c.fm, c.body)
		}
	}
}
