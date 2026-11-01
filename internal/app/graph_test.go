package app

import (
	"slices"
	"testing"
)

func TestNoteGraph(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	write(t, root, "Estudos.md", "---\ntags: [estudos, \"#hub\"]\ncreated: 2026-05-22\n---\nVeja [Go](Linguagens/Go%20basico.md) e [[Rust]].\n`[[Exemplo]]` e https://x.dev\n")
	write(t, root, "Linguagens/Go basico.md", "Volte para [[Estudos|o índice]].\n```\n[[Rust]]\n```\n")
	write(t, root, "Linguagens/Rust.md", "Sem links.")
	write(t, root, "Outra/Rust.md", "Mesmo nome, outra pasta.")
	write(t, root, "Mapa.canvas", `{"nodes":[{"id":"a","type":"file","file":"Linguagens/Rust.md"},{"id":"b","type":"text","text":"oi"}],"edges":[]}`)
	write(t, root, "Fluxo.draw", `{"version":1,"elements":[]}`)

	g, err := NoteGraph(site)
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{}
	for _, n := range g.Nodes {
		paths = append(paths, n.Path)
	}
	for _, p := range []string{"Estudos.md", "Linguagens/Go basico.md", "Linguagens/Rust.md", "Outra/Rust.md", "Mapa.canvas", "Fluxo.draw"} {
		if !slices.Contains(paths, p) {
			t.Fatalf("missing %s in %v", p, paths)
		}
	}
	if slices.Contains(paths, "_index.md") {
		t.Fatalf("a section's _index.md is not a note: %v", paths)
	}
	for _, n := range g.Nodes {
		if n.Path == "Estudos.md" && (!slices.Equal(n.Tags, []string{"estudos", "hub"}) || n.Created != "2026-05-22") {
			t.Fatalf("tags = %v, created = %q", n.Tags, n.Created)
		}
		if n.Path == "Fluxo.draw" && (n.Kind != "drawing" || n.Name != "Fluxo") {
			t.Fatalf("drawing = %+v", n)
		}
	}
	want := []GraphEdge{
		{"Estudos.md", "Linguagens/Go basico.md"},
		{"Estudos.md", "Linguagens/Rust.md"}, // [[Rust]]: the first one found
		{"Linguagens/Go basico.md", "Estudos.md"},
		{"Mapa.canvas", "Linguagens/Rust.md"},
	}
	for _, e := range want {
		if !slices.Contains(g.Edges, e) {
			t.Errorf("missing edge %v in %v", e, g.Edges)
		}
	}
	if len(g.Edges) != len(want) {
		t.Errorf("edges = %v (links in code count?)", g.Edges)
	}
}
