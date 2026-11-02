---
title: Activar y desactivar secciones
description: Todas las secciones son opcionales. Agrégalas, quítalas y restáuralas cuando quieras.
weight: 9
---
## Con Blup Studio

Haz clic en **Secciones** en la barra lateral (solo en Bento). La pantalla muestra cada parte del sitio de producto, con una línea sobre ella:

| Sección | Qué es |
| --- | --- |
| Documentación | Páginas con barra lateral e índice |
| Roadmap | Tablero con planeado, en curso y hecho |
| Changelog | Qué cambió en cada versión |
| Centro de ayuda | Colecciones de artículos con búsqueda |
| Blog | Noticias con portadas y etiquetas |
| Feedback | Tablero donde quien visita pide funciones, reporta errores y vota. Solo en Blup Studio |

- **Agregar** crea la carpeta de la sección con notas de ejemplo que enseñan a usarla.
- Las que ya están en el sitio aparecen como **En el sitio**. Quitar una sección envía la carpeta a la **Papelera**, y el aviso tiene **Deshacer**.
- Una sección quitada se puede restaurar desde la Papelera.

La vista previa se actualiza sola después de cada cambio.

## Con la línea de comandos

Ejecuta `blup customize` y elige **Secciones del sitio**. Blup muestra las que faltan y crea la carpeta de cada una, con notas de ejemplo. Para quitar una, elimina la carpeta. El Feedback no aparece en esta lista.

## A mano

Una carpeta en la raíz del sitio con una nota **_index** que tenga la propiedad **blup**:

| Sección | Propiedad |
| --- | --- |
| Documentación | `blup: docs` |
| Roadmap | `blup: roadmap` |
| Changelog | `blup: changelog` |
| Centro de ayuda | `blup: help` |
| Blog | `blup: blog` |
| Feedback | `blup: feedback` |
| Perfiles de los autores | `blup: authors` |

## Renombrar y reordenar

- El nombre de la carpeta es libre: "Docs", "Guía", "Manual"... lo que cuenta es la propiedad **blup**.
- El **title** de la _index es el nombre en el menú y arriba de la sección.
- El **weight** de la _index ordena las secciones en el menú automático y en las tarjetas de la página de inicio.
