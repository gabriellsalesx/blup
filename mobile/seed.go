package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// The example vault of the app, with the age of each note in days.
var examples = []struct {
	path string
	days int
	tags []string
	body string
}{
	{"Diário/Reunião de sexta.md", 0, []string{"trabalho", "mobile"}, "Fechamos o escopo do [[Blup no celular]] e o que entra primeiro.\n\n- [x] Mostrar as telas no emulador\n- [ ] Testar o editor com o teclado\n- [ ] Decidir a barra de navegação\n"},
	{"Projetos/Blup no celular.md", 1, []string{"mobile", "produto"}, "O objetivo é levar as notas para o bolso, sem perder o que já funciona no computador.\n\n## Primeira versão\n\n- Notas e pastas\n- Busca\n- Calendário\n- Sincronização com o [[Blup Cloud]]\n\n## Fica para depois\n\nPublicar sites e narração.\n"},
	{"Projetos/Blup Cloud.md", 2, []string{"produto"}, "Guarda as notas com a senha que só você conhece.\n\n> Sem a senha ou a chave de recuperação, ninguém abre as notas.\n\nVeja também [[Blup no celular]].\n"},
	{"Ideias/Títulos para o blog.md", 4, []string{"escrita"}, "1. Como escrever todo dia sem travar\n2. Notas que viram site\n3. O que guardar e o que apagar\n"},
	{"Ideias/Lista de leitura.md", 7, []string{"leitura"}, "- [ ] Como fazer anotações melhores\n- [x] Escrever para pensar\n- [ ] Um bom livro de design\n"},
	{"Projetos/Blup/Mobile/Telas do app.md", 1, []string{"mobile"}, "Notas, Busca, Calendário e Conta na barra de baixo.\n"},
	{"Projetos/Blup/Mobile/Testes no Android.md", 2, []string{"mobile"}, "- [x] Editor com teclado\n- [ ] Celular físico\n"},
	{"Projetos/Blup/Site/Textos da página inicial.md", 3, []string{"escrita"}, "Escreva no Blup e publique sem complicação.\n"},
	{"Projetos/Clientes/Modelo de proposta.md", 5, []string{"trabalho"}, "Objetivo, prazo e valor, nessa ordem.\n"},
	{"Estudos/Go/Canais.md", 6, []string{"leitura"}, "Um canal liga duas goroutines sem memória compartilhada.\n"},
	{"Estudos/Inglês/Verbos irregulares.md", 8, []string{"leitura"}, "- go, went, gone\n- write, wrote, written\n"},
	{"Ideias/Viagens/Roteiro de férias.md", 9, []string{}, "Três dias na serra e dois na praia.\n"},
	{"Diário/Domingo.md", 12, []string{"diario"}, "Dia calmo. Separei as ideias da semana em [[Títulos para o blog]].\n"},
}

func seed(dir string) error {
	for _, e := range examples {
		path := filepath.Join(dir, e.path)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		when := time.Now().AddDate(0, 0, -e.days)
		date := when.Format("2006-01-02")
		head := "---\ncreated: " + date + "\nupdated: " + date + "\n---\n\n"
		if err := os.WriteFile(path, []byte(head+e.body), 0o600); err != nil {
			return err
		}
		if len(e.tags) > 0 {
			if err := obsidian.SetProperties(path, obsidian.Prop{Key: "tags", Value: e.tags}); err != nil {
				return err
			}
		}
		if err := os.Chtimes(path, when, when); err != nil {
			return err
		}
	}
	return nil
}
