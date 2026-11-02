---
title: Feedback
linkTitle: Feedback
blup: feedback
weight: 6
description: Cuéntanos qué falta, qué se rompió y qué se puede mejorar. Vota las solicitudes de otras personas.
examples:
  - title: "Más temas para elegir"
    kind: feature
    status: planned
    votes: 42
    days: 2
    body: "Me gusta mucho Bento, pero quisiera otras opciones de estilo para el sitio de mi producto."
    replies:
      - author: marina
        days: 1
        text: "Sería genial tener un tema con más aire de revista."
      - author: rafael
        days: 1
        text: "Un voto más. Y que se pueda cambiar sin perder las notas."
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
  - title: "Elegir el color de acento del sitio"
    kind: improvement
    status: new
    votes: 11
    days: 15
    body: "Quisiera usar el color de mi marca en los enlaces y botones, directo en la pantalla Sitio del Studio."
    replies:
      - author: pedro
        days: 10
        text: "Con una vista previa del color antes de guardar, ojalá."
  - title: "Narración de los posts en audio"
    kind: feature
    status: done
    votes: 29
    days: 22
    body: "Mucha gente lee en el autobús. Sería bueno poder escuchar los posts."
    replies:
      - author: carla
        days: 20
        text: "Hoy narré mis posts con la voz en la computadora. Quedó genial, ¡gracias!"
  - title: "Versión para Windows XP"
    kind: other
    status: declined
    votes: 3
    days: 40
    body: "Todavía uso una computadora vieja en la oficina."
    replies:
      - author: equipo
        days: 38
        text: "Lamentablemente, Windows XP ya no recibe actualizaciones de seguridad, así que no lo vamos a soportar."
---
%%
Esta carpeta es el tablero de feedback del sitio: quien visita abre solicitudes (error, función, mejora u otra), vota y comenta, entrando con GitHub. Las solicitudes quedan en las issues del repositorio de comentarios del sitio, no aquí. Para que el tablero reciba solicitudes reales, activa los comentarios en la pantalla Comentarios de Blup Studio. El estado de cada solicitud (planeado, en curso, hecho, rechazado) lo cambias en la pantalla Feedback.

Las solicitudes de la lista examples son ejemplos, como las notas de las otras secciones: aparecen en el tablero hasta que llega la primera solicitud real. En el archivo de la nota, cambia o borra la lista en las propiedades de esta nota.
%%
