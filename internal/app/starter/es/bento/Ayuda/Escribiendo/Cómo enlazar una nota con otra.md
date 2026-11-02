---
title: ¿Cómo enlazo una nota con otra?
description: Los enlaces entre notas se convierten en enlaces entre páginas.
weight: 3
---
## En Blup Studio

- Escribe `[[` y elige la nota de la lista.
- O escribe `@` al principio de la línea (o después de un espacio) y elige una nota o una fecha.

El Studio inserta un enlace en el formato que el sitio entiende. En el editor, el enlace se ve solo por el texto, en azul. Haz clic en él para abrir la otra nota.

## En el archivo de la nota

Usa enlaces Markdown relativos, como `[Instalación](../Primeros%20pasos/Instalación.md)`. Si las notas ya tienen `[[wikilinks]]`, ejecuta `blup convert` (o `blup converter`) para cambiarlos todos de una vez.
