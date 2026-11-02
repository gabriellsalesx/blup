---
title: Posts com senha
date: {{date-5}}
description: Proteja um post com senha, sem servidor e sem ninguém ler pelo código da página.
tags: [Escrita, Recursos]
image: "[[blup-senha.jpg]]"
---
Qualquer post pode ter senha. Quem abrir vê o título e um campo para digitar a senha; o texto só aparece com ela.

Veja funcionando: [Um post trancado](Um%20post%20trancado.md). A senha é `blup`.

## No Studio

No painel **Propriedades** do post, ligue **Senha** e escolha:

- **Senha própria**: digite a senha deste post.
- **Senha geral**: uma senha só para vários posts. Ela fica nas **Configurações** do Studio e vale para todos os seus sites. Se quiser, um site pode ter uma senha própria, que vale só para ele.

## No Obsidian

Adicione a propriedade **password** à nota:

```yaml
password: minha-senha
```

Para usar a senha geral, escreva:

```yaml
password: sim
```

Pela linha de comando, a senha geral se define em `blup personalizar`.

## É seguro?

O texto é **criptografado** no seu computador, quando o site é gerado (AES-256). Sem a senha, ninguém consegue ler, nem olhando o código da página. Não existe servidor guardando nada, e a senha nunca vai para o site.

> [!warning] O que não é protegido
> As imagens dentro do post não são criptografadas. O título, a descrição e a capa continuam visíveis, porque aparecem nas listas. Posts com senha também não ganham narração nem entram no podcast, porque o áudio entregaria o texto.

## Esconder das listas

Para um post que só quem tem o link deve achar, ligue também **Escondida** (`hidden: true`): ele sai da página inicial, das tags e do Google, mas continua no ar.
