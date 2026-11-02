---
title: Cookies y estadísticas
date: {{date-11}}
description: Google Analytics, Meta Pixel, Umami, Plausible, videos de YouTube y el aviso de cookies.
tags: [Personalización, Studio]
---
Blup no pone cookies de seguimiento propias. Esta parte es para cuando usas herramientas para **contar visitas** o **medir anuncios**, y para los **videos de YouTube** en las notas.

## Herramientas listas

En el menú lateral, haz clic en **Cookies y estadísticas**. En **Herramientas**, haz clic en **Agregar herramienta**, elige cuál y pega solo su ID:

| Herramienta | Qué pegar | Dónde encontrarlo |
| --- | --- | --- |
| Google Analytics | ID de medición, `G-...` | Administrar › Flujos de datos |
| Google Tag Manager | ID del contenedor, `GTM-...` | Arriba en el espacio de trabajo, junto al nombre del contenedor |
| Meta Pixel | ID del píxel, solo números | Administrador de eventos › Orígenes de datos |
| Umami | ID del sitio | Settings › Websites › Edit |
| Plausible | Dominio del sitio | El dominio registrado en Plausible |

¿Usas Umami o Plausible en tu propio servidor? Completa también **Servidor propio (opcional)** con la dirección del script.

**Google Tag Manager** puede tener etiquetas de estadísticas y de anuncios. Con el aviso, solo se carga después de que la persona acepta **Estadísticas** o **Marketing**, y Blup informa lo que se aceptó con el Consent Mode de Google: las etiquetas de Google siguen cada elección. Las etiquetas de otras empresas dentro de Tag Manager tienen que configurarse para respetar el consentimiento.

## El aviso

En la misma pantalla, activa el **Aviso de cookies**. Activado:

- En la primera visita aparece un aviso con **Aceptar todo**, **Rechazar** y **Elegir**.
- En **Elegir**, la persona activa o desactiva cada grupo: **Estadísticas** (Google Analytics, Umami, Plausible), **Marketing** (Meta Pixel), Google Tag Manager en los dos y **Videos y contenido externo** (YouTube). Los **Esenciales** (tema claro u oscuro, iniciar sesión para comentar) siempre están activados.
- Nada de terceros se carga antes de aceptar. La elección queda guardada en el navegador por un año.
- El enlace **Cookies** en el pie de página vuelve a abrir las opciones en cualquier momento.

Desactivado, las herramientas y los videos se cargan directo, sin preguntar.

## Política de privacidad

En la misma pantalla, elige dónde está la **Política de privacidad**:

- **Una página del sitio (plantilla para editar)**: Blup crea la nota **Política de privacidad** en la carpeta del sitio, ya escrita a partir de las herramientas, los videos, los comentarios y el hosting. Haz clic en **Abrir la política**, revísala y edítala como quieras: es tuya y Blup nunca la reescribe. Un comentario al principio de la nota recuerda qué revisar y no aparece en el sitio.
- **Una dirección mía**, si la política ya existe en otro lugar.

Completa también el **Correo de contacto (opcional)**. La página aparece con un enlace en el aviso de cookies y en el pie de página, queda fuera de las listas de posts y no recibe comentarios. Si cambias las herramientas después, actualiza la nota.

## Videos de YouTube

Pega el enlace del video como si fuera una imagen:

```md
![Cómo publicar](https://www.youtube.com/watch?v=XXXXXXXXXXX)
```

En el sitio se vuelve el reproductor, con la dirección `youtube-nocookie.com`. Con el aviso activado, aparece el botón **Cargar video** hasta que la persona acepta los videos.

Para desactivarlo todo, quita las herramientas y desactiva el aviso en la misma pantalla.

> [!info] Con Obsidian
> Ejecuta `blup customize` y elige **Cookies, estadísticas de visitantes y anuncios**. El terminal pregunta lo mismo: las herramientas, los ID, el aviso y la política de privacidad.
