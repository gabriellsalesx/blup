---
title: Onde publicar
description: Blup Cloud, Cloudflare Workers ou GitHub Pages. Dois deles são de graça.
weight: 1
---
O Blup gera o site no seu computador e envia as páginas prontas para um destes lugares:

| Destino | Preço | Endereço | O que precisa |
| --- | --- | --- | --- |
| **Blup Cloud** | Nos planos pagos (a partir do Pro) | `seusite.getblup.site` | Só a conta do Blup. Nada para configurar |
| **Cloudflare Workers** | Grátis | `seusite.suaconta.workers.dev` | Uma conta grátis no Cloudflare |
| **GitHub Pages** | Grátis | `usuario.github.io/site` | Uma conta grátis no GitHub |

![O Blup lançando o site](../../Anexos/blup-publicar.jpg)

## Qual escolher?

- **Quer o mais simples?** O Blup Cloud: você escolhe o endereço e clica em Publicar. Veja [Sites pelo Blup Cloud](../Blup%20Cloud/Sites%20pelo%20Blup%20Cloud.md).
- **Quer de graça e rápido no mundo todo?** O Cloudflare Workers.
- **Já usa o GitHub?** O GitHub Pages.

Nos três, o site é rápido, tem HTTPS e funciona como app.

## Domínio próprio

Quer `meusite.com.br` no lugar do endereço do serviço? O domínio próprio é ligado só pelo Blup Cloud, nos planos Max e Ultra. Pelo Cloudflare e pelo GitHub, o site fica no endereço grátis. Veja [Domínio próprio](Domínio%20próprio.md).

## O que vai para o ar

O site é público por natureza. Só vai o que foi publicado:

- Rascunhos (`draft: true`) ficam de fora.
- Comentários entre `%%` ficam de fora.
- Dos anexos, só vão os que alguma nota usa.
- Posts com senha saem criptografados do seu computador.
- As notas dos cofres de anotações nunca são publicadas.

## Como publicar

- **No Blup Studio**: veja [Publicar pelo Studio](Publicar%20pelo%20Studio.md).
- **Na linha de comando**: veja [Publicar](../Linha%20de%20comando/Publicar.md). O Blup Cloud é só pelo Studio.
