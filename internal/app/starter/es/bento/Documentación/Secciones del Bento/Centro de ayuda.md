---
title: Centro de ayuda
description: Colecciones de artículos con íconos y búsqueda, como los centros de soporte.
weight: 4
---
El centro de ayuda es una carpeta cuya **_index** tiene `blup: help`. La página de entrada tiene una búsqueda y las **colecciones** en tarjetas.

![Blup con un salvavidas](../../Adjuntos/blup-ajuda.jpg)

## Colecciones y artículos

- Cada **subcarpeta** es una colección.
- Cada **nota** dentro de ella es un artículo.
- La **_index** de la colección le da título, descripción e ícono.

```yaml
---
title: Primeros pasos
description: Lo básico para empezar.
icon: rocket
weight: 1
---
```

## Íconos

**En Blup Studio**: abre la nota **_index** de la colección. En las propiedades aparece **Ícono de la colección**: haz clic para abrir la cuadrícula de íconos y busca por el nombre en inglés o por palabras comunes en español (cohete, libro, ayuda...). **Automático** quita el ícono y deja que el tema los alterne.

**En Obsidian**: la propiedad **icon** acepta cualquier nombre de la biblioteca [Lucide](https://lucide.dev/icons), con más de 2 mil íconos: `rocket`, `pen-line`, `globe`, `life-buoy`, `credit-card`, `shield-check`, `users`... Copia el nombre del sitio de Lucide. Sin icon, el tema elige uno.

## Título y menú

En la _index del centro, **title** es la pregunta grande de la entrada ("¿Cómo podemos ayudarte?") y **linkTitle** es el nombre corto en el menú ("Centro de ayuda").
