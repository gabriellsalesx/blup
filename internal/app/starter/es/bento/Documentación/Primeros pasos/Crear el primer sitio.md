---
title: Crear el primer sitio
description: Un paso a la vez, en el Studio o en la terminal. Puedes cambiarlo todo después.
weight: 2
---
## Con Blup Studio

En la pantalla de inicio, haz clic en **Crear un sitio nuevo**. El Studio muestra una pregunta a la vez, con **Continuar** y **Atrás** abajo.

1. **¿Qué tipo de sitio quieres hacer?**
   - **Blog personal**: blog, diario, newsletter o portafolio de textos.
   - **Sitio de producto**: para presentar una app, SaaS, API o proyecto, con documentación y novedades.
2. **¿Qué tema va con tu sitio?**
   - Blog: **Mochi** (post destacado y cuadrícula de posts) o **Nori** (una sola columna, con el texto en primer plano).
   - Producto: **Bento**, este mismo.
3. **¿Qué partes tendrá el sitio?** (solo en Bento): Documentación, Roadmap, Changelog, Centro de ayuda y Blog. Cada parte trae notas de ejemplo que enseñan a usarla. Elige al menos una.
4. **¿Cómo se llama el sitio?** El nombre, una descripción opcional, tu nombre (firma los posts) y el **Idioma del sitio**.
5. **¿Dónde viven las notas?**
   - **En una carpeta nueva**: la forma más simple.
   - **En una carpeta que yo elija**: una carpeta que ya tienes, con notas en Markdown escritas en cualquier editor.
6. **¿Todo listo?** Revisa el resumen y haz clic en **Crear el sitio**.

Listo: haz clic en **Abrir el sitio**. El editor se abre con la vista previa al lado. Mira [La pantalla del Studio](../Blup%20Studio/La%20pantalla%20del%20Studio.md).

> [!tip] Ajustar la carpeta
> Si eliges una carpeta que ya tiene notas, el Studio ofrece **Ajustar la carpeta para que los enlaces funcionen en el sitio**. Así los enlaces nuevos entre notas se guardan en el formato que entiende la web. Blup guarda antes una copia de la configuración.

> [!note] Dónde quedan los archivos del sitio
> En **Dónde guardar los archivos del sitio**, eliges la carpeta de la configuración y de las páginas generadas. Queda fuera de las notas y no necesitas abrirla.

### ¿Ya tienes un sitio de Blup?

En la pantalla de inicio, **Ya tengo un sitio de Blup** abre una carpeta que ya es un sitio, muestra los **Abiertos recientemente** y, con la cuenta de GitHub conectada, ofrece **Traer** los sitios guardados allí. Con la cuenta de Blup Cloud, el Studio también muestra tus sitios guardados en la nube.

## Con la línea de comandos

Ejecuta `blup new` (o solo `blup`, la primera vez). Cada pregunta aparece sola y ya trae una respuesta sugerida. Si tienes dudas, presiona Enter.

1. **Bóveda de Obsidian**: Blup muestra las bóvedas que encontró.
2. **Carpeta**: la bóveda entera o solo una carpeta, como "Sitio".
3. **Tema**: **Mochi** (blog) o **Bento** (sitio de producto).
4. **Secciones** (solo en Bento): marca las que quieras.
5. **Nombre del sitio**, **una frase sobre él** y **tu nombre**.
6. **Idioma del sitio** y **dónde guardar** los archivos del sitio en la computadora.
7. **Resumen** para confirmar.
8. **Ajuste de los enlaces**: enlaces Markdown con ruta relativa.
9. **Guardar las notas en GitHub**: opcional. Mira [Sincronizar](../Línea%20de%20comandos/Sincronizar.md).
10. **Abrir la vista previa**.

La línea de comandos también crea la carpeta **Plantillas** con plantillas de nota para cada sección.

## Qué se crea

- **En las notas**: este sitio de ejemplo (si la carpeta está vacía), tu nota de autor y las carpetas de las secciones elegidas.
- **En una carpeta aparte**: el **blup.toml** y los archivos que usa el generador del sitio (Hugo).

> [!tip] ¿Ya tenías notas en la carpeta?
> Nada se elimina ni se sobrescribe. Blup solo crea la nota **_index** si falta y las carpetas de las secciones elegidas.
