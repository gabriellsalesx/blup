---
title: Fórmulas, código y comentarios
description: Matemáticas, bloques de código con colores y notas que solo tú ves.
weight: 4
---
## Fórmulas

Entre `$` en medio del texto, como $a^2 + b^2 = c^2$, o entre `$$` en bloque:

$$
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

Las fórmulas se vuelven MathML: el navegador las dibuja, sin descargar nada.

## Código

Tres comillas invertidas, el nombre del lenguaje y el código:

```js
function hola(nombre) {
  return `¡Hola, ${nombre}!`;
}
```

Los colores siguen el lenguaje: `go`, `js`, `py`, `sh`, `yaml`, `html`, `css` y muchos otros.

## Comentarios

El texto entre `%%` y `%%` es un comentario de Obsidian: aparece mientras editas y no va al sitio. %%Como este fragmento, que solo existe en la nota.%%

Úsalos para recordatorios, borradores e instrucciones para quien edita, como en la nota **_index** de la carpeta del sitio.
