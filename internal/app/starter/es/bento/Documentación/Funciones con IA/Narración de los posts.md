---
title: Narración de los posts
description: Una voz lee el post y el sitio muestra un reproductor arriba. Gratis en la computadora o con la nube.
weight: 2
---
La narración convierte el texto de un post en un audio MP3. En el sitio, el post recibe un **reproductor** arriba, para quien prefiere escuchar.

> [!info] Solo en Blup Studio
> La narración se crea en el Studio. Una vez lista, es un archivo común en Adjuntos y se publica por cualquier camino.

## Elegir la voz (una vez)

1. Abre **Configuración > IA y voz > Narración**.
2. Para cada idioma (Português, English, Español), elige la voz. Vienen en grupos:
   - **En la computadora, gratis**: las voces Kokoro. En portugués: Dora, Alex y Santa. La primera vez, descarga la voz en **Configuración > IA y voz > IA**, botón **Descargar voz local**.
   - **Blup Cloud, incluida en el plan**: necesita un plan de pago.
   - **OpenRouter, de pago**: necesita tu clave.
3. **Reproducir muestra** toca un fragmento de la voz.

La voz sirve para todos los sitios. Mira las formas de pagar en [IA en Blup Studio](IA%20en%20Blup%20Studio.md).

## Narrar un post

1. Abre el post en el Studio.
2. En las **Propiedades**, en el bloque **Narración**, haz clic en **Generar narración**.
3. El Studio muestra el avance: preparando el texto, generando la voz parte por parte, guardando el audio.
4. Listo: aparecen la duración y el tamaño. Escúchala en el reproductor de la vista previa, al lado.

El audio se guarda en **Adjuntos**, y la nota recibe las propiedades `audio`, `audio_hash` y `audio_seconds`.

## Lo que lee la voz

El título, la descripción y el texto. Se salta lo que no tiene sentido escuchar: bloques de código, fórmulas, imágenes, tablas y comentarios `%%`. Los títulos y los elementos de lista llevan pausas.

## Cuando cambia el texto

El Studio compara el texto con el de la narración:

- **Narración al día con el texto**: todo bien.
- **El texto cambió después de la narración**: haz clic en **Generarla de nuevo** para que el audio lo acompañe.

## Posts en otros idiomas

La traducción de un post (en la carpeta `en`, `es` o `pt`) usa la voz de ese idioma y recibe su propio audio, como `Post (en).mp3`. Mira [Sitios en varios idiomas](../Configuración%20del%20sitio/Sitios%20en%20varios%20idiomas.md).

> [!warning] Los posts con contraseña no se narran
> Un audio público revelaría el texto protegido. Por eso el Studio se niega a narrar posts con contraseña.

> [!tip] Conviértelo en podcast
> En los blogs Mochi y Nori, las narraciones pueden volverse episodios de un podcast. Mira [Podcast de los blogs](Podcast%20de%20los%20blogs.md).
