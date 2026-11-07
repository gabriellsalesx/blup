---
title: Dónde publicar
description: Blup Cloud, Cloudflare Workers o GitHub Pages. Dos de ellos son gratis.
weight: 1
---
Blup genera el sitio en tu computadora y envía las páginas listas a uno de estos lugares:

| Destino | Precio | Dirección | Qué necesita |
| --- | --- | --- | --- |
| **Blup Cloud** | En los planes de pago (desde Pro) | `tusitio.getblup.site` | Solo la cuenta de Blup. Nada que configurar |
| **Cloudflare Workers** | Gratis | `tusitio.tucuenta.workers.dev` | Una cuenta gratis en Cloudflare |
| **GitHub Pages** | Gratis | `usuario.github.io/sitio` | Una cuenta gratis en GitHub |

![Blup lanzando el sitio](../../Adjuntos/blup-publicar.jpg)

## ¿Cuál elegir?

- **¿Quieres lo más simple?** Blup Cloud: eliges la dirección y haces clic en Publicar. Mira [Sitios con Blup Cloud](../Blup%20Cloud/Sitios%20con%20Blup%20Cloud.md).
- **¿Quieres algo gratis y rápido en todo el mundo?** Cloudflare Workers.
- **¿Ya usas GitHub?** GitHub Pages.

En los tres, el sitio es rápido, tiene HTTPS y funciona como app.

## Dominio propio

¿Quieres `misitio.com` en lugar de la dirección del servicio? El dominio propio se vincula solo con Blup Cloud, en los planes Max y Ultra. Con Cloudflare y con GitHub, el sitio queda en la dirección gratis. Mira [Dominio propio](Dominio%20propio.md).

## Lo que se publica

El sitio es público por naturaleza. Solo va lo que se publicó:

- Los borradores (`draft: true`) quedan fuera.
- Los comentarios entre `%%` quedan fuera.
- De los adjuntos, solo van los que alguna nota usa.
- Los posts con contraseña salen cifrados de tu computadora.
- Las notas de las bóvedas de notas nunca se publican.

## Cómo publicar

- **En Blup Studio**: mira [Publicar desde el Studio](Publicar%20desde%20el%20Studio.md).
- **En la línea de comandos**: mira [Publicar](../Línea%20de%20comandos/Publicar.md). Blup Cloud es solo desde el Studio.
