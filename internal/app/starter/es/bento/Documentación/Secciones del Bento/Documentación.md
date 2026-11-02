---
title: Documentación
description: Páginas organizadas en grupos, con menú lateral, índice y navegación.
weight: 1
---
La documentación es la sección que estás leyendo. Está en una carpeta de la raíz cuya nota **_index** tiene `blup: docs`.

## Cómo se organiza

- Cada **nota** es una página.
- Cada **subcarpeta** es un grupo en el menú lateral, con el título de su _index.
- La **_index** de la carpeta de la documentación es la página de apertura.
- El orden sigue la propiedad **weight** (menor primero) y, sin ella, el nombre de la nota.

```text
Documentación/
├── _index.md           blup: docs
├── Primeros pasos/
│   ├── _index.md       title: Primeros pasos, weight: 1
│   ├── Instalación.md  weight: 1
│   └── Crear el primer sitio.md   weight: 2
└── Escribiendo/
    └── _index.md       weight: 2
```

## Lo que recibe cada página

- **Menú lateral** con todos los grupos; el grupo actual queda abierto.
- **En esta página**: los títulos `##` y `###`, al lado del texto.
- **Ruta** (Inicio > Documentación > grupo) arriba.
- **Anterior y Siguiente** al final, siguiendo el orden del menú.
- **Búsqueda** en todas las páginas del sitio.

## Crear una página

- **En Blup Studio**: haz clic en una subcarpeta de la Documentación, luego en **Nueva nota** (o `Ctrl+N`). Completa **Título**, **Descripción** y **Orden** en las propiedades.
- **En Obsidian**: crea la nota en la subcarpeta y escribe `title`, `description` y `weight` arriba.

Para un grupo nuevo, crea una subcarpeta con una nota **_index** (título, descripción y `weight`).

> [!tip] Una descripción corta ayuda
> La propiedad **description** aparece debajo del título y en Google. Basta con una frase.
