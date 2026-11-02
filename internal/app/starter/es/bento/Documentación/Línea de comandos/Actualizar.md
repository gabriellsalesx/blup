---
title: Actualizar
description: Blup avisa cuando hay una versión nueva y solo se actualiza si tú quieres.
weight: 5
---
Una vez al día, al abrir el menú, Blup revisa si hay una versión nueva. Si la hay, muestra qué cambió y pregunta:

- **Actualizar ahora**
- **Ahora no**
- **No recordarme esta versión**

Para actualizar en cualquier momento: `blup update` (o `blup atualizar`).

> [!note] ¿Y Blup Studio?
> Este comando actualiza solo la línea de comandos. El Studio busca una versión nueva en **Configuración > Acerca de**. Mira [Instalación](../Primeros%20pasos/Instalación.md).

## Volver atrás

```sh
blup update --rollback
```

Vuelve a la versión anterior, guardada durante la actualización.

## Los temas

Los temas vienen dentro de Blup: después de actualizar, los sitios reciben el tema nuevo en la próxima vista previa o publicación. Tus notas no cambian.

> [!info] ¿Sin internet o con prisa?
> La revisión se rinde en 2 segundos y nunca impide que Blup abra.
