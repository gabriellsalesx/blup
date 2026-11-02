---
title: Autores e perfis
date: {{date-7}}
description: Um perfil por autor, com foto, cargo, redes e bio. Posts com mais de um autor.
tags: [Personalização, Studio]
image: "[[blup-autores.jpg]]"
authors: [Blup]
---
Este post foi "escrito" pelo **Blup**, o mascote. Repare na foto e no nome no topo: eles vêm do perfil dele, na pasta **Autores**.

## No Studio

A tela **Autores**, no menu ao lado, lista os perfis. Em cada um:

- **Foto**: uma imagem sua ou um dos avatares prontos.
- **Cargo**: a função, como "Editora".
- **Redes sociais**: escolha a rede e complete o endereço do perfil.
- **Bio**: um texto curto sobre a pessoa.

Quando você criou o site, o Blup já criou o seu perfil. Preencha a foto e a bio.

Para dizer quem escreveu um post, use o campo **Autores** no painel **Propriedades** do post. Comece a digitar e escolha o nome.

## Direto nas notas

Você também pode editar os perfis em qualquer editor de texto. A pasta **Autores** tem uma nota **_index** com a propriedade `blup: authors`. Ela avisa o Blup que as notas dali são perfis, não posts: a pasta não vira página nem aparece no menu.

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

O texto da nota é a bio. Nos posts:

```yaml
authors: [Ana]
authors: [Ana, Bruno]
```

## Como aparece

Sem autor, o post é do autor do site. Com mais de um, as fotos aparecem lado a lado e os nomes juntos.

Clique no nome do autor de um post: a página dele mostra foto, cargo, bio, redes e todos os posts que ele escreveu. Num blog em vários idiomas, o cargo e a bio também são traduzidos.
