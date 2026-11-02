---
title: Podcast de los blogs
description: En los temas Mochi y Nori, las narraciones de los posts se vuelven episodios con feed para Spotify y Apple Podcasts.
weight: 4
---
El **Podcast** es una sección de los blogs (temas **Mochi** y **Nori**). Reúne las narraciones de los posts como episodios en una página con su propio reproductor y publica un **feed** para Spotify, Apple Podcasts y otras apps.

> [!info] De los temas de blog, solo en el Studio
> Bento (este tema) no tiene la sección Podcast. En los blogs, se agrega con Blup Studio. Un blog que ya tiene la sección sigue generando la página y el feed también con la línea de comandos.

## 1. Agregar la sección

1. Abre el blog en el Studio.
2. Haz clic en **Secciones** y, en **Podcast**, haz clic en **Agregar**.
3. El Studio crea la carpeta **Podcast** y la pantalla **Podcast** aparece en la barra lateral, en el grupo Público.

## 2. Crear los episodios

Cada **post con narración** es un episodio. Narra los posts que quieras (mira [Narración de los posts](Narración%20de%20los%20posts.md)) y publica el sitio. La pantalla Podcast muestra cuántos episodios hay.

Quedan fuera: los borradores, los posts ocultos y los posts con contraseña.

## 3. Completar los datos del programa

En la pantalla **Podcast**:

| Campo | Si queda vacío |
| --- | --- |
| Nombre del podcast | El nombre del sitio |
| Descripción (Google, compartir y apps de podcast) | El texto de la nota de la sección |
| Autor | El autor del sitio |
| E-mail de contacto | Sin e-mail. Es con él que Spotify y Apple confirman que el podcast es tuyo, y queda público en el feed |
| Categoría | Tecnología (la lista es la de Apple Podcasts) |
| Contenido explícito | No |
| Portada del podcast | La portada del episodio más nuevo, y si no, el logo del sitio |

La portada ideal es un cuadrado de 1400 a 3000 píxeles, en JPG o PNG. El Studio avisa cuando la imagen no sirve.

## 4. Enviar a Spotify y a Apple

El feed necesita la dirección completa del sitio, así que **publica el sitio** primero, con Blup Cloud, Cloudflare Pages o GitHub Pages (mira [Dónde publicar](../Publicación/Dónde%20publicar.md)). Con dominio propio, el feed usa tu dominio. Entonces la pantalla muestra el **Feed del podcast** de cada idioma, con **Copiar**.

- **Enviar a Spotify**: en Spotify for Creators, elige "ya tengo un podcast" y pega la dirección. El código de confirmación llega al e-mail de contacto.
- **Enviar a Apple Podcasts**: en Apple Podcasts Connect, inicia sesión con tu Apple ID, crea un programa a partir del feed y espera la aprobación.

Hazlo una sola vez: los episodios nuevos llegan solos a las apps cuando publicas el sitio.

## La página del podcast en el sitio

- La lista de episodios, del más nuevo al más antiguo, con la portada de cada post.
- Un reproductor fijo que pasa solo al siguiente episodio, incluso en el celular con la pantalla apagada.
- Retroceder y avanzar 15 segundos, velocidad (0,75 a 2), la lista de reproducción y las notas del episodio (el texto del post), sin salir de la página.
- El punto donde la persona se quedó se guarda en su dispositivo.
- Un botón para copiar el feed.

> [!tip] En varios idiomas
> En un blog en varios idiomas, cada idioma tiene su propio feed, como `/podcast/feed.xml` y `/en/podcast/feed.xml`.
