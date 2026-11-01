---
title: Feedback
linkTitle: Feedback
blup: feedback
weight: 6
description: Cuéntanos qué falta, qué se rompió y qué puede mejorar. Vota las solicitudes de otras personas.
examples:
  - title: "Modo oscuro automático"
    kind: feature
    status: planned
    votes: 42
    days: 2
    body: "Seguir el tema del sistema, sin cambiarlo a mano cada noche."
    replies:
      - author: marina
        days: 1
        text: "Sería genial, de noche la pantalla clara cansa bastante."
      - author: rafael
        days: 1
        text: "Un voto más. ¿Puede seguir el horario del sistema también?"
  - title: "La búsqueda no encuentra palabras con tilde"
    kind: bug
    status: progress
    votes: 17
    days: 9
    body: "Buscar \"configuración\" no trae nada, pero \"configuracion\" sí."
    replies:
      - author: julia
        days: 6
        text: "Me pasa también, con \"acción\" y \"está\"."
      - author: equipo
        days: 2
        text: "Ya encontramos la causa y la corrección sale en la próxima versión."
  - title: "Atajo para crear una nota nueva"
    kind: improvement
    status: new
    votes: 11
    days: 15
    body: "Un atajo de teclado ayudaría mucho a quien escribe todo el día."
    replies:
      - author: pedro
        days: 10
        text: "Ctrl+N sería lo más natural."
  - title: "Exportar a PDF"
    kind: feature
    status: done
    votes: 29
    days: 22
    body: "Poder descargar una página entera en PDF para enviarla por correo."
    replies:
      - author: carla
        days: 20
        text: "Lo usé hoy para enviar un informe, quedó perfecto. ¡Gracias!"
  - title: "Versión para Windows XP"
    kind: other
    status: declined
    votes: 3
    days: 40
    body: "Todavía uso una computadora vieja en la oficina."
    replies:
      - author: equipo
        days: 38
        text: "Windows XP ya no recibe actualizaciones de seguridad, así que no lo vamos a soportar."
---
%%
Esta carpeta es el tablero de feedback del sitio: quien visita abre solicitudes (error, función, mejora u otra), vota y comenta, entrando con GitHub. Las solicitudes quedan en las issues del repositorio de comentarios del sitio, no aquí. El estado de cada solicitud (planificado, en curso, hecho, rechazado) lo cambias en Blup Studio, en la pantalla Feedback.

Las solicitudes de la lista examples son ejemplos, como las notas de las otras secciones: aparecen en el tablero hasta que llega la primera solicitud real. Puedes borrar o cambiar la lista.
%%
