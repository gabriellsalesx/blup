package app

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/ui"
)

// covers are the example posts' cover images (made for the theme).
//
//go:embed samples/covers/*.jpg
var covers embed.FS

// note is a file written into the vault: path relative to the site folder.
type note struct{ path, body string }

// writeSamples fills an empty site folder with example notes that teach the
// basics by example (links, images, callouts, properties). For the docs model,
// sections lists the parts chosen (docs, roadmap, changelog).
func writeSamples(site *config.Site, sections []string) error {
	today := time.Now().Format("2006-01-02")
	attach := i18n.T("attachments_folder")
	notes := samples[site.Lang+"/"+site.Model]
	if site.Model == "blog" {
		// Covers go to an attachments folder inside the site folder, where
		// the example posts point to them.
		for _, name := range []string{"capa-1.jpg", "capa-2.jpg"} {
			b, err := covers.ReadFile("samples/covers/" + name)
			if err != nil {
				return err
			}
			dest := filepath.Join(site.ContentDir(), attach, name)
			if _, err := os.Stat(dest); err == nil {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(dest, b, 0o644); err != nil {
				return err
			}
		}
	}
	for _, n := range notes {
		body := strings.NewReplacer("{{date}}", today, "{{title}}", site.Title, "{{attach}}", attach).Replace(n.body)
		if _, err := obsidian.WriteNoteIfMissing(filepath.Join(site.ContentDir(), filepath.FromSlash(n.path)), body); err != nil {
			return err
		}
	}
	for _, kind := range sections {
		if err := writeSection(site, kind); err != nil {
			return err
		}
	}
	return nil
}

// ensureHome creates the home page note (_index.md) when it is missing.
func ensureHome(site *config.Site) error {
	for _, n := range samples[site.Lang+"/"+site.Model] {
		if n.path == "_index.md" {
			body := strings.ReplaceAll(n.body, "{{title}}", site.Title)
			created, err := obsidian.WriteNoteIfMissing(filepath.Join(site.ContentDir(), "_index.md"), body)
			if created {
				ui.OK(i18n.T("new.home_note"))
			}
			return err
		}
	}
	return nil
}

// writeTemplates adds note templates for the core Templates plugin, in the
// vault's templates folder (created and configured if there is none). The
// folder is excluded from the site.
func writeTemplates(site *config.Site) error {
	folder := obsidian.TemplatesFolder(site.Vault)
	if folder == "" {
		folder = i18n.T("templates_folder")
		if err := obsidian.SetTemplatesFolder(site.Vault, folder); err != nil {
			return err
		}
	}
	list := templates[site.Lang+"/"+site.Model]
	if site.Model == "docs" {
		list = docsTemplates[site.Lang]
	}
	for _, n := range list {
		if _, err := obsidian.WriteNoteIfMissing(filepath.Join(site.Vault, filepath.FromSlash(folder), n.path), n.body); err != nil {
			return err
		}
	}
	ui.OK(i18n.T("new.templates", folder))
	return nil
}

var templates = map[string][]note{
	"pt/blog": {{"Post do blog.md", "---\ntitle: \"{{title}}\"\ndate: {{date:YYYY-MM-DD}}\ndescription: \ntags: \ndraft: false\n---\n\n"}},
	"en/blog": {{"Blog post.md", "---\ntitle: \"{{title}}\"\ndate: {{date:YYYY-MM-DD}}\ndescription: \ntags: \ndraft: false\n---\n\n"}},
}

var samples = map[string][]note{
	"pt/blog": {
		{"_index.md", `---
title: "{{title}}"
button_text:
button_link:
---
Este é o texto da página inicial. Ele fica na nota **_index** da pasta do site: edite à vontade no Obsidian.

Para mostrar um botão no topo do site, preencha as propriedades **button_text** (o texto, como "Assinar") e **button_link** (o endereço).
`},
		{"Meu primeiro post.md", `---
title: Meu primeiro post
image: {{attach}}/capa-1.jpg
date: {{date}}
description: Um post de exemplo para você ver como o site funciona.
tags: [exemplo]
draft: false
---
Cada nota desta pasta vira uma página do site, e cada subpasta vira uma seção.

## O que você pode usar

- **Negrito**, *itálico*, ==marcador== e [links para outras notas](Como%20escrever.md).
- Imagens: arraste para a nota, como sempre faz no Obsidian.
- Tags nas propriedades da nota viram páginas de tag.

> [!tip] Dica
> Para esconder uma nota do site, mude a propriedade **draft** para verdadeiro.
`},
		{"Como escrever.md", `---
title: Como escrever
image: {{attach}}/capa-2.jpg
date: {{date}}
description: As propriedades que o site entende.
tags: [exemplo]
---
As propriedades no topo da nota controlam o site:

| Propriedade | Para que serve |
| --- | --- |
| title | Título da página (sem ela, usa o nome da nota) |
| date | Data do post |
| description | Resumo que aparece no Google e nas redes |
| tags | Assuntos do post |
| image | Imagem de capa nas redes sociais |
| draft | Verdadeiro esconde a nota do site |

> [!note]- Clique para abrir
> Callouts dobráveis do Obsidian também funcionam.
`},
	},
	"en/blog": {
		{"_index.md", `---
title: "{{title}}"
button_text:
button_link:
---
This is the home page text. It lives in the **_index** note of the site folder: edit it freely in Obsidian.

To show a button at the top of the site, fill in the properties **button_text** (the label, like "Subscribe") and **button_link** (the address).
`},
		{"My first post.md", `---
title: My first post
image: {{attach}}/capa-1.jpg
date: {{date}}
description: An example post so you can see how the site works.
tags: [example]
draft: false
---
Every note in this folder becomes a page, and every subfolder becomes a section.

## What you can use

- **Bold**, *italic*, ==highlight== and [links to other notes](How%20to%20write.md).
- Images: drag them into the note, as you always do in Obsidian.
- Tags in the note properties become tag pages.

> [!tip] Tip
> To hide a note from the site, set the **draft** property to true.
`},
		{"How to write.md", `---
title: How to write
image: {{attach}}/capa-2.jpg
date: {{date}}
description: The properties the site understands.
tags: [example]
---
The properties at the top of a note control the site:

| Property | What it does |
| --- | --- |
| title | Page title (without it, the note name is used) |
| date | Post date |
| description | Summary shown on Google and social networks |
| tags | Post topics |
| image | Cover image on social networks |
| draft | True hides the note from the site |

> [!note]- Click to open
> Obsidian's foldable callouts work too.
`},
	},
	"pt/docs": {
		{"_index.md", `---
title: "{{title}}"
description: Documentação, roadmap e novidades do projeto.
---
Este é o texto da página inicial. Ele fica na nota **_index** da pasta do site.

Cada seção do site é uma pasta: **Documentação**, **Roadmap** e **Changelog**. Para tirar uma seção, apague a pasta. Para adicionar, use "Personalizar o site" no Blup.
`},
	},
	"en/docs": {
		{"_index.md", `---
title: "{{title}}"
description: Docs, roadmap and news about the project.
---
This is the home page text. It lives in the **_index** note of the site folder.

Each section of the site is a folder: **Docs**, **Roadmap** and **Changelog**. To remove a section, delete its folder. To add one, use "Customize the site" in Blup.
`},
	},
}
