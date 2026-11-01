---
title: Vista previa y publicación
date: {{date-2}}
description: Mira el sitio mientras escribes y publícalo gratis en Cloudflare Pages o en GitHub Pages.
tags: [Primeros pasos, CLI]
image: "[[blup-publicar.jpg]]"
---
## La vista previa

Ejecuta `blup preview` (o elige la vista previa en el menú). El sitio se abre en el navegador y **se actualiza solo** cada vez que guardas una nota en Obsidian. Para detenerla, presiona Ctrl+C en la terminal.

La vista previa también muestra los borradores, para que los revises antes de publicar.

## Publicar

Ejecuta `blup publish`. La primera vez Blup pregunta dónde publicar. Los dos son gratuitos:

- **Cloudflare Pages**: rápido en todo el mundo, con una dirección como `tusitio.pages.dev`.
- **GitHub Pages**: una dirección como `tunombre.github.io/tusitio`.

Blup abre la página correcta para crear la clave de acceso, ya completada, y te explica dónde hacer clic. Pegas la clave en la terminal una vez; queda guardada solo en tu computadora.

Las siguientes veces, `blup publish` solo envía lo que cambió.

> [!tip] Publicar siempre es igual
> Escribiste, revisaste en la vista previa, ejecutaste `blup publish`. No hay servidor que mantener ni base de datos.

## Dominio propio

¿Quieres una dirección tuya, como `misitio.com`? Ejecuta `blup customize` y elige **Dominio propio**. Blup conecta el dominio donde el sitio está publicado, te muestra el registro DNS que debes crear donde compraste el dominio y activa HTTPS cuando todo está listo.

## Qué se publica

- Todas las notas de la carpeta del sitio, excepto:
  - notas con `draft: true`;
  - carpetas que empiezan con punto, la carpeta de plantillas (Templates) y la carpeta de autores, que solo se usa como datos de los perfiles.
- Las imágenes y los adjuntos que usan las notas, optimizados.
- La búsqueda, el feed RSS, el mapa del sitio y la tarjeta para compartir de cada página.
