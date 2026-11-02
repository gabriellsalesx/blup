---
title: Posts con contraseña
date: {{date-5}}
description: Protege un post con contraseña, sin servidor y sin que nadie lo lea en el código de la página.
tags: [Escritura, Funciones]
image: "[[blup-senha.jpg]]"
---
Cualquier post puede tener contraseña. Quien lo abre ve el título y un campo para escribir la contraseña; el texto solo aparece con ella.

Míralo funcionando: [Un post bloqueado](Un%20post%20bloqueado.md). La contraseña es `blup`.

## En el Studio

En el panel **Propiedades** del post, activa **Contraseña** y elige:

- **Contraseña propia**: escribe la contraseña de este post.
- **Contraseña general**: una sola contraseña para varios posts. Está en la **Configuración** del Studio y vale para todos tus sitios. Si quieres, un sitio puede tener su propia contraseña, que vale solo para él.

## En el archivo de la nota

Agrega la propiedad **password** a la nota:

```yaml
password: mi-contraseña
```

Para usar la contraseña general, escribe:

```yaml
password: sí
```

Con la línea de comandos, la contraseña general se define en `blup customize`.

## ¿Es seguro?

El texto se **cifra** en tu computadora, cuando se genera el sitio (AES-256). Sin la contraseña, nadie puede leerlo, ni mirando el código de la página. No hay ningún servidor guardando nada, y la contraseña nunca llega al sitio.

> [!warning] Lo que no está protegido
> Las imágenes dentro del post no se cifran. El título, la descripción y la portada siguen visibles, porque aparecen en las listas. Los posts con contraseña tampoco tienen narración ni entran al podcast, porque el audio revelaría el texto.

## Ocultarlo de las listas

Para un post que solo debe encontrar quien tiene el enlace, activa también **Oculto** (`hidden: true`): sale de la página de inicio, de las etiquetas y de Google, pero sigue en línea.
