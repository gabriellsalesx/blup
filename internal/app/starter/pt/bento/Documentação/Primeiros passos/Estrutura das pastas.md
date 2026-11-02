---
title: Estrutura das pastas
description: Como as pastas e notas viram as páginas do site.
weight: 5
---
## O mapa

```text
Site/                     a pasta das notas do site
├── _index.md             página inicial e configurações do site
├── Documentação/         seção de documentação (blup: docs)
│   ├── _index.md         abertura da documentação
│   └── Primeiros passos/ grupo no menu lateral
├── Roadmap/              quadro do roadmap (blup: roadmap)
├── Changelog/            lista de versões (blup: changelog)
├── Ajuda/                central de ajuda (blup: help)
├── Blog/                 blog (blup: blog)
├── Feedback/             quadro de pedidos (blup: feedback)
├── Autores/              perfis dos autores (blup: authors)
└── Anexos/               imagens, áudios e outros arquivos
```

## As regras

- Cada **nota** vira uma página.
- Cada **pasta** vira um grupo ou seção.
- A nota **_index** de uma pasta dá título, descrição e texto de abertura a ela.
- No Bento, a propriedade **blup** na _index de uma pasta da raiz diz que tipo de seção ela é: `docs`, `roadmap`, `changelog`, `help`, `blog`, `feedback` ou `authors`. O nome da pasta é livre.
- Uma pasta chamada `en`, `es` ou `pt` ao lado de uma nota guarda a **tradução** dela. Veja [Sites em vários idiomas](../Configuração%20do%20site/Sites%20em%20vários%20idiomas.md).

## O que fica de fora do site

- Notas com `draft: true` (aparecem só na prévia).
- Pastas que começam com ponto, como `.obsidian` e `.blup`.
- A pasta de modelos do Obsidian (Modelos).
- A pasta de autores: ela vira só os dados dos perfis.
- Os guias para agentes de IA (`AGENTS.md`, `CLAUDE.md` e `GEMINI.md`), que o Studio cria ao abrir o terminal.
- Anexos que nenhuma nota usa.

## Pastas que o Blup usa por dentro

- **.blup**, dentro da pasta das notas: guarda o arquivo `site.toml`, a parte das configurações que viaja com a sincronização.
- **A pasta do site** (fora das notas): o `blup.toml`, os arquivos do gerador, a lixeira do Studio e o estado da sincronização. Você não precisa abrir.

> [!note] Endereços das páginas
> O endereço vem do nome da pasta e da nota, sem acentos e com hífens: a nota "Estrutura das pastas" em "Primeiros passos" vira `/documentacao/primeiros-passos/estrutura-das-pastas/`.
