---
title: Ajustes del sitio
description: Nombre, descripción, logo, favicon y redes están en la nota _index de la carpeta del sitio.
weight: 1
---
Los ajustes del sitio son **propiedades de la nota _index** de la carpeta del sitio. Cambia en Obsidian, cambia en el sitio. En la terminal, `blup customize` hace las mismas preguntas y guarda en las mismas propiedades.

| Propiedad | Qué hace |
| --- | --- |
| `site_name` | Nombre del sitio: pestaña del navegador, pie de página y Google. |
| `site_description` | Una frase sobre el sitio, para Google y las redes. |
| `logo` | Imagen del logo arriba, como `"[[logo.svg]]"`. |
| `favicon` | Ícono de la pestaña del navegador, como `"[[favicon.png]]"`. |
| `menu` | Elementos del menú superior. Mira [Menú y botón](Menú%20y%20botón.md). |
| `button_text` y `button_link` | El botón de la esquina superior derecha. |
| `social` | Enlaces de tus perfiles, con ícono en el pie de página. |

## Ejemplo

```yaml
---
title: Acme
description: La forma simple de hacer X.
site_name: Acme
site_description: La forma simple de hacer X.
logo: "[[acme-logo.svg]]"
favicon: "[[acme-icono.png]]"
social:
  - https://github.com/acme
  - https://www.linkedin.com/company/acme
  - mailto:hola@acme.com
---
```

## Redes sociales

Blup reconoce la red por la dirección y pone el ícono correcto: Instagram, X, LinkedIn, YouTube, GitHub, Facebook, WhatsApp (`wa.me`) y correo (`mailto:`). Otras direcciones reciben un ícono de enlace.

## Logo y favicon

- En **SVG**, se ven nítidos en cualquier pantalla.
- En **PNG**, Blup genera los tamaños que piden los navegadores y los celulares.
- Sin logo, el sitio muestra la mascota. Sin favicon, también.

> [!note] ¿Y el blup.toml?
> En la carpeta del sitio en la computadora, el **blup.toml** guarda lo que no es contenido: dónde publicar, el dominio propio, la contraseña general y el repositorio de la sincronización. Todo eso se cambia con `blup customize`, sin abrir el archivo. El nombre y la descripción que diste en el asistente también quedan ahí, como respaldo, pero lo que vale es la nota _index.
