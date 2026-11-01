---
title: Sincronizar las notas
date: {{date-8}}
description: Tus notas en un repositorio privado de GitHub, para usarlas en otra computadora. Sin git.
tags: [Personalización, CLI]
image: "[[blup-sincronizar.jpg]]"
---
`blup sync` guarda una copia de las notas del sitio en un **repositorio privado** de GitHub. En otra computadora, instala Blup y sincroniza: las notas llegan. Edita donde quieras y sincroniza de nuevo.

No necesitas saber git ni instalar nada además de Blup.

## La primera vez

1. Ejecuta `blup sync` (o elígelo en el menú, o responde "sí" al final del asistente de sitio nuevo).
2. Entra con la misma clave de GitHub que usas para publicar.
3. Elige el nombre del repositorio. Se crea **privado**: solo tú lo ves.

## Cada vez

Blup compara las notas de aquí con las de GitHub y **te muestra lo que va a hacer** antes de hacerlo:

- lo que cambió aquí se envía;
- lo que cambió en otra computadora se recibe;
- lo que se borró de un lado se borra del otro.

## Cuando cambiaron los dos lados

Si la misma nota se editó aquí y en otra computadora, nada se pierde: tu versión se queda, y la otra se vuelve una copia con "conflict" en el nombre. Compara las dos y borra la que sobre.

> [!info] Nada desaparece de repente
> Las notas borradas en otra computadora van a una papelera fuera de la bóveda, y Blup te muestra dónde está. Si la carpeta del sitio aparece vacía de repente (por ejemplo, si la bóveda cambió de lugar), Blup se detiene y no borra nada en GitHub.
