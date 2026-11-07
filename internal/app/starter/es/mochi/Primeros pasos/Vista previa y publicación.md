---
title: Vista previa y publicación
date: {{date-2}}
description: Mira el sitio mientras escribes y publícalo con Blup Cloud, Cloudflare Workers o GitHub Pages.
tags: [Primeros pasos, Studio]
image: "[[blup-publicar.jpg]]"
---
## La vista previa

En el Studio, haz clic en **Vista previa**, arriba. El sitio aparece al lado del editor y **se actualiza solo** mientras escribes. Si cambias de nota, la vista previa va a su página; si haces clic en un enlace de la vista previa, el editor abre la nota de esa página.

La vista previa también muestra los borradores, para que los revises antes de publicar.

> [!info] Por la línea de comandos
> Ejecuta `blup preview`. El sitio se abre en el navegador y se actualiza cada vez que guardas una nota. Para detenerlo, presiona Ctrl+C en el terminal.

## Dónde publicar

Haz clic en **Publicar**, arriba. La primera vez, elige dónde va a estar el sitio:

- **Blup Cloud**: una dirección como `tusitio.getblup.site`, sin configurar nada. Viene en los planes de pago, desde Pro.
- **Cloudflare Workers**: gratis, con una dirección como `tusitio.tucuenta.workers.dev`. Necesitas una cuenta de Cloudflare.
- **GitHub Pages**: gratis, con una dirección como `tunombre.github.io/tusitio`. Necesitas una cuenta de GitHub.

### Con Blup Cloud

Inicia sesión con tu cuenta de Blup y escribe la dirección que quieres. Mientras escribes, el Studio revisa si el nombre está libre y, si no lo está, sugiere otros parecidos. La dirección queda fija después de la primera publicación.

Cada plan tiene un número de sitios: Pro tiene 1, Max tiene 3 y Ultra tiene 10. El dominio propio en los sitios de Blup Cloud está por llegar, en los planes Max y Ultra.

### Con Cloudflare o con GitHub

Conecta la cuenta una vez:

1. Haz clic en **Conectar la cuenta** y después en **Abrir en el navegador**. El formulario de la clave se abre con los permisos correctos ya elegidos.
2. Crea la clave y cópiala.
3. Pega la clave en el Studio y haz clic en **Conectar**. Se guarda solo en tu computadora.

Listo: haz clic en **Publicar ahora**. Cuando termina, el Studio muestra la dirección del sitio.

> [!info] Por la línea de comandos
> Ejecuta `blup publish`. La primera vez pregunta dónde publicar (Cloudflare Workers o GitHub Pages), abre la página de la clave y te pide pegarla en el terminal.

Las siguientes veces, solo se envía lo que cambió.

> [!tip] Publicar siempre es igual
> Escribiste, revisaste en la vista previa, hiciste clic en Publicar. No hay servidor que mantener ni base de datos.

## Dominio propio

¿Quieres una dirección tuya, como `misitio.com`? El dominio propio se vincula solo con Blup Cloud, en los planes Max y Ultra. Con el sitio en línea en Blup Cloud, ve a la pantalla **Publicar**, en **Dominio propio**, y escribe la dirección. El Studio muestra los registros DNS que debes crear donde compraste el dominio. Con Cloudflare o con GitHub, el sitio queda en la dirección gratis del servicio.

## Qué se publica

El sitio es público: cualquier persona con la dirección puede leerlo. Por eso solo se publica lo que tú publicaste:

- Todas las notas de la carpeta del sitio, menos las marcadas como **Borrador** (`draft: true`).
- Los comentarios entre `%%` quedan fuera.
- Solo los adjuntos que usan las notas, optimizados.
- Los posts con contraseña salen cifrados de tu computadora.
- La búsqueda, el feed RSS, el mapa del sitio y la tarjeta para compartir de cada página.
