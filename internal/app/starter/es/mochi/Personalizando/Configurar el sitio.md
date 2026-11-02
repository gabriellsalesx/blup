---
title: Configurar el sitio
date: {{date-6}}
description: Nombre, descripción, logo, menú con submenús, botón y redes sociales, en la pantalla Sitio del Studio o en la nota _index.
tags: [Personalización, Studio]
image: "[[blup-configurar.jpg]]"
---
La configuración del sitio está en la pantalla **Sitio**, en el menú lateral del Studio. Lo que cambias ahí cambia en el sitio: la vista previa lo muestra al instante.

Todo esto se guarda en las propiedades de la nota **_index** de la carpeta del sitio. Al final de cada parte está cómo queda escrito en esa nota, por si prefieres editar el archivo directamente.

## Nombre y descripción

El **Nombre del sitio** aparece arriba, en la pestaña del navegador, en el pie de página (© año y nombre) y en Google. La **Descripción** es una frase sobre el sitio: aparece en Google y cuando alguien comparte el sitio.

En **Autor**, elige quién firma los posts que no dicen el autor.

```yaml
site_name: Mi Blog
site_description: Notas sobre escritura, viajes y libros.
```

## Logo y favicon

En **Logo y favicon**, haz clic en **Elegir**. La imagen va a la carpeta **Adjuntos** sola.

- El **Logo** aparece arriba en el sitio. Sin logo, aparece el nombre.
- **Mostrar el nombre junto al logo**: desactivado, la parte de arriba del sitio muestra solo el logo. Sirve cuando el logo ya trae el nombre.
- El **Favicon** es el ícono de la pestaña del navegador y de la app. En SVG, se ve nítido en cualquier tamaño; en PNG, Blup crea los tamaños que piden los navegadores y el iPhone. Cuadrado, de preferencia de 512 píxeles.

```yaml
logo: "[[logo.svg]]"
favicon: "[[favicon.png]]"
header_name: false
```

## Menú superior

Las páginas del menú, en orden:

- **Agregar página** pone una carpeta o nota del sitio. Una carpeta, como **Escribiendo**, lleva a toda la sección.
- **Agregar enlace** pone un enlace a otro sitio, como tu tienda.
- El nombre de cada elemento es libre: escribe el texto que debe aparecer.
- Arrastra por el asa para cambiar el orden.

**Submenús**: arrastra un elemento encima de otro. El de arriba pasa a abrir una lista con los de abajo, sin enlace propio. **Nuevo submenú** crea una entrada vacía para que arrastres elementos dentro. Es un solo nivel.

Sin ningún elemento, el menú es automático: las carpetas de la raíz del sitio.

En la nota **_index**, un elemento por línea. `Nombre | destino` cambia el texto del elemento, una dirección se vuelve un enlace externo, y `+ ` al principio pone el elemento en el submenú de la línea de arriba:

```yaml
menu:
  - Inicio
  - Escribiendo
  - Lecturas |
  - + Libros | Libros
  - + Revistas | Revistas
  - Tienda | https://mitienda.com
```

## Botón superior

Un botón destacado a la derecha del menú, como "Suscribirse". Completa el **Texto del botón** y **A dónde lleva**. Deja el texto vacío para no tener botón.

```yaml
button_text: Suscribirse
button_link: https://minewsletter.com
```

## Redes sociales

En **Redes sociales**, haz clic en la red (Instagram, X, LinkedIn, YouTube, Discord, GitHub, Facebook, WhatsApp o e-mail): el Studio ya pone el principio de la dirección, y tú la completas con tu perfil. Los íconos aparecen en el pie de página.

```yaml
social:
  - https://instagram.com/tunombre
  - https://github.com/tunombre
  - mailto:tu@ejemplo.com
```

> [!tip] ¿Quieres verlo en la práctica?
> Abre la nota **_index** de la carpeta del sitio: el menú de este blog está ahí. Con la línea de comandos, `blup customize` hace las mismas preguntas y las guarda en las mismas propiedades.
