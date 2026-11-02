---
title: Escribir en el editor
description: Comandos con /, enlaces con [[, menciones con @, imágenes pegadas y corrector.
weight: 2
---
El editor del Studio escribe Markdown, pero muestra el texto ya con formato: la negrita aparece en negrita, los títulos aparecen grandes, los callouts muestran el ícono. Las marcas (como `**` o `#`) vuelven solo mientras el cursor está sobre ellas, para que puedas editar.

![Blup escribiendo](../../Adjuntos/blup-escrever.jpg)

## La barra de formato

Selecciona un texto y usa la barra: negrita, cursiva, tachado, resaltado, código, títulos, listas, tareas, cita, callout, enlace, imagen, tabla, separador y comentario. El botón de alineación centra, alinea a la derecha o justifica.

## Comandos con /

Al principio de la línea (o después de un espacio), escribe `/` y elige con las flechas y Enter (o Tab):

- Título 1, Título 2 y Título 3
- Lista, Lista numerada y Lista de tareas
- Cita y Callout
- Tabla, Bloque de código y Separador
- Enlace a otra nota (abre la búsqueda de notas) y Enlace externo
- Imagen de la computadora
- Fecha de hoy

En una bóveda de notas, el `/` también tiene **Nuevo dibujo** y **Nuevo lienzo**: el Studio crea el archivo y pone el enlace en la nota.

## Enlaces con [[

Escribe `[[` y empieza a escribir el nombre de una nota. Elígela en la lista y el Studio guarda un enlace Markdown relativo, el formato que entiende el sitio.

En el editor, el enlace aparece solo por el texto, en azul y subrayado. Un clic abre la nota. Para enlaces web, usa `Ctrl` y haz clic: el clic simple sigue editando.

El botón de enlace (o `Ctrl+K`) inserta un enlace externo y ya selecciona la dirección para que la pegues.

## Menciones con @

Al principio de la línea o después de un espacio, escribe `@`:

- **Notas**: elige una nota y el Studio pone un enlace a ella, con la `@` en el texto.
- **Fechas**: hoy, mañana, ayer y los próximos días de la semana por su nombre. Se convierte en la fecha escrita, como 01/11/2026.

En una bóveda, la nota mencionada muestra quién la citó en el panel **Mencionada en**.

## Imágenes

Pega (`Ctrl+V`) o arrastra una imagen al editor. El Studio guarda el archivo en **Adjuntos** y pone la imagen en el texto. Aparece en su lugar, dentro del editor.

Las imágenes de internet esperan un clic en "Mostrar imagen de…", porque cargar la imagen le avisa a su servidor que se abrió la nota.

## Tablas

Créalas con el `/` o con la barra. Haz clic derecho en la tabla para insertar o eliminar filas y columnas, o para **Editar como texto (Markdown)**.

## Títulos que se pliegan

Pasa el mouse a la izquierda de un título: la flechita oculta todo hasta el siguiente título del mismo nivel. Atajos: `Ctrl+Shift+[` pliega, `Ctrl+Shift+]` despliega, `Ctrl+Alt+[` pliega todo y `Ctrl+Alt+]` despliega todo.

## Corrector ortográfico

Las palabras con error se subrayan. Haz clic derecho para ver las sugerencias o para agregar la palabra al diccionario. El corrector funciona en tu computadora: el texto no va a ningún servicio.

En **Configuración > Editor** activas o desactivas el corrector, eliges el idioma del diccionario (portugués, inglés o español) y ves las palabras agregadas.

## Pegar sin formato

El clic derecho tiene **Pegar como texto sin formato**, útil cuando el texto viene de un sitio web con estilos.

## Atajos del editor

| Acción | Teclas |
| --- | --- |
| Guardar | `Ctrl+S` |
| Deshacer y rehacer | `Ctrl+Z` y `Ctrl+Shift+Z` |
| Negrita y cursiva | `Ctrl+B` y `Ctrl+I` |
| Enlace | `Ctrl+K` |
| Aceptar la sugerencia | Tab |

> [!tip] Comentarios que no van al sitio
> El texto entre `%%` y `%%` se queda solo en las notas. Úsalo para recordatorios y borradores. Consulta [Fórmulas, código y comentarios](../Escribiendo/Fórmulas,%20código%20y%20comentarios.md).
