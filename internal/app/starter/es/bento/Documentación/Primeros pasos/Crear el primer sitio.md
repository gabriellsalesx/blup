---
title: Crear el primer sitio
description: El asistente pregunta una cosa a la vez. Si tienes dudas, presiona Enter.
weight: 2
---
Ejecuta `blup new` (o solo `blup`, la primera vez). Cada pregunta aparece sola y ya trae una respuesta sugerida.

## Las preguntas

1. **Bóveda de Obsidian**: Blup muestra las bóvedas que encontró.
2. **Carpeta**: toda la bóveda o solo una carpeta, como "Sitio".
3. **Tema**:
   - **Mochi**: blog personal, con posts, etiquetas y autores.
   - **Bento**: sitio de producto, como este.
4. **Secciones** (solo en Bento): Documentación, Roadmap, Changelog, Centro de ayuda y Blog. Marca las que quieras; puedes agregarlas después.
5. **Nombre del sitio**, **una frase sobre él** y **tu nombre**.
6. **Idioma del sitio** y **dónde guardar** los archivos del sitio en la computadora.
7. **Resumen** para confirmar.
8. **Ajuste en Obsidian**: enlaces Markdown con ruta relativa, el formato que entienden los sitios. Blup guarda una copia de la configuración anterior.
9. **Guardar las notas en GitHub**: opcional, para usarlas en otra computadora. Mira [Sincronizar](../CLI/Sincronizar.md).
10. **Abrir la vista previa**.

## Qué crea el asistente

- En la bóveda: este tutorial (si la carpeta está vacía), tu nota de autor y la carpeta **Templates** con plantillas de nota.
- En una carpeta aparte: el **blup.toml** y los archivos que usa el generador del sitio (Hugo). No necesitas abrir esa carpeta.

> [!tip] ¿Ya tenías notas en la carpeta?
> Nada se borra ni se sobrescribe. Blup solo crea la nota **_index** si falta y las carpetas de las secciones elegidas.
