---
title: Feedback
description: Um quadro onde quem visita pede recursos, conta bugs e vota. Só no Blup Studio.
weight: 8
---
O **Feedback** é um quadro de pedidos: quem visita o site sugere recursos, conta bugs, propõe melhorias e vota nos pedidos dos outros. Você acompanha tudo e muda o status pelo Studio.

> [!info] Só no Blup Studio
> A seção Feedback é adicionada pela tela **Seções** do Studio, e os pedidos são administrados na tela **Feedback**. Ela precisa dos [comentários](Comentários%20e%20reações.md) ligados.

## Adicionar

1. Ligue os comentários, se ainda não ligou. Veja [Comentários e reações](Comentários%20e%20reações.md).
2. Na tela **Seções**, clique em **Adicionar** ao lado de **Feedback**.
3. O Studio cria a pasta **Feedback**, com alguns pedidos de exemplo.
4. Publique o site.

## O quadro no site

| Controle | O que faz |
| --- | --- |
| Categorias | Todas, Recurso, Bug, Melhoria e Outro, com a contagem de cada uma |
| Status | Todos, os ativos (Novo, Planejado, Em andamento) ou um status |
| Mais votados e Recentes | A ordem da lista |
| Busca | Procura no título e no texto |
| Novo pedido | Tipo, título e descrição. A pessoa entra com o GitHub para enviar |
| Carregar mais pedidos | Mostra mais dez |

Clique num pedido para ver o texto, os votos e os comentários.

## Os pedidos de exemplo

A nota **_index** da pasta Feedback guarda alguns pedidos de exemplo, na propriedade `examples`. Eles mostram como o quadro fica antes de chegar o primeiro pedido de verdade. Quando chega o primeiro pedido real, os exemplos somem do quadro.

Pode editar ou apagar os exemplos à vontade. Num site em vários idiomas, os textos deles também são traduzidos.

## Responder e mudar o status

Clique em **Feedback** na faixa lateral do Studio.

- A tela lista os pedidos reais, dos mais votados para os menos, com tipo, autor, comentários, votos e idioma.
- Filtre por categoria.
- Clique no título para abrir no GitHub, onde você responde.
- Mude o **Status**: Novo, Planejado, Em andamento, Feito ou Recusado. Feito e Recusado fecham o pedido; os outros reabrem.
- **Atualizar** lê de novo.

O site mostra o novo status em até um minuto, sem publicar de novo.

> [!tip] Onde ficam os pedidos
> Cada pedido é uma issue no repositório dos comentários, no seu GitHub. Não existe uma nota por pedido.
