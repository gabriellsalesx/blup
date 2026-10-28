---
title: Fórmulas, código e comentários
description: Matemática, blocos de código com cores e notas que só você vê.
weight: 4
---
## Fórmulas

Entre `$` no meio do texto, como $a^2 + b^2 = c^2$, ou entre `$$` em bloco:

$$
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

As fórmulas viram MathML: o navegador desenha, sem baixar nada.

## Código

Três crases, o nome da linguagem e o código:

```js
function ola(nome) {
  return `Olá, ${nome}!`;
}
```

As cores seguem a linguagem: `go`, `js`, `py`, `sh`, `yaml`, `html`, `css` e muitas outras.

## Comentários

Texto entre `%%` e `%%` é um comentário do Obsidian: aparece enquanto você edita e não vai para o site. %%Como este trecho, que só existe na nota.%%

Use para lembretes, rascunhos e instruções para quem edita, como na nota **_index** da pasta do site.
