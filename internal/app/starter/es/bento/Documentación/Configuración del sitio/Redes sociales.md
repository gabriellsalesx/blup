---
title: Redes sociales
description: Los íconos del pie de página, con un botón para cada red.
weight: 3
---
Los perfiles del sitio aparecen como íconos en el pie de página. Los autores también pueden tener sus redes, en la página de cada uno.

## Desde Blup Studio

En la pantalla **Sitio**, bloque **Redes sociales**:

1. Haz clic en el botón de la red: **Instagram**, **X**, **LinkedIn**, **YouTube**, **Discord**, **GitHub**, **Facebook**, **WhatsApp** o **e-mail**.
2. El Studio agrega una línea con el comienzo de la dirección, como `https://instagram.com/`, y pone el cursor al final.
3. Complétala con tu perfil (o el e-mail).

Si falta el resto de la dirección, el Studio te avisa: una línea que solo tiene el comienzo no va al sitio.

En la pantalla **Autores**, el perfil de cada persona tiene los mismos botones.

## Desde Obsidian

En la nota **_index** de la carpeta del sitio:

```yaml
social:
  - https://instagram.com/acme
  - https://github.com/acme
  - https://wa.me/5585999999999
  - mailto:hola@acme.com
```

Blup reconoce la red por la dirección y pone el ícono correcto. Las direcciones que no conoce reciben un ícono de enlace.

## Desde la línea de comandos

`blup customize` (o `blup personalizar`) > redes sociales pregunta las direcciones una por una.
