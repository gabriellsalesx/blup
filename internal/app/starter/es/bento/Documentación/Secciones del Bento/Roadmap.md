---
title: Roadmap
description: Un tablero con tres columnas, una nota por elemento.
weight: 2
---
El roadmap es una carpeta cuya **_index** tiene `blup: roadmap`. Cada nota dentro de ella es una tarjeta.

## La columna

La propiedad **status** elige la columna. Funciona en español, portugués o inglés:

| Columna | Escribe |
| --- | --- |
| Planeado | `planeado`, `planificado` o `planned` (también es el valor por defecto, sin status) |
| En curso | `en progreso`, `en curso`, `haciendo`, `in progress` |
| Completado | `completado`, `hecho`, `lanzado`, `done`, `shipped` |

## La tarjeta

```yaml
---
title: Más temas
description: Nuevos diseños para elegir al crear el sitio.
status: planeado
category: Temas
weight: 1
---
```

- **title** y **description** aparecen en la tarjeta.
- **category** es la etiqueta al pie de la tarjeta. Sin ella, vale la primera etiqueta.
- **weight** ordena las tarjetas dentro de la columna.
- El texto de la nota se vuelve la página del elemento, que se abre al hacer clic en la tarjeta.

Para cambiar un elemento de columna, cambia el **status** y guarda.

## Orden por votos

Con [comentarios y reacciones](Comentarios%20y%20reacciones.md) activados, las columnas **Planeado** y **En curso** ponen arriba los elementos más votados, y la tarjeta más votada del tablero recibe el sello **Más votados**. El **weight** sigue desempatando, y **Completado** mantiene su orden.

Para quedarte solo con el orden del **weight**, pon `sort_by_votes: false` en la _index del roadmap.
