---
title: Contraseñas y páginas ocultas
description: Protege páginas con contraseña, oculta páginas de las listas y guarda borradores.
weight: 7
---
![Blup con un candado](../../Adjuntos/blup-senha.jpg)

## Contraseña

Cualquier página o post puede pedir contraseña. Quien la abre ve el título y un campo para la contraseña. El contenido se **cifra en tu computadora**, cuando se genera el sitio (AES-256-GCM): sin la contraseña, nadie lo lee, ni siquiera mirando el código de la página.

### Desde Blup Studio

En las **Propiedades** de la nota, en **Contraseña**, elige:

- **Contraseña general**: la misma contraseña para varios posts. Está en **Configuración > Contraseña general** y vale para todos tus sitios.
- **Contraseña propia**: una contraseña solo para este post, en **Contraseña de este post**.

### Desde Obsidian

```yaml
password: mi-contraseña
```

Para usar la contraseña general, escribe:

```yaml
password: sí
```

En la línea de comandos, `blup customize` (o `blup personalizar`) también define una contraseña general solo para ese sitio. Las contraseñas generales se quedan solo en tu computadora, nunca en el sitio.

> [!warning] Lo que sigue visible
> El título, la descripción y la portada aparecen en las listas. Las imágenes dentro de la página no se cifran. Los posts con contraseña tampoco se narran ni entran en el podcast, porque el audio revelaría el texto.

## Páginas ocultas

En el Studio, activa **Oculto** en las propiedades. En Obsidian:

```yaml
hidden: true
```

La página se publica, pero sale de las listas, de las etiquetas y de la búsqueda del sitio, y le pide a Google que no la muestre. Quien tiene el enlace la abre con normalidad. Combina bien con contraseña.

## Borradores

En el Studio, activa **Borrador**. En Obsidian:

```yaml
draft: true
```

El borrador aparece solo en la vista previa y nunca se publica.

## Partes que solo tú ves

Lo que está entre `%%` tampoco va nunca al sitio. Mira [Fórmulas, código y comentarios](../Escribiendo/Fórmulas,%20código%20y%20comentarios.md).
