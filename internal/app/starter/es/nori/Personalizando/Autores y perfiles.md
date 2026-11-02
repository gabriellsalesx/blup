---
title: Autores y perfiles
date: {{date-7}}
description: Un perfil por autor, con foto, cargo, redes y bio. Posts con más de un autor.
tags: [Personalización, Studio]
image: "[[blup-autores.jpg]]"
authors: [Blup]
---
Este post lo "escribió" **Blup**, la mascota. Fíjate en la foto y el nombre arriba: vienen de su perfil, en la carpeta **Autores**.

## En el Studio

La pantalla **Autores**, en el menú lateral, muestra los perfiles. En cada uno:

- **Foto**: una imagen tuya o uno de los avatares listos.
- **Cargo**: la función, como "Editora".
- **Redes sociales**: elige la red y completa la dirección del perfil.
- **Bio**: un texto corto sobre la persona.

Cuando creaste el sitio, Blup ya creó tu perfil. Completa la foto y la bio.

Para decir quién escribió un post, usa el campo **Autores** en el panel **Propiedades** del post. Empieza a escribir y elige el nombre.

## En el archivo de la nota

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

El texto de la nota es la bio. En los posts:

```yaml
authors: [Ana]
authors: [Ana, Bruno]
```

## Cómo se ve

Sin autor, el post es del autor del sitio. Con más de uno, las fotos aparecen una al lado de la otra y los nombres juntos.

Haz clic en el nombre del autor de un post: su página muestra la foto, el cargo, la bio, las redes y todos los posts que escribió. En un blog en varios idiomas, el cargo y la bio también se traducen.
