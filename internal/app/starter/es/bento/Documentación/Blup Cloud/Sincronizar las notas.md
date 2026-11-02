---
title: Sincronizar las notas
description: Las mismas notas en varias computadoras, cifradas de extremo a extremo. O en un repositorio de GitHub.
weight: 2
---
Sincronizar guarda una copia de las notas en la nube. En otra computadora, las notas llegan; edita donde quieras y se encuentran.

![Dos computadoras sincronizando](../../Adjuntos/blup-sincronizar.jpg)

## Elegir dónde guardar

Haz clic en **Sincronizar**, arriba (en un sitio) o en la barra lateral (en una bóveda). En **Dónde guardar las notas**:

- **Blup Cloud** (recomendado): cifrado de extremo a extremo. Ni siquiera Blup puede leerlo.
- **GitHub**: en un repositorio privado de tu cuenta de GitHub. Sin git.

## Con Blup Cloud

### La primera vez

1. Inicia sesión en tu cuenta (consulta [Cuenta y planes](Cuenta%20y%20planes.md)).
2. **Crea la contraseña de las notas**, con al menos 8 caracteres. Ella revuelve las notas antes de que salgan de la computadora.
3. **Guarda tu clave de recuperación**. Aparece una sola vez: cópiala y guárdala en un lugar seguro, fuera de esta computadora. Marca **Guardé la clave en un lugar seguro** y haz clic en **Continuar**.

> [!danger] Nadie más tiene esa contraseña
> Ni siquiera Blup. Si pierdes la contraseña **y** la clave de recuperación, las notas de la nube no vuelven. Las notas de tu computadora siguen ahí.

### En otra computadora

1. Instala el Studio e inicia sesión en la misma cuenta.
2. El Studio pide **Escribe la contraseña de las notas**. Usa la contraseña o la clave de recuperación y haz clic en **Abrir las notas**.
3. Elige los sitios y bóvedas que quieres **Traer**.

### Cuándo sincroniza

- **Gratis**: cuando tocas **Sincronizar**.
- **Pro, Max y Ultra**: solo, mientras escribes. Las otras computadoras se enteran al momento.

### Cómo funciona el cifrado

Todo se comprime y se cifra en tu computadora, antes de salir. La nube guarda solo pedazos ilegibles: no ve el texto, ni los nombres de las notas y carpetas. La clave que abre las notas está protegida por tu contraseña y, en una segunda copia, por la clave de recuperación.

Los archivos más grandes que el límite de tu plan no se sincronizan, y el Studio avisa cuáles.

## Con GitHub

1. En **Dónde guardar las notas**, elige **GitHub** y conecta tu cuenta (una clave, como en la publicación).
2. Blup crea un repositorio **privado** para las notas.
3. **Revisar** muestra cuántas notas se van a enviar y a recibir. **Sincronizar ahora** lo hace.

En otra computadora, en la pantalla de inicio, **Ya tengo un sitio de Blup** lista los sitios guardados en tu GitHub para **Traer**. Desde la línea de comandos, consulta [Sincronizar](../Línea%20de%20comandos/Sincronizar.md).

## Conflictos

Si la misma nota se editó en dos computadoras antes de sincronizar, Blup **guarda las dos**: la tuya, con el nombre normal, y la otra, con "conflict" y la fecha en el nombre. Compáralas, junta lo que quieras y elimina la copia.

| Situación | Qué pasa |
| --- | --- |
| Cambió solo aquí | Se envía |
| Cambió solo en otra computadora | Se recibe |
| Eliminada de un lado | Se elimina del otro |
| Eliminada de un lado y editada del otro | Se queda la versión editada |
| Editada de los dos lados | Se quedan las dos |

> [!tip] Lo que no se sincroniza
> Las carpetas que empiezan con punto quedan fuera. La configuración del sitio que necesita viajar (nombre, tema, destino de la publicación) va incluida; las contraseñas y las claves de acceso, no.
