---
title: Terminal e agentes de IA
description: Um terminal dentro do Studio, já na pasta das notas, com botão para abrir uma IA.
weight: 8
---
O Studio tem um terminal embutido, que abre na parte de baixo da janela. Ele já começa **na pasta das notas** do site ou cofre aberto.

## Abrir

- Clique no ícone do terminal, no fim da faixa lateral, ou use Ctrl e a tecla do acento grave.
- **Novo terminal** abre outra aba. Cada aba tem menu para renomear, duplicar, reiniciar e fechar.
- **Minimizar** esconde o painel, e os terminais continuam rodando.

## Abrir uma IA

O botão **Abrir uma IA** lista os agentes de IA para terminal instalados no seu computador:

- Claude Code
- Codex
- OpenCode
- Gemini CLI
- Antigravity

Escolha um e ele começa dentro do terminal, já na pasta das notas. Quando você sai do agente, o terminal continua aberto.

> [!info] O Studio não instala esses programas
> Ele só reconhece os que já estão instalados. Se nenhum estiver, o Studio mostra links para conhecer cada um. A lista fica em **Configurações > Terminal > IAs no terminal**.

## O guia para a IA

Ao abrir o terminal, o Studio cria nas notas o arquivo `AGENTS.md`, que explica para a IA como um site do Blup funciona (pastas, propriedades, links). `CLAUDE.md` e `GEMINI.md` apontam para ele. Esses arquivos nunca viram páginas, e o Studio não os reescreve: pode editar à vontade.

## Ver e desfazer o que a IA mudou

Antes de a IA começar, o Studio guarda uma cópia das notas.

1. Depois que a IA trabalhar, clique em **Ver o que a IA mudou**.
2. O Studio lista cada nota **nova**, **alterada** ou **apagada**.
3. Gostou? Siga em frente. Não gostou? **Desfazer tudo** volta as notas para antes da IA. O que ela criou vai para a Lixeira.

> [!warning] Só com um agente
> A cópia é feita quando você abre um agente pelo botão **Abrir uma IA**. Um terminal comum, sem agente, não guarda cópia. O Studio guarda as cinco cópias mais recentes, e arquivos muito grandes podem ficar de fora.

## Ideias de uso

- "Revise a ortografia das notas da pasta Blog."
- "Crie uma página de documentação explicando a instalação, no formato deste site."
- "Liste as notas sem descrição."

Enquanto a IA escreve, o editor e a prévia mostram as mudanças na hora.
