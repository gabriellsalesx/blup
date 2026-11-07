---
title: Publicar desde el Studio
description: Elige dónde, conecta la cuenta una vez y publica con un botón.
weight: 2
---
Haz clic en **Publicar**, arriba o al final de la barra lateral. La pantalla muestra **Dónde publicar**, con Blup Cloud primero.

## Con Blup Cloud

1. Elige **Blup Cloud**. Sin cuenta, el Studio te pide iniciar sesión. En el plan Gratis, explica que esta opción viene en los planes de pago y muestra los planes.
2. En **Dirección del sitio**, escribe el nombre: de 3 a 40 letras minúsculas, números y guiones.
3. Mientras escribes, el Studio comprueba si está libre: **Disponible**, **Ya está en uso** (con sugerencias parecidas para tocar) o reservado.
4. Haz clic en **Publicar ahora**.

Mira los detalles en [Sitios con Blup Cloud](../Blup%20Cloud/Sitios%20con%20Blup%20Cloud.md).

## Con Cloudflare Workers

1. Elige **Cloudflare Workers** y haz clic en **Conectar la cuenta**.
2. **Abrir en el navegador** abre el formulario de la clave, ya con los permisos correctos. Haz clic en **Continue to summary** y luego en **Create Token**.
3. Pega la clave y haz clic en **Conectar**. Se queda solo en tu computadora.
4. Elige la cuenta y el **Nombre del sitio**: es el inicio de la dirección, `nombre.tucuenta.workers.dev`. Si la cuenta todavía no tiene una dirección en workers.dev, el Studio pide la **Dirección de tu cuenta**: escribe el nombre y haz clic en **Crear**. Se elige una vez y sirve para todos los sitios de la cuenta.
5. Haz clic en **Publicar ahora**.

## Con GitHub Pages

1. Elige **GitHub Pages** y haz clic en **Conectar la cuenta**.
2. **Abrir en el navegador** abre el formulario de la clave. En **Repository access**, elige **All repositories**, genera la clave y cópiala.
3. Pega la clave y haz clic en **Conectar**.
4. Elige el **Repositorio**: se convierte en la dirección, `usuario.github.io/repositorio`.
5. Haz clic en **Publicar ahora**.

## Mientras publica

La pantalla muestra cada etapa. Al final, aparece **En línea en** con la dirección y el botón **Abrir el sitio**. Si algo sale mal, el detalle queda en la pantalla Publicar.

> [!tip] Las claves sirven para todos los sitios
> Las cuentas conectadas quedan en **Configuración > Publicación**. Allí también las desconectas.

## Después de publicar

- Para enviar los cambios, haz clic en **Publicar** de nuevo. Mira [Actualizar el sitio](Actualizar%20el%20sitio.md).
- Con Blup Cloud, en los planes Max y Ultra, aparece el bloque **Dominio propio**. Con Cloudflare o con GitHub, el sitio queda en la dirección gratis. Mira [Dominio propio](Dominio%20propio.md).
