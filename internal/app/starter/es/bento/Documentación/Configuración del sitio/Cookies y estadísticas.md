---
title: Cookies y estadísticas
description: Google Analytics, Meta Pixel, Umami, Plausible y el aviso de cookies.
weight: 6
---
Blup no pone cookies de rastreo propias. Esta parte es para cuando usas herramientas para **contar visitas** o **medir anuncios**, y para los **videos de YouTube** en las notas.

## Desde Blup Studio

Haz clic en **Cookies y estadísticas** en la barra lateral.

1. En **Herramientas**, haz clic en **Agregar herramienta** y elige: Google Analytics, Google Tag Manager, Meta Pixel, Umami o Plausible.
2. Pega solo el **ID** de cada una. Si el formato no parece correcto, el Studio te avisa.
3. ¿Usas Umami o Plausible en tu propio servidor? Completa **Servidor propio (opcional)** con la dirección del script.
4. Activa o desactiva el **Aviso de cookies**.
5. En **Política de privacidad**, elige **Una página del sitio (plantilla para editar)** o **Una dirección mía**. Puedes indicar un **E-mail de contacto**.
6. **Abrir la política** abre la nota de la política en el editor.
7. Publica de nuevo para que se aplique en el sitio.

## Desde la línea de comandos

Ejecuta `blup customize` (o `blup personalizar`) y elige **Cookies, estadísticas de visitantes y anuncios**. Las preguntas son las mismas.

## Dónde encontrar cada ID

| Herramienta | Qué pegar | Dónde encontrarlo |
| --- | --- | --- |
| Google Analytics | ID de medición, `G-...` | Administrar › Flujos de datos |
| Google Tag Manager | ID del contenedor, `GTM-...` | Arriba en el espacio de trabajo, junto al nombre del contenedor |
| Meta Pixel | ID del píxel, solo números | Administrador de eventos › Orígenes de datos |
| Umami | ID del sitio | Settings › Websites › Edit |
| Plausible | Dominio del sitio | El dominio registrado en Plausible |

**Google Tag Manager** puede tener etiquetas de estadísticas y de anuncios. Con el aviso, solo se carga después de que la persona acepta **Estadísticas** o **Marketing**, y Blup informa lo que se aceptó mediante el Consent Mode de Google. Las etiquetas de otras empresas dentro de Tag Manager se deben configurar para respetar el consentimiento.

## El aviso

Activado:

- En la primera visita aparece un aviso con **Aceptar todo**, **Rechazar** y **Elegir**.
- En **Elegir**, la persona activa o desactiva cada grupo: **Estadísticas** (Google Analytics, Umami, Plausible), **Marketing** (Meta Pixel), Google Tag Manager en los dos y **Videos y contenido externo** (YouTube). Los **Esenciales** (tema claro u oscuro, iniciar sesión para comentar) siempre están activos.
- Nada de terceros se carga antes de la aceptación. La elección queda guardada en el navegador por un año.
- El enlace **Cookies** del pie de página vuelve a abrir las opciones en cualquier momento.

Desactivado, las herramientas y los videos se cargan directo, sin preguntar.

## Política de privacidad

- **Una página del sitio** (recomendado): Blup crea la nota **Política de privacidad**, ya escrita a partir de las herramientas que marcaste, los videos, los comentarios y el alojamiento. Revísala y edítala a tu gusto: es tuya y Blup nunca la reescribe. Un comentario al inicio de la nota recuerda qué revisar y no aparece en el sitio.
- **Una dirección tuya**, si la política ya existe en otro lugar.

La página aparece con un enlace en el aviso de cookies y en el pie de página, queda fuera de las listas de posts y no recibe comentarios. Si cambias las herramientas después, actualiza la nota.

## Videos de YouTube

Pega el enlace del video como si fuera una imagen:

```md
![Cómo publicar](https://www.youtube.com/watch?v=XXXXXXXXXXX)
```

En el sitio se convierte en el reproductor, desde la dirección `youtube-nocookie.com`. Con el aviso activado, aparece el botón **Cargar video** hasta que la persona acepta los videos.
