---
title: Studio y línea de comandos
description: Dos formas de usar Blup con las mismas notas. El Studio hace todo el camino, del texto al sitio en línea.
weight: 4
---
Las notas de Blup son **archivos Markdown comunes** en una carpeta de tu computadora. Puedes trabajar con ellas en Blup Studio o con la línea de comandos `blup`, y alternar cuando quieras.

## La forma recomendada: Blup Studio

El Studio reúne todo en una sola app: escribir, ver el sitio, configurar y publicar. Para quien está empezando, es el camino más corto.

## También vale: la línea de comandos

`blup` en la terminal genera la vista previa (`blup preview`), publica (`blup publish`) y cambia la configuración (`blup customize`). Sirve para quien prefiere la terminal o quiere automatizar. Mira [Comandos](../Línea%20de%20comandos/Comandos.md).

> [!tip] Notas editadas fuera del Studio
> El Studio se da cuenta cuando una nota cambia fuera de él y actualiza la pantalla.

## Lo que existe solo en el Studio

| Función | Dónde leer |
| --- | --- |
| Editor con vista previa en vivo del sitio, lado a lado | [Vista previa en vivo](../Blup%20Studio/Vista%20previa%20en%20vivo.md) |
| Pantallas Sitio, Secciones, Autores, Idiomas, Comentarios, Cookies, Podcast y Publicar | [Ajustes del sitio](../Configuración%20del%20sitio/Ajustes%20del%20sitio.md) |
| Submenús en el menú superior, arrastrando | [Menú, submenús y botón](../Configuración%20del%20sitio/Menú,%20submenús%20y%20botón.md) |
| Traducción de las notas a otros idiomas | [Traducción](../Funciones%20con%20IA/Traducción.md) |
| Narración de los posts en audio | [Narración de los posts](../Funciones%20con%20IA/Narración%20de%20los%20posts.md) |
| Tablero de Feedback en Bento | [Feedback](../Secciones%20del%20Bento/Feedback.md) |
| Bóvedas de notas con nota del día, mapa de notas, lienzos, dibujos y bases de datos | [Qué es una bóveda](../Bóvedas%20de%20notas/Qué%20es%20una%20bóveda.md) |
| Sincronización y publicación con Blup Cloud | [Sincronizar las notas](../Blup%20Cloud/Sincronizar%20las%20notas.md) |
| Terminal con agentes de IA y deshacer lo que cambió la IA | [Terminal y agentes de IA](../Blup%20Studio/Terminal%20y%20agentes%20de%20IA.md) |
| Papelera, historial de versiones y copia diaria de las notas | [Papelera e historial de versiones](../Blup%20Studio/Papelera%20e%20historial.md) |

## Lo que funciona en los dos

- Escribir en Markdown, con propiedades, callouts, tablas, fórmulas y comentarios `%%`.
- Generar el sitio, ver la vista previa y publicar en Cloudflare Workers o en GitHub Pages.
- Sincronizar las notas con un repositorio privado de GitHub.
- Contraseñas, páginas ocultas, borradores, cookies, comentarios y reacciones.

## Enlaces entre notas

Los sitios entienden enlaces Markdown con ruta relativa, como `[Instalación](Instalación.md)`. El Studio siempre guarda los enlaces así, incluso cuando escribes `[[`. Las notas antiguas con wikilinks se pueden convertir con `blup convert`. Mira [Enlaces e imágenes](../Escribiendo/Enlaces%20e%20imágenes.md).
