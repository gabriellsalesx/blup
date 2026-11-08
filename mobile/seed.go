package main

import (
	"os"
	"path/filepath"
	"time"
)

// The first vault of the phone starts with one note, in the app's language:
// how the app works, for whoever opens it for the first time. It can be
// edited or deleted like any other.
var welcome = map[string]struct{ path, body string }{
	"pt": {"Bem-vindo ao Blup.md", "# Bem-vindo ao Blup\n\nEsta é a sua primeira nota. Edite ou apague quando quiser.\n\n## O básico\n\n- Crie notas e pastas na tela de Notas.\n- Escreva em Markdown: **negrito**, *itálico*, listas e títulos.\n- Ligue uma nota a outra escrevendo o nome dela entre colchetes duplos, como `[[Ideias]]`.\n- Marque notas com tags, como `#trabalho`, e ache tudo depois na Busca.\n\n## Para começar\n\n- [x] Abrir o Blup\n- [ ] Criar a primeira nota\n- [ ] Conectar ao Blup Cloud\n\n## Blup Cloud\n\nEm Conta, conecte ao Blup Cloud para ter as mesmas notas no computador, com o Blup Studio. Elas são cifradas aqui no celular antes de sair: nem o Blup consegue ler.\n"},
	"en": {"Welcome to Blup.md", "# Welcome to Blup\n\nThis is your first note. Edit or delete it whenever you like.\n\n## The basics\n\n- Create notes and folders on the Notes screen.\n- Write in Markdown: **bold**, *italic*, lists and headings.\n- Link a note to another by writing its name in double brackets, like `[[Ideas]]`.\n- Mark notes with tags, like `#work`, and find everything later in Search.\n\n## To get started\n\n- [x] Open Blup\n- [ ] Create your first note\n- [ ] Connect to Blup Cloud\n\n## Blup Cloud\n\nIn Account, connect to Blup Cloud to have the same notes on your computer, with Blup Studio. They are encrypted here on the phone before they leave: not even Blup can read them.\n"},
	"es": {"Bienvenido a Blup.md", "# Bienvenido a Blup\n\nEsta es tu primera nota. Edítala o bórrala cuando quieras.\n\n## Lo básico\n\n- Crea notas y carpetas en la pantalla de Notas.\n- Escribe en Markdown: **negrita**, *cursiva*, listas y títulos.\n- Enlaza una nota con otra escribiendo su nombre entre corchetes dobles, como `[[Ideas]]`.\n- Marca notas con etiquetas, como `#trabajo`, y encuentra todo después en Buscar.\n\n## Para empezar\n\n- [x] Abrir Blup\n- [ ] Crear tu primera nota\n- [ ] Conectar a Blup Cloud\n\n## Blup Cloud\n\nEn Cuenta, conecta a Blup Cloud para tener las mismas notas en tu ordenador, con Blup Studio. Se cifran aquí en el móvil antes de salir: ni Blup puede leerlas.\n"},
}

func seed(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	w := welcome[appLang()]
	date := time.Now().Format("2006-01-02")
	head := "---\ncreated: " + date + "\nupdated: " + date + "\n---\n\n"
	return os.WriteFile(filepath.Join(dir, w.path), []byte(head+w.body), 0o600)
}
