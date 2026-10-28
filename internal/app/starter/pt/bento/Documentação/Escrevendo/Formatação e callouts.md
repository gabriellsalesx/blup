---
title: Formatação e callouts
description: Texto, listas, tabelas e as caixas de destaque do Obsidian.
weight: 3
---
## Texto

**Negrito**, *itálico*, ~~riscado~~, ==marca-texto== e `código`. Notas de rodapé também.[^1]

[^1]: Elas vão para o fim da página, com link de volta.

## Listas

- Itens com hífen
  - e subitens

1. Passos numerados
2. Em ordem

- [x] Tarefa feita
- [ ] Tarefa pendente

## Tabelas

| Seção | Pasta | Propriedade |
| --- | --- | --- |
| Documentação | Documentação | `blup: docs` |
| Roadmap | Roadmap | `blup: roadmap` |

## Títulos

Os títulos de nível 2 (`##`) e 3 (`###`) aparecem em **Nesta página**, ao lado do texto, e ganham um link próprio.

## Callouts

> [!note] Nota
> Escreva `> [!note] Título` e o texto nas linhas seguintes, começando com `>`.

> [!info]
> Sem título, o callout mostra o nome do tipo.

> [!tip] Dica
> Tipos: note, abstract, info, todo, tip, success, question, warning, failure, danger, bug, example e quote. Os apelidos do Obsidian (hint, important, faq, caution, error...) também valem.

> [!success] Sucesso
> Cada tipo tem cor e ícone.

> [!question] Pergunta
> Útil em perguntas frequentes.

> [!warning] Atenção
> Para avisos.

> [!danger] Perigo
> Para o que não tem volta.

> [!faq]- Callout fechado
> Um `-` depois do tipo deixa o callout fechado; clique para abrir. Um `+` deixa aberto, mas dobrável.
