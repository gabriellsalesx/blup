---
title: Formatação que funciona
date: {{date-4}}
description: Tudo que você escreve no Studio ou no Obsidian e aparece no site, com exemplos.
tags: [Escrita]
image: "[[blup-formatacao.jpg]]"
---
Este post é um mostruário. Abra a nota no Studio (ou no Obsidian) e veja como cada bloco foi escrito. No Studio, quase tudo tem um botão na **barra de formatação** e um item no menu `/`, e o editor já mostra o efeito: as marcas só aparecem quando o cursor passa por elas.

## Texto

**Negrito**, *itálico*, ~~riscado~~, ==marca-texto==, `código` e [link para fora](https://github.com/gabriellsalesx/blup). Uma nota de rodapé também funciona.[^1]

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
| Mochi | Blog com cards e capas |
| Nori | Blog de uma coluna, com o texto em primeiro plano |
| Bento | Site de produto |

## Citação

> Escrever é pensar duas vezes.

## Callouts

Os callouts viram caixas coloridas, com ícone. No Studio, use **Destaque (callout)**:

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

Texto entre `%%` e `%%` é um comentário: aparece enquanto você edita, no Studio ou no Obsidian, e some no site. Serve para lembretes e rascunhos dentro do post. %%Este trecho, por exemplo, só existe nas notas.%%

## Linha divisória

---

Três hífens numa linha sozinha desenham a linha acima.
