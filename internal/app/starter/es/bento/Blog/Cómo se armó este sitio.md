---
title: Cómo se armó este sitio
date: {{date-3}}
description: Por dentro de la plantilla que viene con Blup, carpeta por carpeta.
tags: [Detrás de escena]
image: "[[blup-bento.jpg]]"
authors: [Blup]
---
Este post lo "escribió" **Blup**, la mascota. La foto y el nombre vienen de su nota en la carpeta **Autores**, y la propiedad `authors: [Blup]` conecta los dos.

## Una carpeta para cada sección

En la raíz de la carpeta del sitio, cada carpeta con una nota **_index** y la propiedad **blup** se convierte en una sección:

| Carpeta | Propiedad | Se convierte en |
| --- | --- | --- |
| Documentación | `blup: docs` | Documentación con menú lateral |
| Roadmap | `blup: roadmap` | Tablero con tres columnas |
| Changelog | `blup: changelog` | Lista de versiones |
| Ayuda | `blup: help` | Centro de ayuda |
| Blog | `blup: blog` | Este blog |
| Autores | `blup: authors` | Perfiles (sin página) |

## La página de inicio

La nota **_index** de la carpeta del sitio es la página de inicio: el título, la descripción y el texto aparecen arriba, y las secciones aparecen en tarjetas, en el orden del **weight** de cada una.

## Las imágenes

Las portadas están en la carpeta **Adjuntos** y entran en los posts por la propiedad `image`. Blup genera versiones más pequeñas y en WebP de cada una.

## ¿Y lo demás?

Búsqueda, tema claro y oscuro, tarjetas para redes sociales, feed RSS y mapa del sitio vienen listos, sin configurar nada.
