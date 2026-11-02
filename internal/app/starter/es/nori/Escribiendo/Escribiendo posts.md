---
title: Escribiendo posts
date: {{date-3}}
description: Crear un post, las propiedades, enlaces entre notas, imágenes y secciones.
tags: [Escritura, Studio]
image: "[[blup-escrever.jpg]]"
---
Para crear un post, crea una nota en la carpeta del sitio y escribe. Eso es todo. El resto es opcional.

En el Studio, haz clic en **Nueva nota** en el panel Archivos, ponle un nombre y presiona Enter. La vista previa de al lado muestra cómo va a quedar el post en el sitio, mientras escribes.

## Comandos con /

Al principio de una línea (o después de un espacio), escribe `/` y elige: título, lista, lista de tareas, cita, callout, tabla, bloque de código, línea divisoria, enlace a otra nota, enlace externo, imagen de la computadora o la fecha de hoy. Enter o Tab eligen.

Escribe `@` para mencionar otra nota o una fecha, como hoy, mañana o el próximo viernes.

La **barra de formato**, encima del texto, tiene los mismos comandos en botones. El clic derecho en el editor también tiene los menús **Formato** e **Insertar**.

## Las propiedades

La fecha, las etiquetas, la portada y lo demás están en el panel **Propiedades**, encima del editor. Haz clic en el nombre del panel para abrirlo o cerrarlo. El Studio guarda todo al principio de la propia nota.

| En el Studio | En la nota | Para qué sirve |
| --- | --- | --- |
| **Título** | `title` | Título del post. Si está vacío, se usa el nombre de la nota. |
| **Descripción** | `description` | Resumen en las listas, en Google y al compartir. |
| **Fecha** | `date` | Fecha del post. |
| **Etiquetas** | `tags` | Temas. Cada etiqueta tiene su página. |
| **Autores** | `authors` | Quién escribió. Si está vacío, se usa el autor del sitio. |
| **Portada** | `image` | La imagen de portada, como `"[[mi-portada.jpg]]"`. |
| **Borrador** | `draft: true` | Deja la nota fuera del sitio. Solo aparece en la vista previa. |
| **Oculto** | `hidden: true` | Se publica, pero sale de las listas y de Google. |
| **Contraseña** | `password` | Pide una contraseña para leer. Mira [Posts con contraseña](Posts%20con%20contraseña.md). |

En el archivo de la nota, las mismas propiedades están en el bloque del principio, entre dos líneas `---`. Puedes editarlas en cualquier editor de texto.

## Enlaces entre notas

En el Studio, escribe `[[` y elige la nota por el nombre (Tab elige), o usa **Enlace a otra nota** en el menú `/`. En el texto, el enlace aparece solo con el nombre, en azul. Guardado en la nota, queda así:

```md
[Formato que funciona](Formato%20que%20funciona.md)
```

En el sitio, se vuelve el enlace a la página. Ejemplo: [Formato que funciona](Formato%20que%20funciona.md).

Para un enlace a otro sitio, usa **Enlace** (Ctrl+K).

> [!tip] ¿Renombraste una nota?
> En el Studio, puedes renombrar sin miedo: actualiza solo los enlaces que apuntan a ella.

¿Ya tienes notas antiguas con wikilinks, como `[[Formato que funciona]]`? Ejecuta `blup convert`: Blup cambia los enlaces antiguos por el formato estándar y antes te muestra lo que va a cambiar.

## Imágenes

Pega la imagen directo en la nota, o arrástrala, o usa **Imagen de la computadora**. El Studio guarda el archivo en la carpeta **Adjuntos** solo, y la imagen aparece en su lugar, dentro del texto.

Blup crea versiones más pequeñas y modernas (WebP) de cada imagen, y el navegador descarga solo el tamaño que necesita.

![Blup escribiendo en una tarjeta de nota](../Adjuntos/blup-escrever.jpg)

El texto entre corchetes es la descripción de la imagen: ayuda a quien usa lector de pantalla y a Google.

La pantalla **Adjuntos**, en el menú lateral del Studio, muestra todas las imágenes y archivos que usan las notas. El filtro **Sin usar** encuentra lo que sobró.

## Organizar en secciones

Crea una carpeta en la carpeta del sitio, como **Viajes**, y pon posts en ella: el sitio gana la sección Viajes, con página propia y elemento en el menú. Para darle título y descripción a la sección, crea una nota **_index** dentro de ella, como la de esta sección.

## Corrector ortográfico

El Studio subraya las palabras con errores mientras escribes. Haz clic derecho en la palabra para ver las sugerencias o agregar la palabra al diccionario. El corrector funciona en tu computadora: el texto no sale de ella.
