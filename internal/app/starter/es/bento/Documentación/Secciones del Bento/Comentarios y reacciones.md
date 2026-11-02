---
title: Comentarios y reacciones
description: Votos en el roadmap, reacciones en el changelog y comentarios en los posts, sin salir del sitio.
weight: 7
---
Quien visita el sitio puede votar por los elementos del **roadmap**, reaccionar a las versiones del **changelog** y comentar en los posts del **blog**. Sin servidor y sin mensualidad: todo queda en issues de un repositorio público en tu GitHub.

## Activar con Blup Studio

Haz clic en **Comentarios** en la barra lateral y sigue **Activar por primera vez**:

1. **Conecta GitHub**: haz clic en **Abrir el formulario** (los permisos ya vienen marcados), crea la clave, pégala y haz clic en **Conectar**.
2. **Crea el espacio de comentarios**: haz clic en **Crear y activar**. Blup crea un repositorio público en tu GitHub solo para los comentarios.
3. **Comentar sin salir del sitio**: haz clic en **Instalar la app en GitHub**. En la página que se abre, elige **Only select repositories**, marca el repositorio de comentarios y haz clic en **Install**. Vuelve y haz clic en **Ya la instalé**.
4. Publica de nuevo para que los comentarios aparezcan en el sitio.

El interruptor **Mostrar comentarios y reacciones** los activa y desactiva después.

## Activar con la línea de comandos

Ejecuta `blup customize` y elige **Comentarios y reacciones (votos en el roadmap)** (el asistente de sitio nuevo también lo pregunta al final).

1. Blup crea el repositorio público con el nombre del sitio más `-feedback`.
2. Abre la página de la app **Blup Comments** en GitHub. Elige **Only select repositories**, marca el repositorio y haz clic en **Install**.
3. Ejecuta `blup publish`. En cada publicación Blup abre una issue para cada página nueva y mantiene los títulos al día.

> [!info] Sin la app
> Si te saltas la instalación, todo sigue funcionando, pero votar, reaccionar y comentar llevan a la persona a la issue en GitHub.

## Cómo se ve

- **Roadmap**: cada tarjeta muestra los votos y los comentarios. En la página del elemento, el botón **Votar** suma el voto de la persona; hacer clic de nuevo lo deshace.
- **Changelog**: las reacciones aparecen debajo de cada versión. Haz clic en una reacción para sumar la tuya o en el botón de carita para elegir otra.
- **Blog y elementos**: la página muestra las reacciones, los comentarios y un cuadro para comentar, que acepta Markdown.

## Quién puede participar

Leer es libre para todos. Para votar, reaccionar o comentar, la persona hace clic en **Iniciar sesión con GitHub** y autoriza la app una vez. La conexión queda solo en su navegador, y **Cerrar sesión** la desconecta.

Para responderle a alguien, haz clic en **Responder** en el comentario: el cuadro ya viene con un fragmento citado y la mención a la persona, y GitHub le avisa. Como dueño del repositorio, recibes un aviso de cada comentario nuevo.

Cada persona puede eliminar sus propios comentarios con el botón **Eliminar**. Tú, como dueño del repositorio, ves ese botón en todos los comentarios cuando inicias sesión en el sitio.

Exigir una cuenta de GitHub aleja el spam. Para moderar más, usa las herramientas del propio GitHub: bloquear usuarios o cerrar la conversación.

> [!tip] Los números se actualizan solos
> El sitio lee GitHub en el momento en que se abre la página. Un voto nuevo aparece sin volver a publicar.

## En varios idiomas

Todas las versiones traducidas de una página usan la misma conversación: los votos y las reacciones suman a los lectores de todos los idiomas.

## Desactivar

En el Studio, desactiva **Mostrar comentarios y reacciones**. En la línea de comandos, ejecuta `blup customize`, elige **Comentarios y reacciones (votos en el roadmap)** y responde **No**. En la próxima publicación los bloques salen del sitio. El repositorio y las issues siguen en tu GitHub.
