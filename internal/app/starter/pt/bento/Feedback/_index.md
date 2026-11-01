---
title: Feedback
linkTitle: Feedback
blup: feedback
weight: 6
description: Conte o que falta, o que quebrou e o que pode melhorar. Vote nos pedidos de outras pessoas.
examples:
  - title: "Modo escuro automático"
    kind: feature
    status: planned
    votes: 42
    days: 2
    body: "Seguir o tema do sistema, sem precisar trocar à mão toda noite."
    replies:
      - author: marina
        days: 1
        text: "Seria ótimo, à noite a tela clara cansa bastante."
      - author: rafael
        days: 1
        text: "Mais um voto aqui. Dá para seguir o horário do sistema também?"
  - title: "Busca não acha palavras com acento"
    kind: bug
    status: progress
    votes: 17
    days: 9
    body: "Procurar \"configuração\" não traz nada, mas \"configuracao\" traz."
    replies:
      - author: julia
        days: 6
        text: "Acontece comigo também, com \"não\" e \"ação\"."
      - author: time
        days: 2
        text: "Já achamos a causa e a correção sai na próxima versão."
  - title: "Atalho para criar uma nota nova"
    kind: improvement
    status: new
    votes: 11
    days: 15
    body: "Um atalho de teclado ajudaria muito quem escreve o dia todo."
    replies:
      - author: pedro
        days: 10
        text: "Ctrl+N seria o mais natural."
  - title: "Exportar para PDF"
    kind: feature
    status: done
    votes: 29
    days: 22
    body: "Poder baixar uma página inteira em PDF para mandar por e-mail."
    replies:
      - author: carla
        days: 20
        text: "Usei hoje para mandar um relatório, ficou perfeito. Obrigada!"
  - title: "Versão para Windows XP"
    kind: other
    status: declined
    votes: 3
    days: 40
    body: "Ainda uso um computador antigo no escritório."
    replies:
      - author: time
        days: 38
        text: "Infelizmente o Windows XP não recebe mais atualizações de segurança, então não vamos dar suporte."
---
%%
Esta pasta é o quadro de feedback do site: quem visita abre pedidos (bug, recurso, melhoria ou outro), vota e comenta, entrando com o GitHub. Os pedidos ficam nas issues do repositório de comentários do site, não aqui. O status de cada pedido (planejado, em andamento, feito, recusado) você muda no Blup Studio, na tela Feedback.

Os pedidos da lista examples são exemplos, como as notas das outras seções: aparecem no quadro até chegar o primeiro pedido de verdade. Pode apagar ou mudar a lista.
%%
