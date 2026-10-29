---
title: Formatação que funciona
date: {{date-4}}
description: Tudo que você escreve no Studio e aparece no site, com exemplos.
tags: [Escrita]
image: "[[blup-formatacao.jpg]]"
---
Este post é um mostruário. Abra a nota no Studio e veja como cada bloco foi escrito. Quase tudo tem um botão na **barra de formatação**, em cima do texto. O botão direito no editor também tem os menus **Formatar** e **Inserir**.

## Texto

**Negrito**, *itálico*, ~~riscado~~, ==marca-texto==, `código` e [link para fora](https://github.com/gabriellsalesx/blup). Uma nota de rodapé também funciona.[^1]

[^1]: As notas de rodapé vão para o fim do post, com link de volta.

Selecione o texto e use os botões **Negrito** (Ctrl+B), **Itálico** (Ctrl+I), **Riscado**, **Marca-texto**, **Código** e **Link** (Ctrl+K). A nota de rodapé não tem botão: escreva `[^1]` no texto e, numa linha sozinha, `[^1]:` seguido da nota.

## Listas

- Uma lista
- com itens
  - e subitens

1. Lista numerada
2. Segundo passo

- [x] Tarefa feita
- [ ] Tarefa pendente

Os botões **Lista**, **Lista numerada** e **Lista de tarefas** fazem cada uma. Os títulos ficam no botão **Título**.

## Tabela

| Tema | Para quem |
| --- | --- |
| Nori | Blog com uma coluna só |
| Mochi | Blog com grade de posts |

O botão **Tabela** cria uma tabela pronta para preencher. Clique numa célula com o botão direito para inserir ou apagar linhas e colunas.

## Citação

> Escrever é pensar duas vezes.

Use o botão **Citação (frase em destaque)**.

## Callouts

O botão **Destaque (callout)** abre a lista de tipos. Cada um vira uma caixa colorida, com ícone:

> [!note] Nota
> O tipo vai entre colchetes: `> [!note]`. O título é opcional.

> [!tip] Dica
> Também existem info, todo, success, question, warning, failure, danger, bug, example, quote e abstract.

> [!warning] Atenção
> Sem título, o callout mostra o nome do tipo.

> [!danger]
> Este não tem título.

> [!example]- Clique para abrir
> Marque **Começa recolhido** no mesmo menu e o callout já nasce fechado. Por baixo, é um `-` depois do tipo.

## Código

```go
package main

import "fmt"

func main() {
	fmt.Println("Olá, Blup!")
}
```

Use **Bloco de código** no menu Inserir do botão direito. O destaque de cores segue a linguagem escrita depois das três crases, como `go` aqui.

## Fórmulas

Fórmulas no meio do texto, como $E = mc^2$, e em bloco:

$$
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

As fórmulas não têm botão: escreva entre `$` (no meio do texto) ou entre `$$` (em bloco).

## Comentários

Texto entre `%%` e `%%` é um comentário: aparece enquanto você edita e some no site. Serve para lembretes e rascunhos dentro do post. Selecione o texto e use **Comentário (não vai para o site)**, no menu Formatar do botão direito. %%Este trecho, por exemplo, só existe no editor.%%

## Linha divisória

---

O botão **Linha divisória** desenha a linha acima.
