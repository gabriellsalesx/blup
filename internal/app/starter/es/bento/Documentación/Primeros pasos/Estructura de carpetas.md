---
title: Estructura de carpetas
description: Cómo las carpetas y notas de la bóveda se vuelven las páginas del sitio.
weight: 3
---
## El mapa

```text
Sitio/                    la carpeta del sitio
├── _index.md             página de inicio y configuración del sitio
├── Documentación/        sección de documentación (blup: docs)
│   ├── _index.md         apertura de la documentación
│   └── Primeros pasos/   grupo en el menú lateral
├── Roadmap/              tablero del roadmap (blup: roadmap)
├── Changelog/            lista de versiones (blup: changelog)
├── Ayuda/                centro de ayuda (blup: help)
├── Blog/                 blog (blup: blog)
├── Autores/              perfiles de los autores (blup: authors)
└── Adjuntos/             imágenes
```

## Las reglas

- Cada **nota** se vuelve una página.
- Cada **carpeta** se vuelve un grupo o sección.
- La nota **_index** de una carpeta le da título, descripción y texto de apertura.
- En Bento, la propiedad **blup** en la _index de una carpeta de la raíz dice qué tipo de sección es: `docs`, `roadmap`, `changelog`, `help`, `blog` o `authors`. El nombre de la carpeta es libre.

## Lo que queda fuera del sitio

- Notas con `draft: true`.
- Carpetas que empiezan con punto, como `.obsidian`.
- La carpeta de plantillas (Templates).
- La carpeta de autores: solo se vuelve los datos de los perfiles.

> [!note] Direcciones de las páginas
> La dirección sale del nombre de la carpeta y de la nota, sin acentos y con guiones: la nota "Estructura de carpetas" en "Primeros pasos" se vuelve `/documentacion/primeros-pasos/estructura-de-carpetas/`.
