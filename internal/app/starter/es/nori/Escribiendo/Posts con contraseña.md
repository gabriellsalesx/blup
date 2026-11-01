---
title: Posts con contraseña
date: {{date-5}}
description: Protege un post con contraseña, sin servidor y sin que nadie lo lea en el código de la página.
tags: [Escritura, Funciones]
image: "[[blup-senha.jpg]]"
---
Cualquier post puede tener contraseña. Quien lo abra ve el título y un campo para escribir la contraseña; el texto solo aparece con ella.

## Cómo protegerlo

En el panel **Propiedades** del post, activa **Contraseña** y elige:

- **Contraseña propia**: escribe la contraseña de este post.
- **Contraseña general del sitio**: usa la contraseña definida en la pantalla **Contraseña**, en el menú lateral. Ideal para no repetir la misma contraseña en varios posts.

La contraseña general queda solo en tu computadora, nunca en el sitio.

Mírala funcionando: [Un post bloqueado](Un%20post%20bloqueado.md). La contraseña es `blup`.

## ¿Es seguro?

El texto se **cifra** cuando se genera el sitio (AES-256). Sin la contraseña, nadie puede leerlo, ni mirando el código de la página. No hay ningún servidor guardando nada.

> [!warning] Lo que no queda protegido
> Las imágenes dentro del post no se cifran. El título, la descripción y la portada siguen visibles, porque aparecen en las listas.

## Ocultar de las listas

Para un post que solo debe encontrar quien tenga el enlace, activa también **Oculto**: sale de la página de inicio, de las etiquetas y de Google, pero sigue en línea.
