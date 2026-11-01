---
title: Cookies y estadísticas
date: {{date-11}}
description: Google Analytics, Meta Pixel, Umami, Plausible, videos de YouTube y el aviso de cookies.
tags: [Personalización, Studio]
---
Blup no pone cookies de rastreo propias. Esta parte es para cuando usas herramientas para **contar visitas** o **medir anuncios**, y para **videos de YouTube** en las notas.

## Herramientas listas

En el menú lateral, haz clic en **Cookies y estadísticas**. En **Herramientas**, haz clic en **Agregar herramienta**, elige cuál y pega solo su ID:

| Herramienta | Qué pegar | Dónde encontrarlo |
| --- | --- | --- |
| Google Analytics | ID de medición, `G-...` | Administrar › Flujos de datos |
| Google Tag Manager | ID del contenedor, `GTM-...` | Arriba en el espacio de trabajo, al lado del nombre del contenedor |
| Meta Pixel | ID del píxel, solo números | Administrador de eventos › Orígenes de datos |
| Umami | ID del sitio | Settings › Websites › Edit |
| Plausible | Dominio del sitio | El dominio registrado en Plausible |

¿Usas Umami o Plausible en tu propio servidor? Completa también **Servidor propio (opcional)** con la dirección del script.

**Google Tag Manager** puede tener etiquetas de estadísticas y de anuncios. Con el aviso, solo carga después de que la persona acepta **Estadísticas** o **Marketing**, y Blup informa lo que se aceptó mediante el Consent Mode de Google: las etiquetas de Google siguen cada elección. Las etiquetas de otras empresas dentro de Tag Manager deben configurarse para respetar el consentimiento.

## El aviso

En la misma pantalla, activa el **Aviso de cookies**. Activado:

- En la primera visita aparece un aviso con **Aceptar todo**, **Rechazar** y **Elegir**.
- En **Elegir**, la persona activa o desactiva cada grupo: **Estadísticas** (Google Analytics, Umami, Plausible), **Marketing** (Meta Pixel), Google Tag Manager en los dos y **Videos y contenido externo** (YouTube). Las **Esenciales** (tema claro u oscuro, iniciar sesión para comentar) siempre están activas.
- Nada de terceros carga antes de aceptar. La elección queda guardada en el navegador por un año.
- El enlace **Cookies** del pie de página vuelve a abrir las elecciones en cualquier momento.

Desactivado, las herramientas y los videos cargan directo, sin preguntar.

## Política de privacidad

En la misma pantalla, elige dónde está la **Política de privacidad**:

- **Una página del sitio (plantilla para editar)**: Blup crea la nota **Política de privacidad** en la carpeta del sitio, ya escrita a partir de las herramientas, los videos, los comentarios y el hosting. Haz clic en **Abrir la política**, revísala y edítala a tu gusto: es tuya y Blup nunca la reescribe. Un comentario arriba en la nota recuerda qué revisar y no aparece en el sitio.
- **Una dirección mía**, si la política ya existe en otro lugar.

Completa también el **Correo de contacto (opcional)**. La página aparece enlazada en el aviso de cookies y en el pie de página, queda fuera de las listas de posts y no recibe comentarios. Si cambias las herramientas después, actualiza la nota.

## Videos de YouTube

Pega el enlace del video como imagen:

```md
![Cómo publicar](https://www.youtube.com/watch?v=XXXXXXXXXXX)
```

En el sitio se vuelve el reproductor, mediante la dirección `youtube-nocookie.com`. Con el aviso activado, aparece el botón **Cargar video** hasta que la persona acepte los videos.

Para desactivarlo todo, quita las herramientas y desactiva el aviso en la misma pantalla.
