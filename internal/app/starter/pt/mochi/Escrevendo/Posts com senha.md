---
title: Posts com senha
date: {{date-5}}
description: Proteja um post com senha, sem servidor e sem ninguém ler pelo código da página.
tags: [Obsidian, Recursos]
image: "[[blup-senha.jpg]]"
---
Qualquer post pode ter senha. Quem abrir vê o título e um campo para digitar a senha; o texto só aparece com ela.

## Como proteger

Adicione a propriedade **password** à nota:

```yaml
password: minha-senha
```

Para não repetir a mesma senha em vários posts, defina uma **senha geral** em `blup personalizar` e use:

```yaml
password: sim
```

Veja funcionando: [Um post trancado](Um%20post%20trancado.md). A senha é `blup`.

## É seguro?

O texto é **criptografado** quando o site é gerado (AES-256). Sem a senha, ninguém consegue ler, nem olhando o código da página. Não existe servidor guardando nada.

> [!warning] O que não é protegido
> As imagens dentro do post não são criptografadas. O título, a descrição e a capa continuam visíveis, porque aparecem nas listas.

## Esconder das listas

Para um post que só quem tem o link deve achar, use também `hidden: true`: ele sai da página inicial, das tags e do Google, mas continua no ar.
