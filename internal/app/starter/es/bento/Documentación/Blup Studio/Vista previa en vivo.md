---
title: Vista previa en vivo
description: El sitio de verdad junto al editor, actualizado mientras escribes.
weight: 3
---
Al abrir un sitio, el Studio activa la vista previa junto al editor. Es tu sitio, generado en la computadora, igual al que se publica.

## Cómo funciona

- **Lo escribes, aparece**: cada nota guardada actualiza la vista previa sola, sin parpadear y manteniendo la posición de la página.
- **La vista previa sigue a la nota**: si abres otra nota en el editor, la vista previa va a su página.
- **Y al revés también**: si haces clic en un enlace dentro de la vista previa, la nota de esa página se abre en el editor.
- **Configuración al instante**: el nombre, el logo, el menú y las redes guardados en la pantalla Sitio aparecen en la vista previa sin recargar.
- **Los borradores aparecen** en la vista previa (y nunca en el sitio publicado).
- Si la nota tiene narración, el reproductor aparece arriba del post.

## Activar y desactivar

- El botón **Vista previa** de arriba, o `Ctrl+Alt+P`, la activa y la desactiva.
- En **Configuración > Editor**, **Abrir con la vista previa activada** decide si la vista previa empieza activada.
- Arrastra la división entre el editor y la vista previa para cambiar los anchos. El Studio lo recuerda.

> [!info] Solo en la computadora
> La vista previa funciona solo en tu computadora. Nadie más la ve, y nada va a internet hasta que publicas.

> [!note] ¿Sin reproductor de audio en Linux?
> En algunos Linux falta un componente de audio del sistema (GStreamer). En ese caso la vista previa oculta el reproductor, pero el sitio publicado lo muestra normalmente.

## Desde Obsidian

Sin el Studio, ejecuta `blup preview` en la terminal (en portugués, el alias es `blup ver`): la vista previa se abre en el navegador y se actualiza con cada nota guardada. Consulta [Vista previa](../Línea%20de%20comandos/Vista%20previa.md).
