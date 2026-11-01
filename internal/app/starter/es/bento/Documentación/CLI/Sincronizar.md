---
title: Sincronizar
description: Las notas del sitio en un repositorio privado de GitHub, para usarlas en otra computadora. Sin git.
weight: 4
---
`blup sync` guarda una copia de las notas de la carpeta del sitio en un **repositorio privado** de GitHub. En otra computadora, instala Blup y sincroniza: las notas llegan. Edita donde quieras y vuelve a sincronizar.

![Dos computadoras sincronizando](../../Adjuntos/blup-sincronizar.jpg)

## La primera vez

1. Ejecuta `blup sync`, elígelo en el menú o responde "sí" al final del asistente de sitio nuevo.
2. Usa la misma clave de GitHub de la publicación.
3. Elige el nombre del repositorio. Se crea **privado**. Si el repositorio ya existe y es público, Blup avisa antes.

## Cada vez

Blup compara y **muestra antes** cuántas notas va a enviar, recibir, borrar en GitHub y mover a la papelera aquí. Solo continúa si confirmas.

| Situación | Qué pasa |
| --- | --- |
| Cambió solo aquí | Se envía |
| Cambió solo en otra computadora | Se recibe |
| Borrada de un lado | Se borra del otro |
| Borrada de un lado y editada del otro | Queda la versión editada |
| Editada de los dos lados | Quedan las dos: la tuya y una copia con "conflict" en el nombre |

## Seguridad

- Las notas borradas en otra computadora van a una **papelera fuera de la bóveda**; Blup muestra dónde.
- Si la carpeta del sitio aparece vacía pero ya tenía notas, Blup **se detiene** y no borra nada en GitHub.
- Las carpetas que empiezan con punto (como `.obsidian`) no se sincronizan.
- Los archivos de más de 100 MB quedan fuera, con un aviso.
