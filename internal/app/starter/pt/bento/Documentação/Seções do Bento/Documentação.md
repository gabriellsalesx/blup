---
title: Documentação
description: Páginas organizadas em grupos, com menu lateral, sumário e navegação.
weight: 1
---
A documentação é a seção que você está lendo. Ela fica numa pasta da raiz cuja nota **_index** tem `blup: docs`.

## Como se organiza

- Cada **nota** é uma página.
- Cada **subpasta** é um grupo no menu lateral, com o título da _index dela.
- A **_index** da pasta da documentação é a página de abertura.
- A ordem segue a propriedade **weight** (menor primeiro) e, sem ela, o nome da nota.

```text
Documentação/
├── _index.md           blup: docs
├── Primeiros passos/
│   ├── _index.md       title: Primeiros passos, weight: 1
│   ├── Instalação.md   weight: 1
│   └── Criar o primeiro site.md   weight: 2
└── Escrevendo/
    └── _index.md       weight: 2
```

## O que cada página ganha

- **Menu lateral** com todos os grupos; o grupo atual fica aberto.
- **Nesta página**: os títulos `##` e `###`, ao lado do texto.
- **Trilha** (Início > Documentação > grupo) no topo.
- **Anterior e Próxima** no fim, seguindo a ordem do menu.
- **Busca** em todas as páginas do site.

> [!tip] Uma descrição curta ajuda
> A propriedade **description** aparece embaixo do título e no Google. Uma frase basta.
