package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// The docs model is made of sections, each a top-level folder whose _index
// note has the property "blup: docs|roadmap|changelog". Deleting the folder
// removes the section; adding one (by hand or with the CLI) brings it back.
var docsKinds = []string{"docs", "roadmap", "changelog"}

var kindProperty = regexp.MustCompile(`(?m)^blup:\s*["']?(docs|roadmap|changelog)["']?\s*$`)

// existingSections maps each section kind found in the site folder to its
// folder name.
func existingSections(site *config.Site) map[string]string {
	found := map[string]string{}
	entries, err := os.ReadDir(site.ContentDir())
	if err != nil {
		return found
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(site.ContentDir(), e.Name(), "_index.md"))
		if err != nil {
			continue
		}
		if m := kindProperty.FindSubmatch(b); m != nil {
			found[string(m[1])] = e.Name()
		}
	}
	return found
}

// writeSection creates a section folder with example notes. It does nothing
// when the section already exists.
func writeSection(site *config.Site, kind string) error {
	if _, ok := existingSections(site)[kind]; ok {
		return nil
	}
	folder := i18n.T("section_folder." + kind)
	today := time.Now()
	r := strings.NewReplacer(
		"{{date}}", today.Format("2006-01-02"),
		"{{date-1}}", today.AddDate(0, -1, 0).Format("2006-01-02"),
		"{{title}}", site.Title,
	)
	for _, n := range sectionSamples[site.Lang+"/"+kind] {
		path := filepath.Join(site.ContentDir(), folder, filepath.FromSlash(n.path))
		if _, err := obsidian.WriteNoteIfMissing(path, r.Replace(n.body)); err != nil {
			return err
		}
	}
	return nil
}

