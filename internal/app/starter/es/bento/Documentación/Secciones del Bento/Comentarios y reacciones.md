---
title: Comentarios y reacciones
description: Votos en el roadmap, reacciones en el changelog y comentarios en los posts, sin salir del sitio.
weight: 7
---
Quien visita el sitio puede votar por los elementos del **roadmap**, reaccionar a las versiones del **changelog** y comentar en los posts del **blog**. Sin servidor y sin mensualidad: todo queda en issues de un repositorio público en tu GitHub.

## Activar

Ejecuta `blup customize` y elige **Comentarios y reacciones (votos en el roadmap)**. El asistente de sitio nuevo también lo pregunta al final.

1. Blup crea el repositorio público con el nombre del sitio más `-feedback`. Un sitio llamado "Mi Producto" queda con `mi-producto-feedback`.
2. Abre la página de la app **Blup Comments** en GitHub. Elige **Only select repositories**, marca el repositorio `-feedback` y haz clic en **Install**. Esa app es la que permite que las personas voten y comenten sin salir del sitio.
3. Ejecuta `blup publish`. En cada publicación Blup abre una issue para cada página nueva y mantiene los títulos al día.

> [!info] Sin la app
> Si te saltas la instalación, todo sigue funcionando, pero votar, reaccionar y comentar llevan a la persona a la issue en GitHub. Instálala después por el mismo camino en `blup customize`.

## Cómo se ve

- **Roadmap**: cada tarjeta muestra los votos y los comentarios. En la página del elemento, el botón **Votar** suma el voto de la persona; hacer clic de nuevo lo deshace.
- **Changelog**: las reacciones aparecen debajo de cada versión. Haz clic en una reacción para sumar la tuya o en el botón de carita para elegir otra.
- **Blog y elementos**: la página muestra las reacciones, los comentarios y un cuadro para comentar, que acepta Markdown.

## Quién puede participar

Leer es libre para todos. Para votar, reaccionar o comentar, la persona hace clic en **Iniciar sesión con GitHub** y autoriza la app una vez. Sigue en la misma página y puede participar cuanto quiera; la conexión queda solo en su navegador, y **Cerrar sesión** la desconecta.

Para responderle a alguien, haz clic en **Responder** en el comentario: el cuadro ya viene con un fragmento citado y la mención a la persona, y GitHub le avisa (notificación y, si quiere, correo). Como dueño del repositorio, recibes un aviso de cada comentario nuevo.

Cada persona puede eliminar sus propios comentarios con el botón **Eliminar**. Tú, como dueño del repositorio, ves ese botón en todos los comentarios cuando inicias sesión en el sitio.

Exigir una cuenta de GitHub aleja el spam. Para moderar más, usa las herramientas del propio GitHub: bloquear usuarios o cerrar la conversación.

> [!tip] Los números se actualizan solos
> El sitio lee GitHub en el momento en que se abre la página. Un voto nuevo aparece sin volver a publicar.

## Desactivar

Ejecuta `blup customize`, elige **Comentarios y reacciones (votos en el roadmap)** y responde **No**. En la próxima publicación los bloques salen del sitio. El repositorio y las issues siguen en tu GitHub, para cuando quieras activarlos de nuevo.
