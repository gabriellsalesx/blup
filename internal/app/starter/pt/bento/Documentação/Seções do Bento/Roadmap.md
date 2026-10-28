---
title: Roadmap
description: Um quadro com três colunas, uma nota por item.
weight: 2
---
O roadmap é uma pasta cuja **_index** tem `blup: roadmap`. Cada nota dentro dela é um card.

## A coluna

A propriedade **status** escolhe a coluna. Vale em português ou inglês:

| Coluna | Escreva |
| --- | --- |
| Planejado | `planejado` ou `planned` (também é o padrão, sem status) |
| Em andamento | `em andamento`, `fazendo`, `in progress` |
| Concluído | `concluído`, `feito`, `lançado`, `done`, `shipped` |

## O card

```yaml
---
title: Mais temas
description: Novos visuais para escolher na criação do site.
status: planejado
category: Temas
weight: 1
---
```

- **title** e **description** aparecem no card.
- **category** é a etiqueta no pé do card. Sem ela, vale a primeira tag.
- **weight** ordena os cards dentro da coluna.
- O texto da nota vira a página do item, aberta ao clicar no card.

Para mudar um item de coluna, troque o **status** e salve.

## Ordem pelos votos

Com [comentários e reações](Comentários%20e%20reações.md) ligados, as colunas **Planejado** e **Em andamento** colocam os itens mais votados no topo, e o card mais votado do quadro ganha o selo **Mais votado**. O **weight** continua desempatando, e **Concluído** mantém a ordem.

Para ficar só com a ordem do **weight**, coloque `sort_by_votes: false` na _index do roadmap.
