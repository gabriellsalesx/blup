---
title: Instalar y crear el sitio
date: {{date-1}}
description: Descarga Blup Studio y crea el sitio respondiendo una pregunta a la vez. O usa la línea de comandos.
tags: [Primeros pasos, Studio]
image: "[[blup-dicas.jpg]]"
---
## Instalar Blup Studio

**Blup Studio** se descarga en **getblup.io**, para Linux, macOS y Windows. Instálalo como cualquier app y ábrelo.

La primera vez, el Studio te da la bienvenida y hace algunas preguntas rápidas: el idioma, si quieres conectar tu cuenta de GitHub ahora y una contraseña general para los posts con contraseña. Puedes saltar lo que quieras y volver después, en **Configuración** (el engranaje, abajo en el menú lateral).

## Crear un sitio

En la pantalla de inicio, haz clic en **Crear un sitio nuevo**. El Studio hace una pregunta a la vez, y puedes cambiar todo después.

1. **Qué tipo de sitio**: blog personal o sitio de producto.
2. **Qué tema**: Nori (este, de una sola columna, con el texto en primer plano) o Mochi, para el blog; Bento, para el sitio de producto.
3. **Cómo se llama el sitio**: nombre, una frase sobre él, tu nombre y el idioma del sitio.
4. **Dónde viven las notas**: en una carpeta nueva (lo más simple) o en una carpeta que elijas, que puede tener notas.
5. **¿Todo listo?** Revisa y crea el sitio.

Si la carpeta está vacía, Blup pone el sitio de ejemplo del tema: este tutorial.

> [!tip] ¿Ya tienes una carpeta con notas?
> Si la eliges, deja marcado **Ajustar la carpeta para que los enlaces funcionen en el sitio**. Así los enlaces nuevos se guardan en el formato que el sitio entiende.

## La ventana del sitio

- **Archivos**, a la izquierda: las notas y carpetas. Usa **Nueva nota** y **Nueva carpeta** para crear, y la búsqueda para encontrar una nota.
- **El editor**, en el medio: donde escribes. Encima está el panel **Propiedades** de la nota.
- **Vista previa**, a la derecha: el sitio de verdad, que se actualiza mientras escribes.
- **El menú lateral**: Escribir, Adjuntos, Autores, Sitio, Secciones, Idiomas, Comentarios, Cookies y estadísticas, Publicar y Papelera, además del terminal.

No hay botón de guardar: el Studio guarda solo mientras escribes.

Haz clic derecho en una nota para crear otra ahí, hacer una copia, moverla, copiar el enlace, renombrarla, exportarla o eliminarla. **Renombrar** actualiza solo los enlaces que apuntan a la nota, y **Eliminar** manda la nota a la **Papelera**, desde donde puedes restaurarla.

## Por la línea de comandos

Blup también tiene una línea de comandos, `blup`. Para instalarla:

En macOS o Linux, abre la Terminal y pega:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

En Windows, abre PowerShell y pega:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

Después, ejecuta `blup`. La primera vez pregunta el idioma y abre el asistente de sitio nuevo: qué bóveda de Obsidian, qué carpeta de ella, el tema (Mochi o Bento; Nori se crea en el Studio), el nombre del sitio y dónde guardarlo. Cada pregunta ya trae una respuesta sugerida; si tienes dudas, presiona Enter.

| Comando | Qué hace |
| --- | --- |
| `blup new` | Crea un sitio nuevo |
| `blup preview` | Abre la vista previa en el navegador |
| `blup publish` | Pone el sitio en línea |
| `blup customize` | Menú, redes, logo, comentarios, cookies y dominio |
| `blup sync` | Guarda las notas en un repositorio privado de GitHub |
| `blup convert` | Cambia los `[[wikilinks]]` por enlaces estándar |
| `blup update` | Actualiza Blup |

Los mismos comandos también funcionan en portugués: `novo`, `ver`, `publicar`, `personalizar`, `sincronizar`, `converter` y `atualizar`.

> [!info] El mismo sitio
> Un sitio creado en el Studio se abre en la línea de comandos, y al revés también. Las notas son las mismas.
