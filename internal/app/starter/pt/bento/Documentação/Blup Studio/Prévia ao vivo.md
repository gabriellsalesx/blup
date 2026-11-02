---
title: Prévia ao vivo
description: O site de verdade ao lado do editor, atualizado enquanto você escreve.
weight: 3
---
Ao abrir um site, o Studio liga a prévia ao lado do editor. É o seu site, gerado no computador, igual ao que vai para o ar.

## Como funciona

- **Escreveu, apareceu**: cada nota salva atualiza a prévia sozinha, sem piscar e mantendo a posição da página.
- **A prévia segue a nota**: abriu outra nota no editor, a prévia vai para a página dela.
- **E o contrário também**: clicou num link dentro da prévia, a nota daquela página abre no editor.
- **Configurações na hora**: nome, logo, menu e redes salvos na tela Site aparecem na prévia sem recarregar.
- **Rascunhos aparecem** na prévia (e nunca no site publicado).
- Se a nota tem narração, o player aparece no topo do post.

## Ligar e desligar

- O botão **Prévia** no topo, ou `Ctrl+Alt+P`, liga e desliga.
- Em **Configurações > Editor**, **Abrir com a prévia ligada** decide se a prévia começa ligada.
- Arraste a divisa entre o editor e a prévia para mudar as larguras. O Studio lembra.

> [!info] Só no computador
> A prévia roda só no seu computador. Ninguém mais vê, e nada vai para a internet até você publicar.

> [!note] Sem player de áudio no Linux?
> Em alguns Linux falta um componente de áudio do sistema (GStreamer). Nesse caso a prévia esconde o player, mas o site publicado mostra normalmente.

## Pela linha de comando

Sem o Studio, rode `blup ver` no terminal: a prévia abre no navegador e se atualiza a cada nota salva. Veja [Prévia](../Linha%20de%20comando/Prévia.md).
