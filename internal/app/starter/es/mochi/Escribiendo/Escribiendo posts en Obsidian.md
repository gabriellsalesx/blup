---
title: Escribiendo posts en Obsidian
date: {{date-3}}
description: Las propiedades que entiende el sitio, enlaces entre notas, imágenes y plantillas.
tags: [Obsidian]
image: "[[blup-escrever.jpg]]"
---
Para crear un post, crea una nota en la carpeta del sitio. Eso es todo. Lo demás es opcional y va en las **propiedades**, el bloque al inicio de la nota (en Obsidian, escribe `---` en la primera línea o usa el comando "Agregar propiedad").

## Las propiedades

| Propiedad | Para qué sirve |
| --- | --- |
| `title` | Título del post. Sin ella, se usa el nombre de la nota. |
| `date` | Fecha del post. Sin ella, se usa `created` o la fecha del archivo. |
| `description` | Resumen en las tarjetas, en Google y en las redes sociales. |
| `tags` | Temas. Cada etiqueta tiene su propia página. |
| `image` | Portada del post, como `"[[mi-portada.jpg]]"`. |
| `authors` | Quién lo escribió, como `[Ana, Bruno]`. Sin ella, se usa el autor del sitio. |
| `draft` | `true` deja la nota fuera del sitio (solo aparece en la vista previa). |
| `hidden` | `true` la publica, pero la quita de las listas y de Google. |
| `password` | Protege el post con contraseña. Mira [Posts con contraseña](Posts%20con%20contraseña.md). |
| `updated` | Fecha de la última actualización, cuando quieras mostrarla. |

Abre este post en Obsidian para ver sus propiedades al inicio.

## Enlaces entre notas

Arrastra una nota dentro de otra, o escribe `[[` y elige la nota. Con el ajuste que Blup hizo en Obsidian, el enlace se guarda así:

```md
[Formato que funciona](Formato%20que%20funciona.md)
```

En el sitio, se convierte en el enlace de la página. Ejemplo: [Formato que funciona](Formato%20que%20funciona.md).

> [!tip] ¿Ya tienes notas con [[wikilinks]]?
> Ejecuta `blup convert`: Blup cambia los `[[enlaces]]` antiguos por el formato estándar en todas las notas de la carpeta del sitio, y antes te muestra lo que va a cambiar.

## Imágenes

Arrastra la imagen a la nota, como siempre. Blup genera versiones más pequeñas y modernas (WebP) de cada imagen, y el navegador descarga solo el tamaño que necesita.

![Blup escribiendo en una tarjeta de nota](../Adjuntos/blup-escrever.jpg)

El texto entre corchetes es la descripción de la imagen: ayuda a quien usa lector de pantalla y a Google.

## Plantillas

Blup creó la carpeta **Templates** en tu bóveda con una plantilla de post y configuró el plugin Plantillas de Obsidian. Para empezar un post con las propiedades correctas, usa el comando **Plantillas: Insertar plantilla**. La carpeta de plantillas nunca se publica en el sitio.

## Organizar en secciones

Crea una subcarpeta en la carpeta del sitio, como **Viajes**, y pon posts en ella: el sitio gana la sección Viajes, con página propia y elemento en el menú. Para darle título y descripción a la sección, crea una nota **_index** dentro de ella, como la de esta sección.
