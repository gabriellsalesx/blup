---
title: Estructura de carpetas
description: Cómo las carpetas y las notas se vuelven las páginas del sitio.
weight: 5
---
## El mapa

```text
Sitio/                    la carpeta de las notas del sitio
├── _index.md             página de inicio y configuración del sitio
├── Documentación/        sección de documentación (blup: docs)
│   ├── _index.md         apertura de la documentación
│   └── Primeros pasos/   grupo en el menú lateral
├── Roadmap/              tablero del roadmap (blup: roadmap)
├── Changelog/            lista de versiones (blup: changelog)
├── Ayuda/                centro de ayuda (blup: help)
├── Blog/                 blog (blup: blog)
├── Feedback/             tablero de solicitudes (blup: feedback)
├── Autores/              perfiles de los autores (blup: authors)
└── Adjuntos/             imágenes, audios y otros archivos
```

## Las reglas

- Cada **nota** se vuelve una página.
- Cada **carpeta** se vuelve un grupo o una sección.
- La nota **_index** de una carpeta le da título, descripción y texto de apertura.
- En Bento, la propiedad **blup** en la _index de una carpeta de la raíz dice qué tipo de sección es: `docs`, `roadmap`, `changelog`, `help`, `blog`, `feedback` o `authors`. El nombre de la carpeta es libre.
- Una carpeta llamada `en`, `es` o `pt` junto a una nota guarda su **traducción**. Mira [Sitios en varios idiomas](../Configuración%20del%20sitio/Sitios%20en%20varios%20idiomas.md).

## Lo que queda fuera del sitio

- Notas con `draft: true` (aparecen solo en la vista previa).
- Carpetas que empiezan con punto, como `.obsidian` y `.blup`.
- La carpeta de plantillas de Obsidian (Plantillas).
- La carpeta de autores: se vuelve solo los datos de los perfiles.
- Las guías para agentes de IA (`AGENTS.md`, `CLAUDE.md` y `GEMINI.md`), que el Studio crea al abrir la terminal.
- Adjuntos que ninguna nota usa.

## Carpetas que Blup usa por dentro

- **.blup**, dentro de la carpeta de las notas: guarda el archivo `site.toml`, la parte de la configuración que viaja con la sincronización.
- **La carpeta del sitio** (fuera de las notas): el `blup.toml`, los archivos del generador, la papelera del Studio y el estado de la sincronización. No necesitas abrirla.

> [!note] Direcciones de las páginas
> La dirección viene del nombre de la carpeta y de la nota, sin acentos y con guiones: la nota "Estructura de carpetas" en "Primeros pasos" queda como `/documentacion/primeros-pasos/estructura-de-carpetas/`.
