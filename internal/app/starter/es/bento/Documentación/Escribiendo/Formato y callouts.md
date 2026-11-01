---
title: Formato y callouts
description: Texto, listas, tablas y los cuadros destacados de Obsidian.
weight: 3
---
## Texto

**Negrita**, *cursiva*, ~~tachado~~, ==resaltado== y `código`. Notas al pie también.[^1]

[^1]: Van al final de la página, con un enlace de regreso.

## Listas

- Elementos con guion
  - y subelementos

1. Pasos numerados
2. En orden

- [x] Tarea hecha
- [ ] Tarea pendiente

## Tablas

| Sección | Carpeta | Propiedad |
| --- | --- | --- |
| Documentación | Documentación | `blup: docs` |
| Roadmap | Roadmap | `blup: roadmap` |

## Títulos

Los títulos de nivel 2 (`##`) y 3 (`###`) aparecen en **En esta página**, al lado del texto, y reciben un enlace propio.

## Callouts

> [!note] Nota
> Escribe `> [!note] Título` y el texto en las líneas siguientes, empezando con `>`.

> [!info]
> Sin título, el callout muestra el nombre del tipo.

> [!tip] Consejo
> Tipos: note, abstract, info, todo, tip, success, question, warning, failure, danger, bug, example y quote. Los alias de Obsidian (hint, important, faq, caution, error...) también funcionan.

> [!success] Éxito
> Cada tipo tiene color e ícono.

> [!question] Pregunta
> Útil en preguntas frecuentes.

> [!warning] Atención
> Para avisos.

> [!danger] Peligro
> Para lo que no tiene vuelta atrás.

> [!faq]- Callout cerrado
> Un `-` después del tipo deja el callout cerrado; haz clic para abrirlo. Un `+` lo deja abierto, pero plegable.
