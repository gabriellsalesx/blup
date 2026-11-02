---
title: Terminal y agentes de IA
description: Una terminal dentro del Studio, ya en la carpeta de las notas, con un botón para abrir una IA.
weight: 8
---
El Studio tiene una terminal integrada, que se abre en la parte de abajo de la ventana. Ya empieza **en la carpeta de las notas** del sitio o la bóveda abierta.

## Abrir

- Haz clic en el ícono de la terminal, al final de la barra lateral, o usa Ctrl y la tecla del acento grave.
- **Nueva terminal** abre otra pestaña. Cada pestaña tiene un menú para renombrar, duplicar, reiniciar y cerrar.
- **Minimizar** oculta el panel, y las terminales siguen ejecutándose.

## Abrir una IA

El botón **Abrir una IA** lista los agentes de IA para terminal instalados en tu computadora:

- Claude Code
- Codex
- OpenCode
- Gemini CLI
- Antigravity

Elige uno y empieza dentro de la terminal, ya en la carpeta de las notas. Cuando sales del agente, la terminal sigue abierta.

> [!info] El Studio no instala esos programas
> Solo reconoce los que ya están instalados. Si no hay ninguno, el Studio muestra enlaces para conocer cada uno. La lista está en **Configuración > Terminal > IAs en la terminal**.

## La guía para la IA

Al abrir la terminal, el Studio crea en las notas el archivo `AGENTS.md`, que le explica a la IA cómo funciona un sitio de Blup (carpetas, propiedades, enlaces). `CLAUDE.md` y `GEMINI.md` apuntan a él. Esos archivos nunca se convierten en páginas, y el Studio no los reescribe: puedes editarlos con libertad.

## Ver y deshacer lo que cambió la IA

Antes de que la IA empiece, el Studio guarda una copia de las notas.

1. Después de que la IA trabaje, haz clic en **Ver qué cambió la IA**.
2. El Studio lista cada nota **nueva**, **cambiada** o **eliminada**.
3. ¿Te gustó? Sigue adelante. ¿No te gustó? **Deshacer todo** deja las notas como estaban antes de la IA. Lo que creó va a la Papelera.

> [!warning] Solo con un agente
> La copia se hace cuando abres un agente con el botón **Abrir una IA**. Una terminal común, sin agente, no guarda copia. El Studio guarda las cinco copias más recientes, y los archivos muy grandes pueden quedar fuera.

## Ideas de uso

- "Revisa la ortografía de las notas de la carpeta Blog."
- "Crea una página de documentación que explique la instalación, con el formato de este sitio."
- "Lista las notas sin descripción."

Mientras la IA escribe, el editor y la vista previa muestran los cambios al momento.
