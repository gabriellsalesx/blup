---
title: Comentarios y reacciones
date: {{date-9}}
description: Quien lee reacciona y comenta en los posts, sin salir del blog. Todo queda en tu GitHub.
tags: [Personalización, Studio]
---
Quien visita el blog puede reaccionar a los posts (👍 ❤️ 🎉 🚀 y otras) y dejar comentarios. Sin servidor y sin mensualidad: todo queda en issues de un repositorio público en tu GitHub.

## Activarlo en el Studio

En el menú lateral, haz clic en **Comentarios** y activa **Mostrar comentarios y reacciones**. La primera vez, son dos pasos:

1. **Conectar GitHub**: haz clic en **Abrir el formulario**, crea la clave (los permisos ya vienen marcados), pégala en el Studio y haz clic en **Conectar**.
2. **Crear el espacio de los comentarios**: haz clic en **Crear y activar**. Blup crea en tu GitHub un repositorio público solo para los comentarios, con el nombre del sitio más `-feedback`.

Después, falta un detalle para que las personas comenten **sin salir del blog**: instalar la app de Blup en el repositorio. Haz clic en **Instalar la app en GitHub** y, en la página que se abre:

- elige **Only select repositories**;
- marca el repositorio `-feedback`;
- haz clic en **Install** y vuelve al Studio. Haz clic en **Ya la instalé**.

Por último, publica de nuevo para que los comentarios aparezcan en el sitio.

> [!info] Con Obsidian
> Ejecuta `blup customize` y elige **Comentarios y reacciones en los posts**. Blup hace los mismos pasos en el terminal.

> [!info] Sin la app
> Si te saltas la instalación, todo sigue funcionando, pero reaccionar y comentar llevan a la persona a la issue en GitHub. Puedes instalarla después, por el mismo camino.

## Cómo se ve

Al final de cada post están las reacciones, los comentarios y una caja para comentar, que acepta Markdown. Haz clic en una reacción para sumar la tuya o en el botón de la carita para elegir otra; hacer clic de nuevo la quita.

Para responder a alguien, haz clic en **Responder** en el comentario: la caja ya trae un fragmento citado y la mención a la persona, y GitHub le avisa. Como dueño del repositorio, recibes un aviso de cada comentario nuevo.

En un blog en varios idiomas, todas las versiones de un post comparten la misma conversación.

En la página de inicio, cuando los posts ya tienen reacciones y comentarios, aparecen las pestañas **Recientes**, **Destacados** (los posts con más reacciones) y **Debates** (los que tienen más comentarios), justo encima de la lista.

## Quién puede participar

Leer es libre para todos. Para reaccionar o comentar, la persona hace clic en **Iniciar sesión con GitHub** y autoriza la app una vez. Sigue en el mismo post; la conexión queda solo en su navegador, y **Cerrar sesión** la desconecta.

Cada persona puede eliminar sus propios comentarios con el botón **Eliminar**. Tú, como dueño del repositorio, ves ese botón en todos los comentarios cuando entras al blog.

Pedir una cuenta de GitHub aleja el spam. Para moderar más, usa las herramientas del propio GitHub: bloquear a un usuario o bloquear la conversación.

## Desactivarlo

En la pantalla **Comentarios**, desactiva **Mostrar comentarios y reacciones** y publica de nuevo: los bloques salen del blog. El repositorio y las issues siguen en tu GitHub.
