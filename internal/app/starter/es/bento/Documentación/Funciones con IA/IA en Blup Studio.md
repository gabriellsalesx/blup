---
title: IA en Blup Studio
description: Qué hace la IA en el Studio y las tres formas de pagarla, incluso gratis.
weight: 1
---
Blup Studio usa IA en cuatro lugares. Todos son opcionales: Blup funciona completo sin IA.

| Función | Qué hace | Dónde leer |
| --- | --- | --- |
| Narración | Una voz lee el post, y el sitio muestra un reproductor arriba | [Narración de los posts](Narración%20de%20los%20posts.md) |
| Traducción | Traduce las notas al inglés, al español o al portugués, y el sitio recibe un selector de idioma | [Traducción](Traducción.md) |
| Podcast | En los blogs, las narraciones se vuelven episodios con feed para Spotify y Apple Podcasts | [Podcast de los blogs](Podcast%20de%20los%20blogs.md) |
| Agentes en la terminal | Claude Code, Codex, OpenCode, Gemini CLI o Antigravity trabajando en tus notas | [Terminal y agentes de IA](../Blup%20Studio/Terminal%20y%20agentes%20de%20IA.md) |

> [!info] Solo en Blup Studio
> La narración, la traducción, el podcast y la terminal con agentes existen solo en el Studio. Un sitio que ya tiene narraciones y podcast sigue publicando los dos con la línea de comandos.

## Tres formas de pagar

La narración y la traducción pueden funcionar de tres maneras. Eliges en **Configuración > IA y voz**.

### 1. En tu computadora, gratis

- **Voz en la computadora (Kokoro)**: gratis y sin internet. El Studio descarga el motor, el modelo y las voces una sola vez, verifica cada archivo y deja todo listo. Todavía no está disponible en todos los sistemas: el Studio avisa cuando no lo está.
- **Traducción en la computadora (Firefox Translations)**: el mismo traductor de Firefox, de Mozilla. Gratis, sin internet, y las notas no salen de la computadora. Descarga unos 100 MB una sola vez.

### 2. Con el plan de Blup Cloud

Los planes de pago (Pro, Max y Ultra) ya incluyen uso de IA para narración y traducción, sin clave de ningún servicio. El uso se mide por mes:

- En **Configuración > Blup Cloud**, la barra **IA este mes** muestra cuánto se usó y cuándo se reinicia.
- Cada semana usa como máximo la mitad del mes, para que el uso no se acabe de una vez.
- Max tiene el doble de uso que Pro, y Ultra, cinco veces más.
- Antes de narrar, el Studio muestra cuánto del mes va a usar ese texto.

Mira [Cuenta y planes](../Blup%20Cloud/Cuenta%20y%20planes.md).

### 3. Con tu clave de OpenRouter

OpenRouter es un servicio que da acceso a varios modelos de IA, pagados por uso.

1. En **Configuración > IA y voz > IA**, haz clic en **Crea una en OpenRouter**.
2. En tu cuenta de OpenRouter, haz clic en **Create Key**, ponle un nombre (por ejemplo, Blup Studio) y copia la clave.
3. Pega la clave en el Studio. Él comprueba que funciona y la guarda solo en tu computadora.

Cada uso gasta créditos de tu cuenta de OpenRouter: una narración cuesta unos US$ 0,15 por post, y la traducción, centavos de dólar por nota.

## ¿Cuál elegir?

| Si quieres | Usa |
| --- | --- |
| No gastar nada y no enviar el texto afuera | La computadora |
| No pensar en claves ni en créditos | El plan de Blup Cloud |
| Elegir entre muchos modelos y voces | OpenRouter |

> [!tip] En la nube queda un poco más natural
> La traducción en la nube suele sonar más natural; la de la computadora es gratis y privada. En **Quién traduce**, elige **En la computadora** o **IA en la nube**.
