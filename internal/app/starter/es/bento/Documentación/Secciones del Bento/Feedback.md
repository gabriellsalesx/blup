---
title: Feedback
description: Un tablero donde quien visita pide funciones, reporta errores y vota. Solo en Blup Studio.
weight: 8
---
El **Feedback** es un tablero de solicitudes: quien visita el sitio sugiere funciones, reporta errores, propone mejoras y vota por las solicitudes de los demás. Tú lo sigues todo y cambias el estado desde el Studio.

> [!info] Solo en Blup Studio
> La sección Feedback se agrega desde la pantalla **Secciones** del Studio, y las solicitudes se administran en la pantalla **Feedback**. Necesita los [comentarios](Comentarios%20y%20reacciones.md) activados.

## Agregar

1. Activa los comentarios, si todavía no lo hiciste. Mira [Comentarios y reacciones](Comentarios%20y%20reacciones.md).
2. En la pantalla **Secciones**, haz clic en **Agregar** junto a **Feedback**.
3. El Studio crea la carpeta **Feedback**, con algunas solicitudes de ejemplo.
4. Publica el sitio.

## El tablero en el sitio

| Control | Qué hace |
| --- | --- |
| Categorías | Todo el feedback, Función, Error, Mejora y Otro, con la cantidad de cada una |
| Estado | Todos los estados, los activos (Nuevo, Planeado, En curso) o un estado |
| Más votados y Recientes | El orden de la lista |
| Búsqueda | Busca en el título y en el texto |
| Nueva solicitud | Tipo, título y descripción. La persona inicia sesión con GitHub para enviarla |
| Cargar más solicitudes | Muestra diez más |

Haz clic en una solicitud para ver el texto, los votos y los comentarios.

## Las solicitudes de ejemplo

La nota **_index** de la carpeta Feedback guarda algunas solicitudes de ejemplo, en la propiedad `examples`. Muestran cómo queda el tablero antes de que llegue la primera solicitud de verdad. Cuando llega la primera solicitud real, los ejemplos desaparecen del tablero.

Puedes editar o eliminar los ejemplos cuando quieras. En un sitio en varios idiomas, sus textos también se traducen.

## Responder y cambiar el estado

Haz clic en **Feedback** en la barra lateral del Studio.

- La pantalla muestra las solicitudes reales, de las más votadas a las menos, con tipo, autor, comentarios, votos e idioma.
- Filtra por categoría.
- Haz clic en el título para abrirla en GitHub, donde respondes.
- Cambia el **Estado**: Nuevo, Planeado, En curso, Hecho o Rechazado. Hecho y Rechazado cierran la solicitud; los demás la reabren.
- **Actualizar** vuelve a leer.

El sitio muestra el nuevo estado en hasta un minuto, sin volver a publicar.

> [!tip] Dónde quedan las solicitudes
> Cada solicitud es una issue en el repositorio de comentarios, en tu GitHub. No existe una nota por solicitud.
