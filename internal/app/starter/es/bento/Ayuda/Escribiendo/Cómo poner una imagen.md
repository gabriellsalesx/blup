---
title: ¿Cómo pongo una imagen?
description: Pega o arrastra la imagen dentro de la nota.
weight: 2
---
## En Blup Studio

Pega la imagen (o arrastra el archivo) dentro de la nota. El Studio guarda la imagen en la carpeta de adjuntos y pone el enlace en el texto. La imagen ya aparece en el editor.

Para la **portada** de un post, elige la imagen en **Propiedades**, en el campo de la imagen.

## En el archivo de la nota

Guarda la imagen en la carpeta de adjuntos y enlázala en el texto. Para la portada, usa la propiedad `image`:

```yaml
image: "[[mi-portada.jpg]]"
```

Blup redimensiona y convierte las imágenes por sí solo. Las portadas se ven bien en 1600x900.
