---
title: Autores e perfis
date: {{date-7}}
description: Uma nota por autor, com foto, cargo, redes e bio. Posts com mais de um autor.
tags: [Personalização]
image: "[[blup-autores.jpg]]"
authors: [Blup]
---
Este post foi "escrito" pelo **Blup**, o mascote. Repare na foto e no nome no topo: eles vêm da nota dele, na pasta **Autores**.

## A pasta de autores

A pasta **Autores** tem uma nota **_index** com a propriedade `blup: authors`. Ela avisa o Blup que as notas dali são perfis, não posts: a pasta não vira página nem aparece no menu.

Cada autor é uma nota com o nome da pessoa:

```yaml
---
photo: "[[ana.jpg]]"
role: Editora
social:
  - https://instagram.com/ana
  - https://github.com/ana
---
Ana escreve sobre livros e viagens.
```

- **photo**: a foto do perfil.
- **role**: o cargo ou função.
- **social**: links dos perfis, com ícone automático.
- O **texto da nota** é a bio.

Quando você criou o site, o Blup já criou a sua nota nesta pasta. Preencha a foto e a bio.

## Nos posts

```yaml
authors: [Ana]
authors: [Ana, Bruno]
```

Sem a propriedade, o post é do autor do site. Com mais de um autor, as fotos aparecem lado a lado e os nomes juntos.

## A página de cada autor

Clique no nome do autor de um post: a página dele mostra foto, cargo, bio, redes e todos os posts que ele escreveu.
