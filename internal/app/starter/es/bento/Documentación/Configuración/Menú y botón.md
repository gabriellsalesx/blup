---
title: Menú y botón
description: Elige los elementos del menú superior y el botón destacado.
weight: 2
---
## Menú automático

Sin la propiedad **menu**, el menú superior muestra las secciones del sitio, en el orden del **weight** de cada una.

## Menú elegido

En la nota **_index** de la carpeta del sitio, un elemento por línea:

```yaml
menu:
  - Documentación
  - Ayuda | Centro de ayuda
  - Novedades | Changelog
  - Precios | https://acme.com/precios
```

| Línea | Se vuelve |
| --- | --- |
| `Documentación` | Enlace a la carpeta (o nota) con ese nombre o título. |
| `Ayuda \| Centro de ayuda` | El texto "Ayuda", que lleva a la carpeta "Centro de ayuda". |
| `Precios \| https://...` | Enlace a otro sitio. |
| `Inicio` | La página de inicio. |
| `Blog \| /blog/` | Una dirección del propio sitio. |

El elemento de la página actual queda marcado.

## El botón

```yaml
button_text: Crear cuenta
button_link: https://app.acme.com/registro
```

Aparece en la esquina superior derecha, destacado. Sin las dos propiedades, no hay botón.
