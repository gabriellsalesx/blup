---
title: Redes sociais
description: Os ícones do rodapé, com um botão para cada rede.
weight: 3
---
Os perfis do site aparecem como ícones no rodapé. Os autores também podem ter as redes deles, na página de cada um.

## Pelo Blup Studio

Na tela **Site**, bloco **Redes sociais**:

1. Clique no botão da rede: **Instagram**, **X**, **LinkedIn**, **YouTube**, **Discord**, **GitHub**, **Facebook**, **WhatsApp** ou **e-mail**.
2. O Studio acrescenta uma linha com o começo do endereço, como `https://instagram.com/`, e põe o cursor no fim.
3. Complete com o seu perfil (ou o e-mail).

Se faltar o resto do endereço, o Studio avisa: uma linha só com o começo não vai para o site.

Na tela **Autores**, o perfil de cada pessoa tem os mesmos botões.

## Pelo Obsidian

Na nota **_index** da pasta do site:

```yaml
social:
  - https://instagram.com/acme
  - https://github.com/acme
  - https://wa.me/5585999999999
  - mailto:oi@acme.com
```

O Blup reconhece a rede pelo endereço e põe o ícone certo. Endereços que ele não conhece ganham um ícone de link.

## Pela linha de comando

`blup personalizar` > redes sociais pergunta os endereços um a um.
