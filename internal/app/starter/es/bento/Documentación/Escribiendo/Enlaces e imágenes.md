---
title: Enlaces e imágenes
description: Enlaces entre notas, enlaces externos, imágenes y adjuntos.
weight: 2
---
## Enlaces entre notas

Escribe `[[` y elige la nota, o arrastra la nota al texto. Con el ajuste que Blup hizo en Obsidian, el enlace se guarda en el formato estándar, con ruta relativa:

```md
[Formato y callouts](Formato%20y%20callouts.md)
```

En el sitio se vuelve el enlace de la página: [Formato y callouts](Formato%20y%20callouts.md). Los enlaces a un título de la nota (`Nota.md#titulo`) también funcionan.

> [!tip] ¿Notas antiguas con [[wikilinks]]?
> Ejecuta `blup convert`. Blup muestra cuántos enlaces va a cambiar y, si confirmas, cambia todos los `[[links]]` de la carpeta del sitio por el formato estándar.

## Enlaces a otros sitios

Escríbelos normalmente: `[Obsidian](https://obsidian.md)`. Abren en el mismo lugar, con seguridad (`noopener`).

## Imágenes

Arrastra la imagen a la nota. Blup:

- genera versiones más pequeñas y en WebP de cada imagen;
- le entrega al navegador solo el tamaño que necesita;
- usa el texto entre corchetes como descripción (bueno para la accesibilidad y para Google).

![Blup en la terminal](../../Adjuntos/blup-terminal.jpg)

## Videos de YouTube

Pega el enlace del video como imagen: `![Cómo publicar](https://www.youtube.com/watch?v=...)`. En el sitio se vuelve el reproductor. Con el aviso de cookies activado, el video espera la aceptación; mira [Cookies y estadísticas](../Configuración/Cookies%20y%20estadísticas.md).

## Otros adjuntos

Los PDF, audios, videos y archivos ZIP de la bóveda también se pueden enlazar: `[Manual en PDF](../../Adjuntos/manual.pdf)`. Se publican junto con el sitio.
