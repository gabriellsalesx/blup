---
title: Publicar
description: Cloudflare Workers ou GitHub Pages, de graça. Um comando a cada atualização.
weight: 3
---
Rode `blup publicar`. Na primeira vez, escolha onde: Cloudflare Workers ou GitHub Pages. Os dois são gratuitos. A publicação pelo Blup Cloud é feita no Studio; veja [Onde publicar](../Publicação/Onde%20publicar.md).

![O Blup lançando o site](../../Anexos/blup-publicar.jpg)

## Cloudflare Workers

Rápido no mundo todo, com endereço como `meusite.suaconta.workers.dev`.

1. Crie uma conta grátis em dash.cloudflare.com, se não tiver.
2. O Blup abre o formulário da chave de acesso. Confira a permissão **Account › Workers Scripts › Edit**, crie a chave e cole no terminal.
3. Se a conta ainda não tiver um endereço no workers.dev, o Blup pergunta o nome dele. Ele é escolhido uma vez e vale para todos os sites da conta.
4. Escolha o nome do site: ele vira o começo do endereço. Se já existir outro site com esse nome na conta, o Blup pede outro.

## GitHub Pages

Endereço como `seunome.github.io/meusite`.

1. Crie uma conta grátis em github.com, se não tiver.
2. O Blup abre o formulário da chave já preenchido. Em **Repository access**, escolha **All repositories**, gere a chave e cole no terminal.
3. Escolha o nome do repositório: ele vira o fim do endereço.

## Das próximas vezes

`blup publicar` envia só o que mudou e mostra o endereço. As chaves ficam guardadas só no seu computador; se uma vencer, o Blup pede outra.

## Domínio próprio

Pelo Cloudflare Workers e pelo GitHub Pages, o site fica no endereço grátis do serviço. Um domínio seu, como `meusite.com.br`, é ligado só pelo Blup Cloud, nos planos Max e Ultra, na tela **Publicar** do Studio. Veja [Domínio próprio](../Publicação/Domínio%20próprio.md).