var sectionSamples = map[string][]note{
	"pt/docs": {
		{"_index.md", `---
title: Documentação
blup: docs
weight: 1
---
Bem-vindo à documentação. Esta é a primeira página: o texto fica na nota **_index** da pasta Documentação.

Cada nota desta pasta vira uma página e cada subpasta vira um grupo no menu lateral.
`},
		{"Primeiros passos/_index.md", `---
title: Primeiros passos
weight: 1
---
`},
		{"Primeiros passos/Instalação.md", `---
title: Instalação
description: Do zero ao primeiro uso.
weight: 1
---
## Requisitos

Liste aqui o que a pessoa precisa antes de começar.

## Passo a passo

1. Baixe o programa.
2. Abra e siga as instruções.

> [!tip] Dica
> Use a propriedade **weight** para escolher a ordem no menu: números menores aparecem primeiro.
`},
		{"Primeiros passos/Configuração.md", `---
title: Configuração
weight: 2
---
## Opções

| Opção | O que faz |
| --- | --- |
| tema | Claro, escuro ou do sistema |
| idioma | Português ou inglês |
`},
		{"Guias/_index.md", `---
title: Guias
weight: 2
---
`},
		{"Guias/Escrevendo páginas.md", `---
title: Escrevendo páginas
weight: 1
---
## Títulos viram o sumário

Os títulos de nível 2 e 3 aparecem em **Nesta página**, à direita.

### Callouts

> [!warning] Atenção
> Os callouts do Obsidian viram caixas de destaque.

## Fórmulas

$$
a^2 + b^2 = c^2
$$
`},
	},
	"en/docs": {
		{"_index.md", `---
title: Docs
blup: docs
weight: 1
---
Welcome to the docs. This is the first page: its text lives in the **_index** note of the Docs folder.

Every note in this folder becomes a page and every subfolder becomes a group in the sidebar.
`},
		{"Getting started/_index.md", `---
title: Getting started
weight: 1
---
`},
		{"Getting started/Installation.md", `---
title: Installation
description: From zero to first use.
weight: 1
---
## Requirements

List here what people need before they start.

## Step by step

1. Download the program.
2. Open it and follow the instructions.

> [!tip] Tip
> Use the **weight** property to order the sidebar: lower numbers come first.
`},
		{"Getting started/Configuration.md", `---
title: Configuration
weight: 2
---
## Options

| Option | What it does |
| --- | --- |
| theme | Light, dark or system |
| language | Portuguese or English |
`},
		{"Guides/_index.md", `---
title: Guides
weight: 2
---
`},
		{"Guides/Writing pages.md", `---
title: Writing pages
weight: 1
---
## Headings become the table of contents

Level 2 and 3 headings show up in **On this page**, on the right.

### Callouts

> [!warning] Heads up
> Obsidian callouts become highlighted boxes.

## Math

$$
a^2 + b^2 = c^2
$$
`},
	},
	"pt/roadmap": {
		{"_index.md", `---
title: Roadmap
blup: roadmap
weight: 2
description: O que estamos planejando e construindo.
---
`},
		{"Modo offline.md", `---
title: Modo offline
description: Usar o app sem internet e sincronizar depois.
status: planejado
category: Novidades
---
Cada nota desta pasta é um item do roadmap. A propriedade **status** escolhe a coluna: planejado, em andamento ou concluído.
`},
		{"Busca mais rápida.md", `---
title: Busca mais rápida
description: Resultados enquanto você digita.
status: em andamento
category: Melhorias
---
Descreva aqui o que vai mudar e por quê.
`},
		{"Tema escuro.md", `---
title: Tema escuro
description: Claro, escuro ou igual ao sistema.
status: concluído
category: Design
---
Itens concluídos ficam na última coluna.
`},
	},
	"en/roadmap": {
		{"_index.md", `---
title: Roadmap
blup: roadmap
weight: 2
description: What we are planning and building.
---
`},
		{"Offline mode.md", `---
title: Offline mode
description: Use the app without internet and sync later.
status: planned
category: Features
---
Every note in this folder is a roadmap item. The **status** property picks the column: planned, in progress or done.
`},
		{"Faster search.md", `---
title: Faster search
description: Results while you type.
status: in progress
category: Improvements
---
Describe what will change and why.
`},
		{"Dark theme.md", `---
title: Dark theme
description: Light, dark or matching the system.
status: done
category: Design
---
Completed items go to the last column.
`},
	},
	"pt/changelog": {
		{"_index.md", `---
title: Changelog
blup: changelog
weight: 3
description: Novidades, melhorias e correções de cada versão.
---
`},
		{"v1.1.0.md", `---
title: Versão 1.1.0
date: {{date}}
tags: [Novidade, Melhoria]
---
- Tema escuro, claro ou igual ao sistema.
- Busca mais rápida.

> [!note] Como escrever o changelog
> Cada nota desta pasta é uma versão. Use **date** para a data e **tags** para marcar o tipo de mudança.
`},
		{"v1.0.0.md", `---
title: Versão 1.0.0
date: {{date-1}}
tags: [Lançamento]
---
Primeira versão pública.
`},
	},
	"en/changelog": {
		{"_index.md", `---
title: Changelog
blup: changelog
weight: 3
description: New features, improvements and fixes in each version.
---
`},
		{"v1.1.0.md", `---
title: Version 1.1.0
date: {{date}}
tags: [New, Improvement]
---
- Dark, light or system theme.
- Faster search.

> [!note] How to write the changelog
> Every note in this folder is a version. Use **date** for the date and **tags** for the kind of change.
`},
		{"v1.0.0.md", `---
title: Version 1.0.0
date: {{date-1}}
tags: [Launch]
---
First public version.
`},
	},
}

var docsTemplates = map[string][]note{
	"pt": {
		{"Página da documentação.md", "---\ntitle: \"{{title}}\"\ndescription: \nweight: 10\n---\n\n"},
		{"Item do roadmap.md", "---\ntitle: \"{{title}}\"\ndescription: \nstatus: planejado\ncategory: \n---\n\n"},
		{"Versão do changelog.md", "---\ntitle: \"{{title}}\"\ndate: {{date:YYYY-MM-DD}}\ntags: \n---\n\n"},
	},
	"en": {
		{"Docs page.md", "---\ntitle: \"{{title}}\"\ndescription: \nweight: 10\n---\n\n"},
		{"Roadmap item.md", "---\ntitle: \"{{title}}\"\ndescription: \nstatus: planned\ncategory: \n---\n\n"},
		{"Changelog version.md", "---\ntitle: \"{{title}}\"\ndate: {{date:YYYY-MM-DD}}\ntags: \n---\n\n"},
	},
}
