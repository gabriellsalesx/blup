---
title: Publicar
description: Cloudflare Pages o GitHub Pages, gratis. Un comando en cada actualización.
weight: 3
---
Ejecuta `blup publish` (o `blup publicar`). La primera vez, elige dónde: Cloudflare Pages o GitHub Pages. Los dos son gratuitos. La publicación con Blup Cloud se hace en el Studio; mira [Dónde publicar](../Publicación/Dónde%20publicar.md).

![Blup lanzando el sitio](../../Adjuntos/blup-publicar.jpg)

## Cloudflare Pages

Rápido en todo el mundo, con una dirección como `tusitio.pages.dev`.

1. Crea una cuenta gratis en dash.cloudflare.com, si no tienes.
2. Blup abre el formulario de la clave de acceso. Revisa el permiso **Cloudflare Pages: Edit**, crea la clave y pégala en la terminal.
3. Elige el nombre del proyecto: se convierte en la dirección.

## GitHub Pages

Una dirección como `tunombre.github.io/tusitio`.

1. Crea una cuenta gratis en github.com, si no tienes.
2. Blup abre el formulario de la clave ya completado. En **Repository access**, elige **All repositories**, genera la clave y pégala en la terminal.
3. Elige el nombre del repositorio: se convierte en el final de la dirección.

## Las próximas veces

`blup publish` envía solo lo que cambió y muestra la dirección. Las claves se guardan solo en tu computadora; si una vence, Blup te pide otra.

## Dominio propio

Para que el sitio responda en una dirección tuya, como `tusitio.com` o `blog.tusitio.com`, ejecuta `blup customize` (o `blup personalizar`) y elige **Dominio propio**. Blup:

1. pregunta el dominio;
2. conecta el dominio donde está publicado el sitio, en Cloudflare Pages o en GitHub Pages;
3. muestra el registro DNS exacto que debes crear donde compraste el dominio, si falta alguno;
4. activa el HTTPS cuando todo está listo.

El DNS puede tardar de minutos a algunas horas. Para comprobarlo, vuelve a **Dominio propio** y elige **Comprobar si está funcionando**; cada `blup publish` también lo revisa. Con la misma opción cambias de dominio o vuelves a la dirección gratis.

> [!tip] Dominio raíz en Cloudflare Pages
> Un dominio raíz (`tusitio.com`, sin `www` ni `blog` delante) en Cloudflare Pages tiene que estar en tu cuenta de Cloudflare, lo cual es gratis. Si prefieres no cambiar eso, usa `www.tusitio.com`.
