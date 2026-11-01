---
title: Instalar y crear el sitio
date: {{date-1}}
description: Un comando para instalar y un asistente que pregunta una cosa a la vez.
tags: [Primeros pasos, CLI]
image: "[[blup-terminal.jpg]]"
---
Blup es un programa de terminal, pero no necesitas memorizar nada: muestra un menú y hace una pregunta a la vez.

## Instalar

En macOS o Linux, abre la Terminal y pega:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

En Windows, abre PowerShell y pega:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

Después, ejecuta `blup`. La primera vez pregunta el idioma y abre el asistente de sitio nuevo.

## El asistente

Cada pregunta ya trae una respuesta sugerida; si tienes dudas, presiona Enter.

1. **Qué bóveda de Obsidian** se convierte en sitio.
2. **Qué carpeta** de ella: toda la bóveda o solo una carpeta, como "Blog".
3. **Tema**: Mochi (blog personal, este mismo) o Bento (sitio de producto, con documentación, roadmap, changelog, centro de ayuda y blog).
4. **Nombre del sitio**, **una frase sobre él** y **tu nombre**.
5. **Idioma del sitio** y **dónde guardar** los archivos del sitio en la computadora.
6. **Ajustes en Obsidian**: Blup pide usar enlaces Markdown con ruta relativa, el formato que entienden los sitios.
7. **Guardar las notas en GitHub**, si quieres usarlas en otra computadora.
8. **Abrir la vista previa**.

> [!info] Dónde queda cada cosa
> Las notas se quedan en tu bóveda, como siempre. En una carpeta aparte, Blup guarda el archivo **blup.toml** (dónde publicar, contraseña general) y los archivos que Hugo usa para generar el sitio. No necesitas abrir esa carpeta.

## Los comandos

Ejecutar solo `blup` abre el menú con todo. Si prefieres, cada acción tiene un comando:

| Comando | Qué hace |
| --- | --- |
| `blup new` | Crea un sitio nuevo |
| `blup preview` | Abre la vista previa en el navegador |
| `blup publish` | Pone el sitio en línea |
| `blup customize` | Menú, redes sociales, logo, favicon y autor |
| `blup sync` | Guarda las notas en un repositorio privado de GitHub |
| `blup convert` | Cambia los `[[wikilinks]]` por enlaces estándar |
| `blup build` | Genera el sitio en la carpeta public, sin preguntas |
| `blup language` | Cambia el idioma de Blup |
| `blup update` | Actualiza Blup |

Los mismos comandos también funcionan en portugués: `novo`, `ver`, `publicar`, `personalizar`, `sincronizar`, `converter`, `gerar`, `idioma` y `atualizar`.

## Actualizaciones

Blup revisa una vez al día si hay una versión nueva y **siempre pregunta** antes de actualizar. Si algo sale mal, `blup update --rollback` vuelve a la versión anterior.
