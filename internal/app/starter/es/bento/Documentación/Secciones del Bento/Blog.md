---
title: Blog
description: Posts con portada, etiquetas, autores y posts relacionados.
weight: 5
---
El blog es una carpeta cuya **_index** tiene `blup: blog`. Cada nota es un post, con el mismo aspecto del tema Mochi.

```yaml
---
title: Cómo se armó este sitio
date: 2026-10-25
description: Por dentro de la plantilla que viene con Blup.
tags: [Detrás de escena]
image: "[[blup-bento.jpg]]"
authors: [Blup]
---
```

- El post más reciente se vuelve el **destacado** de arriba; para elegir otro, usa `featured: true`.
- **image** es la portada. Sin ella, vale la primera imagen del texto.
- **tags** se vuelven filtros arriba de la lista y páginas propias.
- **authors** muestra foto y nombre; cada autor tiene una página. Mira [Autores](../Configuración/Autores.md).
- Cada post tiene índice, barra de lectura, botones para compartir y posts relacionados por las etiquetas.
