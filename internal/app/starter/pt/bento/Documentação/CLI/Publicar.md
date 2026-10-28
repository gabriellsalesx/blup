---
title: Publicar
description: Cloudflare Pages ou GitHub Pages, de graça. Um comando a cada atualização.
weight: 3
---
Rode `blup publicar`. Na primeira vez, escolha onde. Os dois são gratuitos.

![O Blup lançando o site](../../Anexos/blup-publicar.jpg)

## Cloudflare Pages

Rápido no mundo todo, com endereço como `meusite.pages.dev`.

1. Crie uma conta grátis em dash.cloudflare.com, se não tiver.
2. O Blup abre o formulário da chave de acesso. Confira a permissão **Cloudflare Pages: Edit**, crie a chave e cole no terminal.
3. Escolha o nome do projeto: ele vira o endereço.

## GitHub Pages

Endereço como `seunome.github.io/meusite`.

1. Crie uma conta grátis em github.com, se não tiver.
2. O Blup abre o formulário da chave já preenchido. Em **Repository access**, escolha **All repositories**, gere a chave e cole no terminal.
3. Escolha o nome do repositório: ele vira o fim do endereço.

## Das próximas vezes

`blup publicar` envia só o que mudou e mostra o endereço. As chaves ficam guardadas só no seu computador; se uma vencer, o Blup pede outra.

## Domínio próprio

Escreva o domínio no **blup.toml** da pasta do site:

```toml
domain = "meusite.com.br"
```

- No GitHub Pages, o Blup configura o domínio sozinho; aponte o DNS do domínio para o GitHub.
- No Cloudflare Pages, o Blup avisa: adicione o domínio em **Custom domains** do projeto, no painel da Cloudflare.
