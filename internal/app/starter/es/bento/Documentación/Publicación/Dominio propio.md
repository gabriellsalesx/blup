---
title: Dominio propio
description: El sitio en tu dirección, como misitio.com, con HTTPS. Con Blup Cloud, en los planes Max y Ultra.
weight: 3
---
En lugar de `tusitio.getblup.site`, el sitio puede responder en una dirección tuya, como `misitio.com` o `blog.misitio.com`. Necesitas un dominio comprado (GoDaddy, Cloudflare...).

> [!info] Solo con Blup Cloud
> El dominio propio se vincula con **Blup Cloud**, en los planes **Max** y **Ultra**. Los sitios publicados con **Cloudflare Workers** o con **GitHub Pages** quedan en la dirección gratis del servicio (`nombre.tucuenta.workers.dev` o `tunombre.github.io`). Para usar un dominio tuyo, publica el sitio con Blup Cloud: mira [Sitios con Blup Cloud](../Blup%20Cloud/Sitios%20con%20Blup%20Cloud.md).

## Con Blup Studio

El bloque **Dominio propio** aparece en la pantalla **Publicar** después de que el sitio está en línea con Blup Cloud.

1. Escribe solo el dominio, sin `https://`, como `misitio.com`, y haz clic en **Vincular**.
2. El Studio muestra los **registros DNS** que faltan (Tipo, Nombre y Valor, con **Copiar**): un TXT que prueba que el dominio es tuyo y el CNAME que lo apunta a Blup Cloud. Crea esos registros en el sitio donde compraste el dominio.
3. Haz clic en **Verificar**. Mientras la pantalla esté abierta, el Studio verifica de nuevo cada minuto. El DNS puede tardar de minutos a algunas horas.
4. Cuando aparezca **Activo** y **HTTPS activo**, el Studio publica el sitio de nuevo por su cuenta, para que los enlaces usen la nueva dirección.

La dirección `tusitio.getblup.site` sigue funcionando. **Quitar** desactiva el dominio de este sitio, que queda solo en `tusitio.getblup.site`.

> [!tip] Dominio raíz
> Un dominio raíz (`misitio.com`, sin `www` ni `blog` delante) necesita un DNS que acepte CNAME en la raíz (CNAME flattening o ALIAS). Si el tuyo no lo acepta, usa `www.misitio.com`.

> [!warning] ¿Cambiaste de plan?
> Al salir de Max o de Ultra, el dominio sigue funcionando durante 14 días. Si vuelves a uno de esos planes antes, nada cambia. Después del plazo, el sitio queda solo en `tusitio.getblup.site`, y para usar el dominio otra vez hay que vincularlo de nuevo.

> [!note] El podcast necesita la dirección completa
> El feed del podcast de los blogs solo funciona en las apps con la dirección completa del sitio. Con dominio propio, el feed usa tu dominio. Mira [Podcast de los blogs](../Funciones%20con%20IA/Podcast%20de%20los%20blogs.md).
