---
title: Comandos
description: Un menú para todo, y un comando para cada acción.
weight: 1
---
Ejecutar solo `blup` abre el **menú**: vista previa, publicar, personalizar, sincronizar, cambiar de sitio, crear otro e idioma. Es la forma más fácil.

![Blup en la terminal](../../Adjuntos/blup-terminal.jpg)

## Cada acción tiene un comando

| Comando | Alias en portugués | Qué hace |
| --- | --- | --- |
| `blup new` | `blup novo` | Crea un sitio nuevo |
| `blup preview` | `blup ver` | Abre la vista previa en el navegador |
| `blup publish` | `blup publicar` | Pone el sitio en línea |
| `blup customize` | `blup personalizar` | Menú, secciones, redes, logo, favicon, autor, dominio propio, contraseña general y comentarios |
| `blup sync` | `blup sincronizar` | Guarda las notas en un repositorio privado de GitHub |
| `blup convert` | `blup converter` | Cambia los `[[wikilinks]]` por enlaces estándar |
| `blup build` | `blup gerar` | Genera el sitio en `public/`, sin preguntas |
| `blup language` | `blup idioma` | Cambia el idioma de Blup |
| `blup update` | `blup atualizar` | Actualiza Blup |
| `blup version` | `blup version` | Muestra la versión |

## Más de un sitio

Blup recuerda todos los sitios que creaste. En el menú, **Cambiar a otro sitio** elige otro. En los comandos, pasa la carpeta del sitio: `blup publish ~/Sites/mi-sitio`.

## Sin preguntas

`blup build` no pregunta nada: ideal para scripts y automatizaciones. El sitio sale en la carpeta `public` dentro de la carpeta del sitio.

## Un sitio dentro de una subcarpeta

¿Quieres la documentación en `misitio.pages.dev/bento/`, junto a una landing page en la raíz? Genera el sitio para esa dirección y cópialo a la carpeta de la landing:

```sh
blup build --url https://misitio.pages.dev/bento/ --out ../mi-landing/bento
```

- `--url` es la dirección final del sitio, y puede ser una subcarpeta. Los enlaces, las imágenes y la búsqueda funcionan dentro de ella.
- `--out` copia el sitio listo a otra carpeta, reemplazando lo que Blup había puesto ahí antes. Por seguridad, Blup nunca borra una carpeta que no llenó él.
- Después, publica la carpeta de la landing, por ejemplo con Cloudflare Pages conectado a un repositorio de GitHub: cada `git push` actualiza el sitio.

> [!tip] En portugués también
> `blup gerar --url … --saida …` hace lo mismo.
