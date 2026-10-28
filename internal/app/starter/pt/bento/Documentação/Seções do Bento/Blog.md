---
title: Blog
description: Posts com capa, tags, autores e posts relacionados.
weight: 5
---
O blog é uma pasta cuja **_index** tem `blup: blog`. Cada nota é um post, com o mesmo visual do tema Mochi.

```yaml
---
title: Como este site foi montado
date: 2026-10-25
description: Por dentro do modelo que vem com o Blup.
tags: [Bastidores]
image: "[[blup-bento.jpg]]"
authors: [Blup]
---
```

- O post mais novo vira o **destaque** do topo; para escolher outro, use `featured: true`.
- **image** é a capa. Sem ela, vale a primeira imagem do texto.
- **tags** viram filtros no topo da lista e páginas próprias.
- **authors** mostra foto e nome; cada autor ganha uma página. Veja [Autores](../Configuração/Autores.md).
- Cada post tem sumário, barra de leitura, botões de compartilhar e posts relacionados pelas tags.
