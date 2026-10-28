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

Para o site responder num endereço seu, como `meusite.com.br` ou `blog.meusite.com.br`, rode `blup personalizar` e escolha **Domínio próprio**. O Blup:

1. pergunta o domínio;
2. liga o domínio onde o site está publicado, no Cloudflare Pages ou no GitHub Pages;
3. mostra o registro de DNS exato para criar onde você comprou o domínio, se faltar algum;
4. liga o HTTPS quando estiver tudo pronto.

O DNS pode levar de minutos a algumas horas. Para conferir, volte em **Domínio próprio** e escolha **Ver se já está funcionando**; cada `blup publicar` também confere. Pela mesma opção você troca de domínio ou volta ao endereço grátis.

> [!tip] Domínio raiz no Cloudflare Pages
> Um domínio raiz (`meusite.com.br`, sem `www` ou `blog` na frente) no Cloudflare Pages precisa estar na sua conta Cloudflare, o que é grátis. Se preferir não mudar isso, use `www.meusite.com.br`.
