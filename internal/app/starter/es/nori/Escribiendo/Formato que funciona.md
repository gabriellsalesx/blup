---
title: Formato que funciona
date: {{date-4}}
description: Todo lo que escribes en Studio y aparece en el sitio, con ejemplos.
tags: [Escritura]
image: "[[blup-formatacao.jpg]]"
---
Este post es un muestrario. Abre la nota en Studio y mira cómo se escribió cada bloque. Casi todo tiene un botón en la **barra de formato**, arriba del texto. El clic derecho en el editor también tiene los menús **Formato** e **Insertar**.

## Texto

**Negrita**, *cursiva*, ~~tachado~~, ==resaltado==, `código` y [enlace externo](https://github.com/gabriellsalesx/blup). Una nota al pie también funciona.[^1]

[^1]: Las notas al pie van al final del post, con un enlace de regreso.

Selecciona el texto y usa los botones **Negrita** (Ctrl+B), **Cursiva** (Ctrl+I), **Tachado**, **Resaltar**, **Código** y **Enlace** (Ctrl+K). La nota al pie no tiene botón: escribe `[^1]` en el texto y, en una línea aparte, `[^1]:` seguido de la nota.

## Listas

- Una lista
- con elementos
  - y subelementos

1. Lista numerada
2. Segundo paso

- [x] Tarea hecha
- [ ] Tarea pendiente

Los botones **Lista**, **Lista numerada** y **Lista de tareas** hacen cada una. Los títulos están en el botón **Título**.

## Tabla

| Tema | Para quién |
| --- | --- |
| Nori | Blog de una sola columna |
| Mochi | Blog con cuadrícula de posts |

El botón **Tabla** crea una tabla lista para llenar. Haz clic derecho en una celda para insertar o eliminar filas y columnas.

## Cita

> Escribir es pensar dos veces.

Usa el botón **Cita (línea destacada)**.

## Callouts

El botón **Callout** abre la lista de tipos. Cada uno se vuelve una caja de color, con ícono:

> [!note] Nota
> El tipo va entre corchetes: `> [!note]`. El título es opcional.

> [!tip] Consejo
> También existen info, todo, success, question, warning, failure, danger, bug, example, quote y abstract.

> [!warning] Atención
> Sin título, el callout muestra el nombre del tipo.

> [!danger]
> Este no tiene título.

> [!example]- Haz clic para abrir
> Marca **Empieza plegado** en el mismo menú y el callout ya nace cerrado. Por debajo, es un `-` después del tipo.

## Código

```go
package main

import "fmt"

func main() {
	fmt.Println("¡Hola, Blup!")
}
```

Usa **Bloque de código** en el menú Insertar del clic derecho. El resaltado de colores sigue el lenguaje escrito después de las tres comillas invertidas, como `go` aquí.

## Fórmulas

Fórmulas en medio del texto, como $E = mc^2$, y en bloque:

$$
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

Las fórmulas no tienen botón: escribe entre `$` (en medio del texto) o entre `$$` (en bloque).

## Comentarios

El texto entre `%%` y `%%` es un comentario: aparece mientras editas y desaparece en el sitio. Sirve para recordatorios y borradores dentro del post. Selecciona el texto y usa **Comentario (queda fuera del sitio)**, en el menú Formato del clic derecho. %%Este fragmento, por ejemplo, solo existe en el editor.%%

## Línea divisoria

---

El botón **Separador** dibuja la línea de arriba.
