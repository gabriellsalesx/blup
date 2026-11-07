---
title: Tu blog se vuelve podcast
date: {{date-15}}
description: Las narraciones de los posts se vuelven episodios, con reproductor en el blog y feed para Spotify y Apple Podcasts.
tags: [Personalización, IA, Studio]
---
¿Narraste los posts? Entonces tu blog ya tiene un podcast listo. La sección **Podcast** reúne las narraciones como episodios, del más nuevo al más antiguo, en una página con reproductor propio, y publica un feed para Spotify, Apple Podcasts y otras apps.

## Activar la sección

En el Studio, haz clic en **Secciones**, en el menú lateral, y agrega **Podcast**. El blog gana la carpeta **Podcast**, con una nota que tú editas, y el menú lateral gana la pantalla **Podcast**.

Los episodios son los posts con narración: mira [Narrar un post en audio](../Escribiendo/Narrar%20un%20post%20en%20audio.md). La portada de cada episodio es la portada del post. Los posts ocultos, los borradores y los posts con contraseña quedan fuera.

## La página del podcast

- Un reproductor fijo, que pasa solo al siguiente episodio, incluso en el celular con la pantalla apagada.
- Retroceder y adelantar 15 segundos, velocidad de 0,75 a 2 veces y la lista de reproducción.
- Las notas de cada episodio (el texto del post) se abren ahí mismo, sin detener el audio.
- Quien sale y vuelve encuentra el episodio en el punto donde lo dejó.
- Un botón para copiar la dirección del feed.

## La pantalla Podcast

En la pantalla **Podcast** del Studio están los datos del programa:

- **Nombre del podcast** (si está vacío, se usa el nombre del sitio) y **Descripción**.
- **Autor** y **E-mail de contacto**. El e-mail queda público en el feed: así Spotify y Apple confirman que el podcast es tuyo.
- **Categoría**, de la lista de Apple Podcasts, y **Contenido explícito**.
- **Portada del podcast**: un cuadrado de 1400 a 3000 píxeles, en JPG o PNG.

La pantalla también muestra cuántos episodios hay y el **Feed del podcast**, con **Copiar** y los atajos para enviarlo a Spotify y a Apple Podcasts. Pega la dirección una sola vez: los episodios nuevos llegan solos cuando publicas el sitio.

> [!warning] Necesita una dirección completa
> Spotify y Apple necesitan la dirección completa del feed. Después de publicar el sitio (con Blup Cloud, Cloudflare Workers o GitHub Pages), la dirección completa aparece en la pantalla Podcast.

> [!info] ¿Y con la línea de comandos?
> La sección Podcast se activa en el Studio. Una vez activada, los datos del programa quedan en las propiedades de la nota **_index** de la carpeta Podcast, y la página y el feed se siguen generando cuando publicas con la línea de comandos.
