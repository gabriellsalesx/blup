---
title: Comentarios y reacciones
date: {{date-9}}
description: Quien lee reacciona y comenta en los posts, sin salir del blog. Todo queda en tu GitHub.
tags: [Personalización, Studio]
---
Quien visita el blog puede reaccionar a los posts (👍 ❤️ 🎉 🚀 y otras) y dejar comentarios. Sin servidor y sin mensualidad: todo queda en issues de un repositorio público en tu GitHub.

## Activar

En el menú lateral, haz clic en **Comentarios** y activa **Mostrar comentarios y reacciones**. La primera vez son dos pasos:

1. **Conectar GitHub**: haz clic en **Abrir el formulario**, crea la clave (los permisos ya vienen marcados), pégala en Studio y haz clic en **Conectar**.
2. **Crear el espacio de los comentarios**: haz clic en **Crear y activar**. Blup crea en tu GitHub un repositorio público solo para los comentarios, con el nombre del sitio más `-feedback`.

Después falta un detalle para que la gente comente **sin salir del blog**: instalar la app de Blup en el repositorio. Haz clic en **Instalar la app en GitHub** y, en la página que se abre:

- elige **Only select repositories**;
- marca el repositorio `-feedback`;
- haz clic en **Install** y vuelve a Studio. Haz clic en **Ya la instalé**.

Por último, publica de nuevo para que los comentarios aparezcan en el sitio.

> [!info] Sin la app
> Si te saltas la instalación, todo sigue funcionando, pero reaccionar y comentar llevan a la persona al issue en GitHub. Puedes instalarla después, desde la misma pantalla.

## Cómo se ve

Al final de cada post están las reacciones, los comentarios y una caja para comentar, que acepta Markdown. Haz clic en una reacción para sumar la tuya o en el botón de la carita para elegir otra; un nuevo clic la quita.

En la página de inicio, cuando los posts ya tienen reacciones y comentarios, aparecen las pestañas **Recientes**, **Destacados** (los posts con más reacciones) y **Debates** (los que tienen más comentarios), justo arriba de la lista.

Para responderle a alguien, haz clic en **Responder** en el comentario: la caja ya viene con un fragmento citado y la mención a la persona, y GitHub le avisa (notificación y, si ella quiere, correo). Como dueño del repositorio, recibes un aviso de cada comentario nuevo.

## Quién puede participar

Leer es libre para todos. Para reaccionar o comentar, la persona hace clic en **Iniciar sesión con GitHub** y autoriza la app una vez. Sigue en el mismo post; la conexión queda solo en su navegador, y **Cerrar sesión** la desconecta.

Cada persona puede eliminar sus propios comentarios con el botón **Eliminar**. Tú, como dueño del repositorio, ves ese botón en todos los comentarios cuando entras al blog.

Pedir una cuenta de GitHub aleja el spam. Para moderar más, usa las herramientas del propio GitHub: bloquear usuarios o bloquear la conversación.

## Desactivar

En la pantalla **Comentarios**, desactiva **Mostrar comentarios y reacciones** y publica de nuevo: los bloques salen del blog. El repositorio y los issues siguen en tu GitHub.
