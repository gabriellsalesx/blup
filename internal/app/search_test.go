package app

import "testing"

func TestSearchNotes(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	write(t, root, "Ação rápida.md", "Nada aqui.")
	write(t, root, "Projetos/Plano.md", "---\ntitle: Ação\n---\n# Plano\n- Revisar a **ação** de março\nOutra linha\n")
	write(t, root, "Outra.md", "Sem a palavra.")

	got, err := SearchNotes(site, "acao")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Path != "Ação rápida.md" || !got[0].InName {
		t.Fatalf("hits = %+v", got)
	}
	if l := got[1].Lines; got[1].Path != "Projetos/Plano.md" || len(l) != 1 || l[0] != (Snippet{Before: "Revisar a ", Link: "ação", After: " de março"}) {
		t.Fatalf("plano = %+v", got[1])
	}
	if got, _ := SearchNotes(site, "a"); len(got) != 0 {
		t.Fatalf("one letter = %+v", got)
	}
}
