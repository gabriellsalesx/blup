---
title: Sincronizar
description: Las notas del sitio en un repositorio privado de GitHub, para usarlas en otra computadora. Sin git.
weight: 4
---
`blup sync` (o `blup sincronizar`) guarda una copia de las notas de la carpeta del sitio en un **repositorio privado** de GitHub. En otra computadora, instala Blup y sincroniza: las notas llegan. Edita donde quieras y vuelve a sincronizar.

![Dos computadoras sincronizando](../../Adjuntos/blup-sincronizar.jpg)

> [!info] ¿Y Blup Cloud?
> La sincronización cifrada con Blup Cloud se hace en Blup Studio. Mira [Sincronizar las notas](../Blup%20Cloud/Sincronizar%20las%20notas.md).

## La primera vez

1. Ejecuta `blup sync`, elígelo en el menú o responde "sí" al final del asistente de sitio nuevo.
2. Usa la misma clave de GitHub de la publicación.
3. Elige el nombre del repositorio. Se crea **privado**. Si el repositorio ya existe y es público, Blup te avisa antes.

## Cada vez

Blup compara y **muestra antes** cuántas notas va a enviar, recibir, eliminar en GitHub y mover a la papelera aquí. Solo sigue si confirmas.

| Situación | Qué pasa |
| --- | --- |
| Cambió solo aquí | Se envía |
| Cambió solo en otra computadora | Se recibe |
| Eliminada en un lado | Se elimina en el otro |
| Eliminada en un lado y editada en el otro | Se queda la versión editada |
| Editada en los dos lados | Se quedan las dos: la tuya y una copia con "conflict" en el nombre |

## Seguridad

- Las notas eliminadas en otra computadora van a una **papelera fuera de las notas**, en la carpeta del sitio; Blup muestra dónde.
- Si la carpeta del sitio aparece vacía pero ya tenía notas, Blup **se detiene** y no elimina nada en GitHub.
- Las carpetas que empiezan con punto no se sincronizan.
- Los archivos de más de 100 MB quedan fuera, con un aviso.
