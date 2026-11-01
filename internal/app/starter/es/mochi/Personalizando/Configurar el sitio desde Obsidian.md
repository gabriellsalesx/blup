---
title: Configurar el sitio desde Obsidian
date: {{date-6}}
description: Nombre, descripción, menú, botón, redes sociales, logo y favicon van en la nota _index.
tags: [Personalización]
image: "[[blup-configurar.jpg]]"
---
Los ajustes del sitio son **propiedades de la nota _index** de la carpeta del sitio. Cambia en Obsidian, cambia en el sitio. Si prefieres la terminal, `blup customize` hace las mismas preguntas y guarda en las mismas propiedades.

## Nombre y descripción

```yaml
site_name: Mi Blog
site_description: Apuntes sobre escritura, viajes y libros.
```

El nombre aparece en la pestaña del navegador, en el pie de página (© año y nombre) y en Google. La descripción aparece en Google y cuando alguien comparte el sitio.

## Menú superior

Un elemento por línea:

```yaml
menu:
  - Inicio
  - Escribiendo
  - Lecturas | Libros
  - Tienda | https://mitienda.com
```

- Un nombre solo busca una **carpeta o nota** con ese nombre: `Escribiendo` lleva a la sección Escribiendo.
- `Nombre que aparece | carpeta o nota` cambia el texto del elemento: `Lecturas | Libros` muestra "Lecturas" y lleva a la carpeta Libros.
- `Nombre | https://...` se vuelve un enlace a otro sitio.
- `Inicio` es la página de inicio.
- Sin la propiedad, el menú es automático: las carpetas de la raíz del sitio.

## Botón arriba

```yaml
button_text: Suscribirse
button_link: https://minewsletter.com
```

## Redes sociales

```yaml
social:
  - https://instagram.com/tunombre
  - https://github.com/tunombre
  - mailto:tu@ejemplo.com
```

Los íconos aparecen en el pie de página. Blup reconoce la red por la dirección: Instagram, X, LinkedIn, YouTube, GitHub, Facebook, WhatsApp y correo.

## Logo y favicon

Pon las imágenes en la bóveda (por ejemplo en la carpeta **Adjuntos**) y apunta a ellas:

```yaml
logo: "[[logo.svg]]"
favicon: "[[favicon.png]]"
```

El favicon es el ícono de la pestaña del navegador. En SVG se ve nítido en cualquier tamaño; en PNG, Blup genera los tamaños que piden los navegadores y el iPhone.

> [!tip] ¿Quieres verlo en la práctica?
> Abre la nota **_index** de la carpeta de este sitio: el menú de este blog está ahí.
