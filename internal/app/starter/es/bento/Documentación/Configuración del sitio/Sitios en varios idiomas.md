---
title: Sitios en varios idiomas
description: El mismo sitio en portugués, inglés y español, con selector de idioma arriba.
weight: 5
---
Un sitio de Blup se escribe en un **idioma principal** (portugués, inglés o español) y también se puede publicar en los otros dos. Cada nota recibe una traducción, y el sitio muestra un **selector de idioma** arriba.

> [!info] Solo en Blup Studio
> La traducción se hace en el Studio, con IA en la nube o en tu computadora. Mira quién traduce y cuánto cuesta en [Traducción](../Funciones%20con%20IA/Traducción.md).

## Traducir una nota

1. Abre la nota en el Studio.
2. En el bloque **Idiomas**, en las propiedades, haz clic en **Traducir** junto al idioma.
3. El Studio muestra el avance ("Traduciendo, parte 1 de 3…") y te avisa cuando la traducción está lista.
4. **Abrir** te lleva a la traducción, para que la revises.

La IA traduce el título, la descripción y el texto. El código, los enlaces, las imágenes y el tipo de los callouts no cambian.

## Traducir todo el sitio

Haz clic en **Idiomas** en la barra lateral. La pantalla muestra, para cada idioma, cuántas notas están **traducidas**, **desactualizadas** y **sin traducción**.

- **Traducir lo que falta** traduce de una vez todas las notas sin traducción.
- **Actualizar las traducciones** rehace las que quedaron atrás.

## Dónde queda cada traducción

La traducción queda en una carpeta con el código del idioma (`en`, `es` o `pt`), **junto a la nota original**, con el nombre traducido:

```text
Blog/
├── Cómo publicar.md        original, en español
├── en/
│   └── How to publish.md   traducción al inglés
└── pt/
    └── Como publicar.md    traducción al portugués
```

La original y las traducciones quedan unidas por la propiedad `translationKey`, que el Studio pone solo. En el sitio, la página traducida es la misma página en el otro idioma: el selector lleva de una a la otra. Un enlace a una nota que todavía no tiene traducción lleva a la original.

## Lo que también se traduce

- En la nota de inicio: la descripción del sitio, el texto del botón y los nombres del menú.
- En cualquier nota: el título corto (`linkTitle`), la categoría y las etiquetas.
- La nota _index de cada sección da el nombre traducido de la sección.
- Los perfiles de los autores: el cargo y la bio.
- En el tablero de Feedback: los textos de las solicitudes de ejemplo.

## Traducción desactualizada

Cuando cambias la original, la traducción aparece como **Desactualizada: la original cambió**. Haz clic en **Traducir de nuevo**.

## La etiqueta "Traducido con IA"

Una traducción hecha por la IA muestra, junto al título, una etiqueta **Traducido con IA**, que abre un globo con el idioma original y el enlace a él. ¿La revisaste? Haz clic en **Marcar como revisada** y la etiqueta desaparece.

## Comentarios y votos

Todas las versiones de una página comparten la misma conversación: los votos y las reacciones suman a los lectores de todos los idiomas. Un comentario muestra en qué idioma se escribió, como "comentado en English".

> [!tip] Traducir a mano
> Puedes traducir a mano: crea la carpeta `en` junto a la nota y escribe allí la traducción, con el mismo nombre de archivo que la original (o la misma `translationKey`). Los idiomas publicados quedan en `translations` en el blup.toml, que el Studio completa al traducir.
