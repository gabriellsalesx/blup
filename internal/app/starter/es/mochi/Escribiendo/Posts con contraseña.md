---
title: Posts con contraseña
date: {{date-5}}
description: Protege un post con contraseña, sin servidor y sin que nadie lo lea en el código de la página.
tags: [Obsidian, Funciones]
image: "[[blup-senha.jpg]]"
---
Cualquier post puede tener contraseña. Quien lo abre ve el título y un campo para escribir la contraseña; el texto solo aparece con ella.

## Cómo protegerlo

Agrega la propiedad **password** a la nota:

```yaml
password: mi-contraseña
```

Para no repetir la misma contraseña en varios posts, define una **contraseña general** en `blup customize` y usa:

```yaml
password: sí
```

Míralo funcionando: [Un post bloqueado](Un%20post%20bloqueado.md). La contraseña es `blup`.

## ¿Es seguro?

El texto se **cifra** cuando se genera el sitio (AES-256). Sin la contraseña, nadie puede leerlo, ni siquiera mirando el código de la página. No hay ningún servidor guardando nada.

> [!warning] Lo que no se protege
> Las imágenes dentro del post no se cifran. El título, la descripción y la portada siguen visibles, porque aparecen en las listas.

## Ocultar de las listas

Para un post que solo debe encontrar quien tiene el enlace, usa también `hidden: true`: sale de la página de inicio, de las etiquetas y de Google, pero sigue en línea.
