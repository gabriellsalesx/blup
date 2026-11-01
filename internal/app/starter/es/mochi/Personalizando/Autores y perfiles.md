---
title: Autores y perfiles
date: {{date-7}}
description: Una nota por autor, con foto, cargo, redes sociales y biografía. Posts con más de un autor.
tags: [Personalización]
image: "[[blup-autores.jpg]]"
authors: [Blup]
---
Este post lo "escribió" **Blup**, la mascota. Fíjate en la foto y el nombre de arriba: vienen de su nota, en la carpeta **Autores**.

## La carpeta de autores

La carpeta **Autores** tiene una nota **_index** con la propiedad `blup: authors`. Le avisa a Blup que las notas de ahí son perfiles, no posts: la carpeta no se vuelve página ni aparece en el menú.

Cada autor es una nota con el nombre de la persona:

```yaml
---
photo: "[[ana.jpg]]"
role: Editora
social:
  - https://instagram.com/ana
  - https://github.com/ana
---
Ana escribe sobre libros y viajes.
```

- **photo**: la foto del perfil.
- **role**: el cargo o función.
- **social**: enlaces a los perfiles, con ícono automático.
- El **texto de la nota** es la biografía.

Cuando creaste el sitio, Blup ya creó tu nota en esta carpeta. Completa la foto y la biografía.

## En los posts

```yaml
authors: [Ana]
authors: [Ana, Bruno]
```

Sin la propiedad, el post es del autor del sitio. Con más de un autor, las fotos aparecen una al lado de la otra y los nombres juntos.

## La página de cada autor

Haz clic en el nombre del autor de un post: su página muestra foto, cargo, biografía, redes sociales y todos los posts que escribió.
