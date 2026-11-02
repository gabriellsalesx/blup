---
title: ¿Cómo sincronizo las notas entre computadoras?
description: Con Blup Cloud o con un repositorio privado de GitHub.
weight: 3
---
1. En el Studio, haz clic en **Sincronizar**, arriba.
2. Elige el destino:
   - **Blup Cloud**: inicia sesión en tu cuenta y crea la contraseña de las notas.
   - **GitHub**: las notas van a un repositorio privado tuyo. No necesitas saber git.
3. En la otra computadora, abre el Studio e inicia sesión en la misma cuenta. Con Blup Cloud, después de escribir la contraseña de las notas, el Studio muestra lo que está guardado en la nube y trae lo que elijas. Con GitHub, conecta la misma cuenta y el Studio encuentra los sitios guardados allí.

En el plan Gratis, la sincronización ocurre cuando haces clic en el botón. Del Pro en adelante, es en tiempo real.

## En la línea de comandos

Ejecuta `blup sync` (o `blup sincronizar`) para sincronizar con GitHub.

> [!info] ¿Editaste la misma nota en dos lugares?
> Blup guarda las dos versiones. Mira [¿Por qué apareció una nota con "conflict" en el nombre?](../Problemas%20comunes/Nota%20con%20conflict%20en%20el%20nombre.md)
