---
title: Formato y callouts
description: Texto, listas, tablas y las cajas destacadas de Obsidian.
weight: 3
---
Todo aquí es Markdown común, igual en el Studio y en Obsidian. En el Studio, la barra de formato y el menú `/` insertan cada cosa sin que tengas que memorizar la sintaxis. Mira [Escribir en el editor](../Blup%20Studio/Escribir%20en%20el%20editor.md).

## Texto

**Negrita**, *cursiva*, ~~tachado~~, ==resaltado== y `código`. También notas al pie.[^1]

[^1]: Van al final de la página, con un enlace de vuelta.

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

Los títulos de nivel 2 (`##`) y 3 (`###`) aparecen en **En esta página**, junto al texto, y reciben su propio enlace.

## Callouts

> [!note] Nota
> Escribe `> [!note] Título` y el texto en las líneas siguientes, empezando con `>`.

> [!info]
> Sin título, el callout muestra el nombre del tipo.

> [!tip] Consejo
> Tipos: note, abstract, info, todo, tip, success, question, warning, failure, danger, bug, example y quote. Los alias de Obsidian (hint, important, faq, caution, error...) también valen.

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
