---
title: Studio u Obsidian
description: Los dos abren las mismas notas. El Studio hace todo el camino, del texto al sitio en línea.
weight: 4
---
Las notas de Blup son **archivos Markdown comunes** en una carpeta de tu computadora. Por eso la misma carpeta se abre en Blup Studio y en Obsidian, y puedes alternar cuando quieras.

## La forma recomendada: Blup Studio

El Studio reúne todo en una sola app: escribir, ver el sitio, configurar y publicar. Para quien está empezando, es el camino más corto.

## También vale: Obsidian y la línea de comandos

Si ya te gusta Obsidian, sigue con él. Escribe las notas allí y usa `blup` en la terminal para ver la vista previa (`blup preview`), publicar (`blup publish`) y cambiar la configuración (`blup customize`). Mira [Comandos](../Línea%20de%20comandos/Comandos.md).

> [!tip] Usa los dos
> Mucha gente escribe en Obsidian y abre el Studio solo para ver la vista previa, configurar y publicar. El Studio se da cuenta cuando una nota cambia fuera de él y actualiza la pantalla.

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
- Generar el sitio, ver la vista previa y publicar en Cloudflare Pages o en GitHub Pages.
- Sincronizar las notas con un repositorio privado de GitHub.
- Contraseñas, páginas ocultas, borradores, cookies, comentarios y reacciones.

## Enlaces entre notas

Los sitios entienden enlaces Markdown con ruta relativa, como `[Instalación](Instalación.md)`. El Studio siempre guarda los enlaces así, incluso cuando escribes `[[`. En Obsidian, acepta el ajuste que Blup ofrece al crear el sitio (activa **Usar enlaces Markdown** y **ruta relativa**). Las notas antiguas con wikilinks se pueden convertir con `blup convert`. Mira [Enlaces e imágenes](../Escribiendo/Enlaces%20e%20imágenes.md).
