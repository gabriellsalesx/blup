---
title: Estrutura das pastas
description: Como as pastas e notas do cofre viram as páginas do site.
weight: 3
---
## O mapa

```text
Site/                     a pasta do site
├── _index.md             página inicial e configurações do site
├── Documentação/         seção de documentação (blup: docs)
│   ├── _index.md         abertura da documentação
│   └── Primeiros passos/ grupo no menu lateral
├── Roadmap/              quadro do roadmap (blup: roadmap)
├── Changelog/            lista de versões (blup: changelog)
├── Ajuda/                central de ajuda (blup: help)
├── Blog/                 blog (blup: blog)
├── Autores/              perfis dos autores (blup: authors)
└── Anexos/               imagens
```

## As regras

- Cada **nota** vira uma página.
- Cada **pasta** vira um grupo ou seção.
- A nota **_index** de uma pasta dá título, descrição e texto de abertura a ela.
- No Bento, a propriedade **blup** na _index de uma pasta da raiz diz que tipo de seção ela é: `docs`, `roadmap`, `changelog`, `help`, `blog` ou `authors`. O nome da pasta é livre.

## O que fica de fora do site

- Notas com `draft: true`.
- Pastas que começam com ponto, como `.obsidian`.
- A pasta de modelos (Templates).
- A pasta de autores: ela vira só os dados dos perfis.

> [!note] Endereços das páginas
> O endereço vem do nome da pasta e da nota, sem acentos e com hífens: a nota "Estrutura das pastas" em "Primeiros passos" vira `/documentacao/primeiros-passos/estrutura-das-pastas/`.
