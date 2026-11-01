---
title: Formato que funciona
date: {{date-4}}
description: Todo lo que escribes en Obsidian y aparece en el sitio, con ejemplos.
tags: [Obsidian]
image: "[[blup-formatacao.jpg]]"
---
Este post es un muestrario. Abre la nota en Obsidian y mira cómo se escribió cada bloque.

## Texto

**Negrita**, *cursiva*, ~~tachado~~, ==resaltado==, `código` y [enlace externo](https://obsidian.md). Una nota al pie también funciona.[^1]

[^1]: Las notas al pie van al final del post, con un enlace de regreso.

## Listas

- Una lista
- con elementos
  - y subelementos

1. Lista numerada
2. Segundo paso

- [x] Tarea hecha
- [ ] Tarea pendiente

## Tabla

| Tema | Para quién |
| --- | --- |
| Mochi | Blog personal |
| Bento | Sitio de producto |

## Cita

> Escribir es pensar dos veces.

## Callouts

Los callouts de Obsidian se vuelven cajas de colores, con el mismo ícono:

> [!note] Nota
> El tipo va entre corchetes: `> [!note]`. El título es opcional.

> [!tip] Consejo
> También existen info, todo, success, question, warning, failure, danger, bug, example, quote y abstract.

> [!warning] Atención
> Sin título, el callout muestra el nombre del tipo.

> [!danger]
> Este no tiene título.

> [!example]- Haz clic para abrir
> Un `-` después del tipo deja el callout cerrado. Un `+` lo deja abierto, pero plegable.

## Código

```go
package main

import "fmt"

func main() {
	fmt.Println("¡Hola, Blup!")
}
```

El resaltado de colores sigue el lenguaje escrito después de las tres comillas invertidas.

## Fórmulas

Fórmulas en medio del texto, como $E = mc^2$, y en bloque:

$$
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

## Comentarios

El texto entre `%%` y `%%` es un comentario de Obsidian: aparece mientras editas y desaparece en el sitio. Sirve para recordatorios y borradores dentro del post. %%Este fragmento, por ejemplo, solo existe en Obsidian.%%

## Línea divisoria

---

Tres guiones solos en una línea dibujan la línea de arriba.
