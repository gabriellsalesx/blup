---
title: Cookies y estadísticas
description: Google Analytics, Meta Pixel, Umami, Plausible y el aviso de cookies.
weight: 7
---
Blup no pone cookies de rastreo propias. Esta parte es para cuando usas herramientas para **contar visitas** o **medir anuncios**, y para **videos de YouTube** en las notas.

## Herramientas listas

Ejecuta `blup customize` y elige **Cookies, estadísticas de visitantes y anuncios**. Marca las herramientas que usa el sitio y pega el código de cada una:

| Herramienta | Qué pegar | Dónde encontrarlo |
| --- | --- | --- |
| Google Analytics | ID de medición, `G-...` | Administrar › Flujos de datos |
| Google Tag Manager | ID del contenedor, `GTM-...` | En la parte superior del espacio de trabajo, junto al nombre del contenedor |
| Meta Pixel | ID del píxel, solo números | Administrador de eventos › Orígenes de datos |
| Umami | ID del sitio | Settings › Websites › Edit |
| Plausible | Dominio del sitio | El dominio registrado en Plausible |

¿Usas Umami o Plausible en tu propio servidor? Blup pregunta la dirección del script.

**Google Tag Manager** puede tener etiquetas de estadísticas y de anuncios. Con el aviso, solo carga después de que la persona acepta **Estadísticas** o **Marketing**, y Blup informa lo aceptado mediante el Consent Mode de Google: las etiquetas de Google siguen cada elección. Las etiquetas de otras empresas dentro de Tag Manager tienen que configurarse para respetar el consentimiento.

## El aviso

En el mismo lugar, Blup pregunta si muestra el **aviso de cookies**. Activado:

- En la primera visita aparece un aviso con **Aceptar todo**, **Rechazar** y **Elegir**.
- En **Elegir**, la persona activa o desactiva cada grupo: **Estadísticas** (Google Analytics, Umami, Plausible), **Marketing** (Meta Pixel), Google Tag Manager en los dos y **Videos y contenido externo** (YouTube). Los **Esenciales** (tema claro u oscuro, iniciar sesión para comentar) quedan siempre activados.
- Nada de terceros carga antes de aceptar. La elección queda guardada en el navegador por un año.
- El enlace **Cookies** en el pie de página reabre las opciones en cualquier momento.

Desactivado, las herramientas y los videos cargan directo, sin preguntar.

## Política de privacidad

Al activar el aviso o alguna herramienta, Blup pregunta dónde está la política de privacidad:

- **Una página del sitio** (recomendado): Blup crea la nota **Política de privacidad** en la carpeta del sitio, ya escrita a partir de las herramientas que marcaste, los videos, los comentarios y el hosting. Ábrela en Obsidian, revísala y edítala como quieras: es tuya y Blup nunca la reescribe. Un comentario arriba de la nota recuerda qué revisar y no aparece en el sitio.
- **Una dirección tuya**, si la política ya existe en otro lugar.

La página aparece enlazada en el aviso de cookies y en el pie de página, queda fuera de las listas de posts y no recibe comentarios. Si cambias las herramientas después, actualiza la nota.

## Videos de YouTube

Pega el enlace del video como imagen, como lo muestra Obsidian:

```md
![Cómo publicar](https://www.youtube.com/watch?v=XXXXXXXXXXX)
```

En el sitio se vuelve el reproductor, mediante la dirección `youtube-nocookie.com`. Con el aviso activado, aparece el botón **Cargar video** hasta que la persona acepta los videos.

Para desactivar todo, ejecuta `blup customize` de nuevo, desmarca las herramientas y responde **No** al aviso.
