---
title: Contraseñas y páginas ocultas
description: Protege páginas con contraseña y publica páginas que solo encuentra quien tiene el enlace.
weight: 4
---
## Contraseña

Cualquier página o post puede tener contraseña:

```yaml
password: mi-contraseña
```

Quien la abre ve el título y un campo para la contraseña. El contenido se **cifra** cuando se genera el sitio (AES-256-GCM): sin la contraseña, nadie lo lee, ni siquiera en el código de la página. No hay servidor.

Para usar la misma contraseña en varias páginas, define una **contraseña general** en `blup customize` y escribe:

```yaml
password: sí
```

La contraseña general queda solo en tu computadora, en el blup.toml.

> [!warning] Lo que sigue visible
> El título, la descripción y la portada aparecen en las listas. Las imágenes dentro de la página no se cifran.

## Páginas ocultas

```yaml
hidden: true
```

La página se publica, pero sale de las listas, de las etiquetas y de la búsqueda del sitio, y le pide a Google que no la muestre. Quien tiene el enlace la abre normalmente. Combina bien con contraseña.

## Borradores

```yaml
draft: true
```

El borrador aparece solo en la vista previa (`blup preview`) y nunca se publica.
