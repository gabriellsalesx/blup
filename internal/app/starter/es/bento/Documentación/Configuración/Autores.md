---
title: Autores
description: Una nota por persona, con foto, cargo, redes y biografía.
weight: 3
---
Los perfiles están en una carpeta cuya **_index** tiene `blup: authors`, aquí llamada **Autores**. Esa carpeta no se vuelve página: sus notas son solo los datos de los perfiles.

## El perfil

Una nota con el nombre de la persona, como `Ana Lima.md`:

```yaml
---
photo: "[[ana.jpg]]"
role: Diseñadora de producto
social:
  - https://github.com/ana
  - https://x.com/ana
---
Ana diseña el producto y escribe sobre cómo se hace.
```

- **photo**: la foto. Sin ella, aparecen las iniciales (o la mascota, para el autor del sitio).
- **role**: el cargo.
- **social**: los perfiles, con íconos automáticos.
- El **texto** de la nota es la biografía.

El asistente de sitio nuevo ya crea tu nota. `blup customize` > **Nombre, descripción y autor** edita nombre, cargo, biografía y foto.

## En los posts

```yaml
authors: [Ana Lima]
authors: [Ana Lima, Bruno Costa]
```

Sin **authors**, el post es del autor del sitio. Con más de uno, las fotos aparecen lado a lado y los nombres juntos ("Ana Lima y Bruno Costa").

Cada autor tiene una página con foto, cargo, biografía, redes y sus posts.
