---
title: Autores
description: Un perfil por persona, con foto, cargo, redes y bio.
weight: 4
---
Cada autor tiene una página con foto, cargo, bio, redes y sus posts. En los posts, la foto y el nombre aparecen junto al título.

![Los autores](../../Adjuntos/blup-autores.jpg)

## Desde Blup Studio

Haz clic en **Autores** en la barra lateral. La pantalla muestra las personas, con el número de posts de cada una, y marca al **Autor del sitio**.

- **Nuevo autor** abre el formulario: **Nombre** (como aparece en los posts), foto, **Cargo**, **Bio** y **Redes sociales**.
- Para la foto, elige uno de los avatares listos o **Subir foto propia**.
- **Editar** cambia todo. Cambiar el nombre actualiza los posts de esa persona.
- **Eliminar autor**: los posts siguen con el nombre, sin foto. Se puede restaurar desde la Papelera.

En los posts, elige los autores en el campo **Autores** de las [Propiedades de la nota](../Blup%20Studio/Propiedades%20de%20la%20nota.md).

## Desde Obsidian

Los perfiles están en una carpeta cuya **_index** tiene `blup: authors`, aquí llamada **Autores**. Esa carpeta no se convierte en página: sus notas son solo los datos.

Una nota con el nombre de la persona, como `Ana Lima.md`:

```yaml
---
photo: "[[ana.jpg]]"
role: Diseñadora de producto
social:
  - https://github.com/ana
---
Ana diseña el producto y escribe sobre cómo se hace.
```

- **photo**: la foto. Sin ella, aparecen las iniciales (o la mascota, para el autor del sitio).
- **role**: el cargo.
- **social**: los perfiles, con íconos automáticos.
- El **texto** de la nota es la bio.

En los posts:

```yaml
authors: [Ana Lima, Bruno Costa]
```

Sin **authors**, el post es del autor del sitio. Con más de uno, las fotos aparecen una al lado de la otra y los nombres juntos ("Ana Lima y Bruno Costa").

## Desde la línea de comandos

`blup customize` (o `blup personalizar`) > **Nombre, descripción y autor** edita el nombre, el cargo, la bio y la foto del autor del sitio.

> [!tip] Perfiles en otros idiomas
> En un sitio en varios idiomas, el cargo y la bio también se traducen, en una carpeta del idioma dentro de Autores. Mira [Sitios en varios idiomas](Sitios%20en%20varios%20idiomas.md).
