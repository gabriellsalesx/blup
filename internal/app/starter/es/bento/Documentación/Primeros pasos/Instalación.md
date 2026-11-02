---
title: Instalación
description: Descarga Blup Studio en getblup.io. La línea de comandos es opcional.
weight: 1
---
Hay dos formas de usar Blup, y conviven bien:

- **Blup Studio** (recomendado): una app para la computadora, con editor, vista previa en vivo, pantallas de configuración y publicación con un botón.
- **Línea de comandos** `blup` (opcional): para quien prefiere la terminal.

## Requisitos

- Linux, macOS o Windows.
- Nada más. Blup descarga solo lo que necesita para generar el sitio.

## Instalar Blup Studio

1. Abre **getblup.io** en el navegador.
2. Descarga la versión de tu sistema (Linux, macOS o Windows).
3. Instálala como cualquier otra app y abre **Blup Studio**.

La primera vez, el Studio te da la bienvenida, pregunta el idioma y ofrece iniciar sesión en tu cuenta de Blup Cloud. La cuenta es **opcional**: todo funciona en esta computadora sin ella. Después solo sigue a [Crear el primer sitio](Crear%20el%20primer%20sitio.md).

> [!info] Actualizaciones del Studio
> En **Configuración > Acerca de**, el Studio busca una versión nueva una vez al día (o cuando haces clic en **Buscar actualización**). Cuando la hay, aparece un punto azul al pie de la configuración y el botón **Descargar**.

## Instalar la línea de comandos (opcional)

Solo la necesitas si vas a usar `blup` en la terminal. El Studio no depende de ella.

### macOS y Linux

Abre la Terminal y pega:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

### Windows

Abre PowerShell y pega:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

### Comprobar

Ejecuta `blup`. La primera vez te pregunta el idioma (inglés, portugués o español) y abre el asistente de sitio nuevo.

> [!info] Actualizaciones de la línea de comandos
> `blup` revisa una vez al día si hay una versión nueva y siempre pregunta antes de actualizar. Mira [Actualizar](../Línea%20de%20comandos/Actualizar.md).
