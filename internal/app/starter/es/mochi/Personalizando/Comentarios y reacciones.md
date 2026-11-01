---
title: Comentarios y reacciones
date: {{date-9}}
description: Tus lectores reaccionan y comentan en los posts sin salir del blog. Todo queda en tu GitHub.
tags: [Personalización, CLI]
---
Quien visita el blog puede reaccionar a los posts (👍 ❤️ 🎉 🚀 y otras) y dejar comentarios. Sin servidor y sin mensualidad: todo queda en issues de un repositorio público en tu GitHub.

## Activar

Ejecuta `blup customize` y elige **Comentarios y reacciones en los posts**. El asistente de sitio nuevo también lo pregunta al final.

1. Blup crea el repositorio público con el nombre del sitio más `-feedback`. Un blog llamado "Mi Blog" queda con `mi-blog-feedback`.
2. Abre la página de la app **Blup Comments** en GitHub. Elige **Only select repositories**, marca el repositorio `-feedback` y haz clic en **Install**. Esta app es la que permite que la gente reaccione y comente sin salir del blog.
3. Ejecuta `blup publish`. En cada publicación Blup abre un issue para cada post nuevo y mantiene los títulos al día.

> [!info] Sin la app
> Si te saltas la instalación, todo sigue funcionando, pero reaccionar y comentar llevan a la persona al issue en GitHub. Instálala después por el mismo camino en `blup customize`.

## Cómo se ve

Al final de cada post aparecen las reacciones, los comentarios y una caja para comentar, que acepta Markdown. Haz clic en una reacción para sumar la tuya o en el botón de la carita para elegir otra; hacer clic de nuevo la quita.

Para responderle a alguien, haz clic en **Responder** en el comentario: la caja ya viene con un fragmento citado y la mención a la persona, y GitHub le avisa (notificación y, si quiere, correo). Como dueño del repositorio, recibes un aviso de cada comentario nuevo.

## Quién puede participar

Leer es libre para todos. Para reaccionar o comentar, la persona hace clic en **Iniciar sesión con GitHub** y autoriza la app una vez. Sigue en el mismo post; la conexión queda solo en su navegador, y **Cerrar sesión** la desconecta.

Cada persona puede eliminar sus propios comentarios con el botón **Eliminar**. Tú, como dueño del repositorio, ves ese botón en todos los comentarios cuando entras al blog.

Pedir una cuenta de GitHub aleja el spam. Para moderar más, usa las herramientas del propio GitHub: bloquear a un usuario o bloquear la conversación.

## Desactivar

Ejecuta `blup customize`, elige **Comentarios y reacciones en los posts** y responde **No**. En la próxima publicación los bloques salen del blog. El repositorio y los issues siguen en tu GitHub.
