---
title: Ajustes del sitio
description: Nombre, descripción, autor, logo y favicon, en la pantalla Sitio o en la nota _index.
weight: 1
---
Los ajustes del sitio se guardan como **propiedades de la nota _index** de la carpeta del sitio. Hay tres formas de cambiarlos, y todas guardan en el mismo lugar.

## Desde Blup Studio: la pantalla Sitio

Haz clic en **Sitio** en la barra lateral. La pantalla muestra cómo se presenta el sitio, en bloques:

1. **Nombre y descripción**
   - **Nombre del sitio**: aparece arriba, en la pestaña del navegador y en Google.
   - **Descripción**: una frase. Aparece arriba, en Google y al compartir.
   - **Autor**: quien firma los posts que no indican autor.
2. **Logo y favicon**
   - **Logo**: arriba en el sitio. Sin logo, aparece el nombre.
   - **Mostrar el nombre junto al logo**: desactivado, la parte superior del sitio muestra solo el logo.
   - **Favicon**: el ícono de la pestaña y de la app. Cuadrado, de preferencia de 512 píxeles o en SVG.
3. **Menú superior** y **Botón superior**. Mira [Menú, submenús y botón](Menú,%20submenús%20y%20botón.md).
4. **Redes sociales**. Mira [Redes sociales](Redes%20sociales.md).

Cada cambio aparece al instante en la vista previa de al lado, sin recargar.

![Blup configurando](../../Adjuntos/blup-configurar.jpg)

## En el archivo: la nota _index

| Propiedad | Qué hace |
| --- | --- |
| `site_name` | Nombre del sitio: pestaña del navegador, pie de página y Google. |
| `site_description` | Una frase sobre el sitio, para Google y las redes. |
| `logo` | Imagen del logo arriba, como `"[[logo.svg]]"`. |
| `header_name` | `false` muestra solo el logo arriba, sin el nombre. |
| `favicon` | Ícono de la pestaña del navegador, como `"[[favicon.png]]"`. |
| `menu` | Elementos del menú superior. |
| `button_text` y `button_link` | El botón de la esquina superior derecha. |
| `social` | Enlaces a tus perfiles, con ícono en el pie de página. |

```yaml
---
title: Acme
description: La forma simple de hacer X.
site_name: Acme
site_description: La forma simple de hacer X.
logo: "[[acme-logo.svg]]"
favicon: "[[acme-icono.png]]"
social:
  - https://github.com/acme
  - mailto:hola@acme.com
---
```

## Desde la línea de comandos

`blup customize` (o `blup personalizar`) hace las mismas preguntas (nombre, descripción, autor, logo, favicon, menú, redes) y guarda en las mismas propiedades.

## Logo y favicon

- En **SVG**, se ven nítidos en cualquier pantalla.
- En **PNG**, Blup genera los tamaños que piden los navegadores y los celulares.
- Sin logo, arriba se muestra el nombre. Sin favicon, el sitio usa la mascota.

## Las otras pantallas del sitio en el Studio

| Pantalla | Qué configura |
| --- | --- |
| Secciones | Las partes del Bento. Mira [Activar y desactivar secciones](../Secciones%20del%20Bento/Activar%20y%20desactivar%20secciones.md) |
| Autores | Los perfiles. Mira [Autores](Autores.md) |
| Idiomas | El sitio en otros idiomas. Mira [Sitios en varios idiomas](Sitios%20en%20varios%20idiomas.md) |
| Comentarios | Votos, reacciones y comentarios. Mira [Comentarios y reacciones](../Secciones%20del%20Bento/Comentarios%20y%20reacciones.md) |
| Cookies y estadísticas | Herramientas y aviso. Mira [Cookies y estadísticas](Cookies%20y%20estadísticas.md) |
| Podcast | Solo en los blogs. Mira [Podcast de los blogs](../Funciones%20con%20IA/Podcast%20de%20los%20blogs.md) |
| Publicar | Dónde y cómo se publica el sitio. Mira [Publicar desde el Studio](../Publicación/Publicar%20desde%20el%20Studio.md) |

> [!note] ¿Y el blup.toml?
> En la carpeta del sitio en la computadora, el **blup.toml** guarda lo que no es contenido: dónde publicar, los idiomas, la contraseña del sitio y el repositorio de la sincronización. El Studio y `blup customize` cambian todo eso sin que abras el archivo.
