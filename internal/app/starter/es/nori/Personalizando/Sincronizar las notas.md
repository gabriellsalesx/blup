---
title: Sincronizar las notas
date: {{date-8}}
description: Las mismas notas en más de una computadora, con Blup Cloud y cifrado de extremo a extremo o en un repositorio privado de GitHub.
tags: [Personalización, Studio]
image: "[[blup-sincronizar.jpg]]"
---
Sincronizar guarda una copia de las notas en la nube. En otra computadora, abre el Studio y sincroniza: las notas llegan. Edita donde quieras y sincroniza de nuevo.

En el Studio, haz clic en **Sincronizar**, arriba. La primera vez, elige dónde guardar las notas:

- **Blup Cloud** (recomendado): cifrado de extremo a extremo. Ni Blup puede leerlas.
- **GitHub**: en un repositorio privado de tu cuenta de GitHub. Sin necesidad de saber git.

## Con Blup Cloud

1. Haz clic en **Iniciar sesión con Blup**. Blup Cloud se abre en el navegador: revisa que el código sea el mismo y apruébalo.
2. Crea la **contraseña de las notas**. Ella mezcla las notas antes de que salgan de la computadora.
3. Guarda la **clave de recuperación** que aparece. Abre las notas si olvidas la contraseña, y no se vuelve a mostrar.

En otra computadora, inicia sesión con la misma cuenta y escribe la contraseña de las notas (o la clave de recuperación).

> [!warning] Nadie más tiene tu contraseña
> Ni Blup. Si pierdes la contraseña y la clave de recuperación, las notas en la nube no vuelven. Guarda la clave en un lugar seguro, fuera de la computadora.

| Plan | Por mes | Por año | Notas | Cómo sincroniza | Computadoras | Versiones anteriores |
| --- | --- | --- | --- | --- | --- | --- |
| Gratis | R$ 0 | R$ 0 | 50 MB | Con el botón | 1 | No |
| Pro | R$ 14,90 | R$ 149 | 1 GB | En tiempo real | 2 | 1 mes |
| Max | R$ 29,90 | R$ 299 | 10 GB | En tiempo real | 3 | 12 meses |
| Ultra | R$ 59,90 | R$ 599 | 50 GB | En tiempo real | Sin límite | 12 meses |

En tiempo real, las notas se sincronizan solas mientras escribes. En los planes de pago, el clic derecho en una nota muestra las **Versiones anteriores**: elige una y recupérala. Los planes de pago también publican sitios con Blup Cloud e incluyen IA para narrar y traducir.

## Con GitHub

1. Conecta tu cuenta de GitHub, con la misma clave que usas para publicar.
2. Blup crea un repositorio **privado**: solo tú lo ves.
3. Haz clic en **Revisar**. Blup compara las notas de aquí con las de GitHub y **te muestra lo que va a hacer** antes de hacerlo: lo que cambió aquí se envía, lo que cambió en otra computadora se recibe, y lo que se eliminó de un lado se elimina del otro.
4. Si todo está bien, haz clic en **Sincronizar ahora**.

> [!info] Por la línea de comandos
> Ejecuta `blup sync`. La línea de comandos sincroniza con GitHub, de la misma forma.

## Cuando los dos lados cambiaron

Si la misma nota se editó aquí y en otra computadora, no se pierde nada: tu versión se queda, y la otra se vuelve una copia con "conflict" en el nombre. Compara las dos y elimina la que sobre.

> [!info] Nada desaparece de repente
> Las notas eliminadas en otra computadora van a una papelera fuera de la carpeta de las notas. Si la carpeta del sitio aparece vacía de repente (por ejemplo, si cambió de lugar), Blup se detiene y no elimina nada en la nube.
