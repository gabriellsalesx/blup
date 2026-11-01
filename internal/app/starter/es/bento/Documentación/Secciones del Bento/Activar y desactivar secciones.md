---
title: Activar y desactivar secciones
description: Todas las secciones son opcionales. Agrégalas con Blup, quítalas borrando la carpeta.
weight: 6
---
## Agregar

Ejecuta `blup customize` y elige **Secciones del sitio (docs, roadmap, changelog)**. Blup muestra las que faltan y crea la carpeta de cada una, con notas de ejemplo.

También puedes crearlas a mano: una carpeta en la raíz del sitio con una nota **_index** que tenga la propiedad **blup**:

| Sección | Propiedad |
| --- | --- |
| Documentación | `blup: docs` |
| Roadmap | `blup: roadmap` |
| Changelog | `blup: changelog` |
| Centro de ayuda | `blup: help` |
| Blog | `blup: blog` |
| Perfiles de los autores | `blup: authors` |

## Quitar

Borra la carpeta en Obsidian. Listo: la sección sale del sitio y del menú.

## Renombrar y reordenar

- El nombre de la carpeta es libre: "Docs", "Guía", "Manual"... lo que vale es la propiedad **blup**.
- El **title** de la _index es el nombre en el menú y arriba de la sección.
- El **weight** de la _index ordena las secciones en el menú automático y en las tarjetas de la página de inicio.
