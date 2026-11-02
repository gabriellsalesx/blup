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

Texto entre `%%` e `%%` é um comentário: aparece enquanto você edita e não vai para o site. %%Como este trecho, que só existe na nota.%%

Use para lembretes, rascunhos e instruções para quem edita, como na nota **_index** da pasta do site.

No Studio, a barra de formatação tem o botão **Comentário (não vai para o site)**. A narração também pula os comentários.

> [!tip] Bloco de código no Studio
> No editor do Studio, o bloco de código ganha fundo, o nome da linguagem e o botão **Copiar**. As três crases só aparecem com o cursor dentro do bloco.
