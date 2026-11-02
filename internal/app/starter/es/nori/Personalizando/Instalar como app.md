---
title: Instalar como app
date: {{date-10}}
description: El blog se instala como app en el celular y en la computadora y se abre sin internet.
tags: [Personalización]
---
Todo sitio publicado con Blup se puede **instalar como app**, en el celular y en la computadora, y **se abre sin internet**. No hay que activar nada.

## Instalar

- **Android y computadora (Chrome, Edge):** aparece el botón **Instalar app** en el pie de página del sitio, y el navegador también ofrece el ícono de instalar en la barra de direcciones.
- **iPhone y iPad (Safari):** toca **Compartir** y después **Agregar a inicio**.

La app se abre en pantalla completa, con el ícono y el nombre del sitio.

## El ícono

Blup crea los íconos de cada dispositivo a partir del **favicon** del sitio: Android, iPhone, el ícono recortado en círculo de Android y el `favicon.ico`. Para cambiarlo, cambia el favicon en la pantalla **Sitio** del Studio (mira [Configurar el sitio](Configurar%20el%20sitio.md)). Una imagen cuadrada, de al menos 512 píxeles o en SVG, queda mejor.

## Sin internet

En la primera visita, el navegador guarda todas las páginas, los estilos y las fuentes. Después, el sitio se abre incluso sin internet.

- **Las fotos** se guardan a medida que la persona las ve, para no descargarlas todas de una vez.
- **Los comentarios y las reacciones** necesitan internet; sin ella, el resto de la página funciona normalmente.
- Con cada publicación, el navegador descarga la versión nueva solo.

La vista previa (la del Studio o la de `blup preview`) no se vuelve app, porque cambia todo el tiempo; solo el sitio publicado.
