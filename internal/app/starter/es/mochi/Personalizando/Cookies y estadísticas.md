---
title: Cookies y estadísticas
date: {{date-11}}
description: Google Analytics, Meta Pixel, Umami, Plausible, videos de YouTube y el aviso de cookies.
tags: [Personalización, CLI]
---
Blup no pone cookies de rastreo propias. Esta parte es para cuando usas herramientas para **contar visitas** o **medir anuncios**, y para **videos de YouTube** en las notas.

## Herramientas listas

Ejecuta `blup customize` y elige **Cookies, estadísticas de visitantes y anuncios**. Marca las herramientas que usa el sitio y pega el código de cada una:

| Herramienta | Qué pegar | Dónde encontrarlo |
| --- | --- | --- |
| Google Analytics | ID de medición, `G-...` | Administrar › Flujos de datos |
| Google Tag Manager | ID del contenedor, `GTM-...` | Arriba en el espacio de trabajo, junto al nombre del contenedor |
| Meta Pixel | ID del píxel, solo números | Administrador de eventos › Orígenes de datos |
| Umami | ID del sitio | Settings › Websites › Edit |
| Plausible | Dominio del sitio | El dominio registrado en Plausible |

¿Usas Umami o Plausible en tu propio servidor? Blup te pregunta la dirección del script.

**Google Tag Manager** puede tener etiquetas de estadísticas y de anuncios. Con el aviso activado, solo carga después de que la persona acepta **Estadísticas** o **Marketing**, y Blup informa lo que se aceptó mediante el Consent Mode de Google: las etiquetas de Google siguen cada elección. Las etiquetas de otras empresas dentro de Tag Manager deben configurarse para respetar el consentimiento.

## El aviso

En el mismo lugar, Blup pregunta si debe mostrar el **aviso de cookies**. Activado:

- En la primera visita aparece un aviso con **Aceptar todo**, **Rechazar** y **Elegir**.
- En **Elegir**, la persona activa o desactiva cada grupo: **Estadísticas** (Google Analytics, Umami, Plausible), **Marketing** (Meta Pixel), Google Tag Manager en los dos y **Videos y contenido externo** (YouTube). Las **Esenciales** (tema claro u oscuro, iniciar sesión para comentar) siempre están activas.
- Nada de terceros carga antes de aceptar. La elección queda guardada en el navegador durante un año.
- El enlace **Cookies** en el pie de página vuelve a abrir las opciones en cualquier momento.
Desactivado, las herramientas y los videos cargan directamente, sin preguntar.

## Política de privacidad

Al activar el aviso o alguna herramienta, Blup pregunta dónde está la política de privacidad:

- **Una página del sitio** (recomendado): Blup crea la nota **Política de privacidad** en la carpeta del sitio, ya escrita a partir de las herramientas que marcaste, los videos, los comentarios y el hosting. Ábrela en Obsidian, revísala y edítala como quieras: es tuya y Blup nunca la reescribe. Un comentario al inicio de la nota te recuerda qué revisar y no aparece en el sitio.
- **Una dirección tuya**, si la política ya existe en otro lugar.

La página aparece enlazada en el aviso de cookies y en el pie de página, queda fuera de las listas de posts y no recibe comentarios. Si cambias las herramientas después, actualiza la nota.

## Videos de YouTube

Pega el enlace del video como imagen, de la forma en que Obsidian lo muestra:

```md
![Cómo publicar](https://www.youtube.com/watch?v=XXXXXXXXXXX)
```

En el sitio se convierte en el reproductor, con la dirección `youtube-nocookie.com`. Con el aviso activado, aparece el botón **Cargar video** hasta que la persona acepte los videos.

Para desactivar todo, ejecuta `blup customize` otra vez, desmarca las herramientas y responde **No** al aviso.
