---
title: Sites em vários idiomas
description: O mesmo site em português, inglês e espanhol, com seletor de idioma no topo.
weight: 5
---
Um site do Blup é escrito num **idioma principal** (português, inglês ou espanhol) e pode ser publicado também nos outros dois. Cada nota ganha uma tradução, e o site mostra um **seletor de idioma** no topo.

> [!info] Só no Blup Studio
> A tradução é feita no Studio, por IA na nuvem ou no seu computador. Veja quem traduz e quanto custa em [Tradução](../Recursos%20com%20IA/Tradução.md).

## Traduzir uma nota

1. Abra a nota no Studio.
2. No bloco **Idiomas**, nas propriedades, clique em **Traduzir** ao lado do idioma.
3. O Studio mostra o andamento ("Traduzindo, parte 1 de 3…") e avisa quando a tradução fica pronta.
4. **Abrir** leva à tradução, para você revisar.

A IA traduz o título, a descrição e o texto. Código, links, imagens e o tipo dos callouts não mudam.

## Traduzir o site inteiro

Clique em **Idiomas** na faixa lateral. A tela mostra, para cada idioma, quantas notas estão **traduzidas**, **desatualizadas** e **sem tradução**.

- **Traduzir o que falta** traduz de uma vez todas as notas sem tradução.
- **Atualizar as traduções** refaz as que ficaram para trás.

## Onde fica cada tradução

A tradução fica numa pasta com o código do idioma (`en`, `es` ou `pt`), **ao lado da nota original**, com o nome traduzido:

```text
Blog/
├── Como publicar.md        original, em português
├── en/
│   └── How to publish.md   tradução em inglês
└── es/
    └── Cómo publicar.md    tradução em espanhol
```

A original e as traduções ficam ligadas pela propriedade `translationKey`, que o Studio põe sozinho. No site, a página traduzida é a mesma página no outro idioma: o seletor leva de uma para a outra. Um link para uma nota ainda sem tradução cai na original.

## O que também é traduzido

- Na nota inicial: a descrição do site, o texto do botão e os nomes do menu.
- Em qualquer nota: o título curto (`linkTitle`), a categoria e as tags.
- A nota _index de cada seção dá o nome traduzido da seção.
- Os perfis dos autores: o cargo e a bio.
- No quadro de Feedback: os textos dos pedidos de exemplo.

## Tradução desatualizada

Quando você muda a original, a tradução aparece como **Desatualizada: a original mudou**. Clique em **Traduzir de novo**.

## O selo "Traduzido com IA"

Uma tradução feita pela IA mostra, junto do título, um selo **Traduzido com IA**, que abre um balão com o idioma original e o link para ele. Revisou? Clique em **Marcar como revisada** e o selo sai.

## Comentários e votos

Todas as versões de uma página dividem a mesma conversa: votos e reações somam os leitores de todos os idiomas. Um comentário mostra em que idioma foi escrito, como "comentado em English".

> [!tip] Tradução à mão
> Dá para traduzir à mão: crie a pasta `en` ao lado da nota e escreva a tradução lá, com o mesmo nome de arquivo da original (ou a mesma `translationKey`). Os idiomas publicados ficam em `translations` no blup.toml, que o Studio preenche ao traduzir.
