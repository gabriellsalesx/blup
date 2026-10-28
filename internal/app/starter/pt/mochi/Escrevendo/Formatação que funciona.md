---
title: Formatação que funciona
date: {{date-4}}
description: Tudo que você escreve no Obsidian e aparece no site, com exemplos.
tags: [Obsidian]
image: "[[blup-formatacao.jpg]]"
---
Este post é um mostruário. Abra a nota no Obsidian e veja como cada bloco foi escrito.

## Texto

**Negrito**, *itálico*, ~~riscado~~, ==marca-texto==, `código` e [link para fora](https://obsidian.md). Uma nota de rodapé também funciona.[^1]

[^1]: As notas de rodapé vão para o fim do post, com link de volta.

## Listas

- Uma lista
- com itens
  - e subitens

1. Lista numerada
2. Segundo passo

- [x] Tarefa feita
- [ ] Tarefa pendente

## Tabela

| Tema | Para quem |
| --- | --- |
| Mochi | Blog pessoal |
| Bento | Site de produto |

## Citação

> Escrever é pensar duas vezes.

## Callouts

Os callouts do Obsidian viram caixas coloridas, com o mesmo ícone:

> [!note] Nota
> O tipo vai entre colchetes: `> [!note]`. O título é opcional.

> [!tip] Dica
> Também existem info, todo, success, question, warning, failure, danger, bug, example, quote e abstract.

> [!warning] Atenção
> Sem título, o callout mostra o nome do tipo.

> [!danger]
> Este não tem título.

> [!example]- Clique para abrir
> Um `-` depois do tipo deixa o callout fechado. Um `+` deixa aberto, mas dobrável.

## Código

```go
package main

import "fmt"

func main() {
	fmt.Println("Olá, Blup!")
}
```

O destaque de cores segue a linguagem escrita depois das três crases.

## Fórmulas

Fórmulas no meio do texto, como $E = mc^2$, e em bloco:

$$
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

## Comentários

Texto entre `%%` e `%%` é um comentário do Obsidian: aparece enquanto você edita e some no site. Serve para lembretes e rascunhos dentro do post. %%Este trecho, por exemplo, só existe no Obsidian.%%

## Linha divisória

---

Três hífens numa linha sozinha desenham a linha acima.
