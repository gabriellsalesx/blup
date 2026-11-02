---
title: Dominio propio
description: El sitio en tu dirección, como misitio.com, con HTTPS.
weight: 3
---
En lugar de la dirección gratis (`sitio.pages.dev` o `tunombre.github.io`), el sitio puede responder en una dirección tuya, como `misitio.com` o `blog.misitio.com`. Necesitas un dominio comprado (GoDaddy, Cloudflare...).

> [!info] Dónde ya funciona
> Con **Cloudflare Pages** y con **GitHub Pages**, el dominio propio ya funciona. En los sitios publicados con **Blup Cloud**, viene en los planes Max y Ultra.

## Con Blup Studio

El bloque **Dominio propio** aparece en la pantalla **Publicar** después de que el sitio está en línea con Cloudflare o con GitHub.

1. Escribe solo la dirección, como `misitio.com`, y haz clic en **Conectar dominio**.
2. El Studio conecta el dominio donde está publicado el sitio.
3. Si falta un paso, muestra el **registro DNS** exacto (Tipo, Nombre y Valor, con **Copiar**). Crea ese registro en el sitio donde compraste el dominio.
4. Haz clic en **Verificar**. Puede tardar de minutos a algunas horas.
5. Cuando aparezca **Funcionando**, publica de nuevo para que los enlaces del sitio usen la nueva dirección.

**Cambiar** pasa a otro dominio. **Volver a la dirección gratis** desconecta el dominio de este sitio (sigue siendo tuyo).

## Con la línea de comandos

Ejecuta `blup customize` y elige **Dominio propio**. Blup pregunta el dominio, lo conecta donde está publicado el sitio, muestra el registro DNS si falta y activa el HTTPS. Para comprobar, vuelve a **Dominio propio** y elige **Comprobar si está funcionando**; cada `blup publish` también lo comprueba.

> [!tip] Dominio raíz en Cloudflare Pages
> Un dominio raíz (`misitio.com`, sin `www` ni `blog` delante) en Cloudflare Pages tiene que estar en tu cuenta de Cloudflare, lo cual es gratis: agrega el dominio en dash.cloudflare.com y, donde lo compraste, cambia los servidores de nombres por los que muestre Cloudflare. Si prefieres no cambiar eso, usa `www.misitio.com`.

> [!note] El podcast necesita dominio
> El feed del podcast de los blogs solo funciona en las apps con la dirección completa del sitio. Mira [Podcast de los blogs](../Funciones%20con%20IA/Podcast%20de%20los%20blogs.md).
