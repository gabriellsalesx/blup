---
title: Instalación
description: Un comando en la terminal y Blup está listo.
weight: 1
---
## Requisitos

- [Obsidian](https://obsidian.md) con una bóveda (vault).
- macOS, Linux o Windows.
- Nada más. Blup descarga solo lo que necesita para generar el sitio.

## macOS y Linux

Abre la Terminal y pega:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

## Windows

Abre PowerShell y pega:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

## Comprobar

Ejecuta `blup`. La primera vez te pregunta el idioma y abre el asistente de sitio nuevo.

> [!info] Actualizaciones
> Blup revisa una vez al día si hay una versión nueva y siempre pregunta antes de actualizar. Mira [Actualizar](../CLI/Actualizar.md).
