---
title: Enlaces e imágenes
description: Enlaces entre notas, enlaces a otros sitios, imágenes, videos y adjuntos.
weight: 2
---
## Enlaces entre notas

- **En Blup Studio**: escribe `[[`, elige la nota y listo. O usa `/` y **Enlace a otra nota**. Mira [Escribir en el editor](../Blup%20Studio/Escribir%20en%20el%20editor.md).
- **En cualquier editor de texto**: escribe el enlace Markdown a mano, con la ruta de la nota.

En los dos casos, el enlace queda así, con ruta relativa:

```md
[Formato y callouts](Formato%20y%20callouts.md)
```

En el sitio se convierte en el enlace de la página: [Formato y callouts](Formato%20y%20callouts.md). Los enlaces a un título de la nota (`Nota.md#titulo`) también funcionan.

> [!tip] ¿Renombraste o moviste? Los enlaces te siguen
> En el Studio, **Renombrar** y **Mover a…** corrigen solos los enlaces de las otras notas.

> [!tip] Notas antiguas con wikilinks
> Ejecuta `blup convert` (o `blup converter`). Blup muestra cuántos enlaces va a cambiar y, si confirmas, cambia todos los wikilinks de la carpeta del sitio por el formato estándar. Antes, guarda una copia de las notas afectadas.

## Enlaces a otros sitios

Escribe normalmente: `[Blup](https://getblup.io)`. En el Studio, el botón de enlace (o `Ctrl+K`) ya pone el formato y selecciona la dirección para que la pegues.

## Imágenes

En el Studio, pega (`Ctrl+V`) o arrastra la imagen al editor. Va a **Adjuntos** y aparece en su lugar.

Al publicar, Blup:

- genera versiones más pequeñas y en WebP de cada imagen;
- entrega al navegador solo el tamaño que necesita;
- usa el texto entre corchetes como descripción (bueno para la accesibilidad y para Google).

![Blup en la terminal](../../Adjuntos/blup-terminal.jpg)

## Videos de YouTube

Pega el enlace del video como si fuera una imagen: `![Cómo publicar](https://www.youtube.com/watch?v=...)`. En el sitio se convierte en el reproductor. Con el aviso de cookies activado, el video espera la aceptación; mira [Cookies y estadísticas](../Configuración%20del%20sitio/Cookies%20y%20estadísticas.md).

## Otros adjuntos

Los PDF, audios, videos y archivos ZIP también se pueden enlazar, como `[Manual en PDF](../../Adjuntos/manual.pdf)`. Se publican junto con el sitio. Mira la pantalla [Adjuntos](../Blup%20Studio/Adjuntos.md).
