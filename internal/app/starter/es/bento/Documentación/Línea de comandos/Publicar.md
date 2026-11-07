---
title: Publicar
description: Cloudflare Workers o GitHub Pages, gratis. Un comando en cada actualización.
weight: 3
---
Ejecuta `blup publish` (o `blup publicar`). La primera vez, elige dónde: Cloudflare Workers o GitHub Pages. Los dos son gratuitos. La publicación con Blup Cloud se hace en el Studio; mira [Dónde publicar](../Publicación/Dónde%20publicar.md).

![Blup lanzando el sitio](../../Adjuntos/blup-publicar.jpg)

## Cloudflare Workers

Rápido en todo el mundo, con una dirección como `tusitio.tucuenta.workers.dev`.

1. Crea una cuenta gratis en dash.cloudflare.com, si no tienes.
2. Blup abre el formulario de la clave de acceso. Revisa el permiso **Account › Workers Scripts › Edit**, crea la clave y pégala en la terminal.
3. Si la cuenta todavía no tiene una dirección en workers.dev, Blup pregunta su nombre. Se elige una vez y sirve para todos los sitios de la cuenta.
4. Elige el nombre del sitio: es el inicio de la dirección. Si la cuenta ya tiene otro sitio con ese nombre, Blup pide otro.

## GitHub Pages

Una dirección como `tunombre.github.io/tusitio`.

1. Crea una cuenta gratis en github.com, si no tienes.
2. Blup abre el formulario de la clave ya completado. En **Repository access**, elige **All repositories**, genera la clave y pégala en la terminal.
3. Elige el nombre del repositorio: se convierte en el final de la dirección.

## Las próximas veces

`blup publish` envía solo lo que cambió y muestra la dirección. Las claves se guardan solo en tu computadora; si una vence, Blup te pide otra.

## Dominio propio

Con Cloudflare Workers y con GitHub Pages, el sitio queda en la dirección gratis del servicio. Un dominio tuyo, como `tusitio.com`, se vincula solo con Blup Cloud, en los planes Max y Ultra, en la pantalla **Publicar** del Studio. Mira [Dominio propio](../Publicación/Dominio%20propio.md).
