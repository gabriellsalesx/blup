---
title: IA no Blup Studio
description: O que a IA faz no Studio e as três formas de pagar por ela, inclusive de graça.
weight: 1
---
O Blup Studio usa IA em quatro lugares. Todos são opcionais: o Blup funciona inteiro sem IA.

| Recurso | O que faz | Onde ler |
| --- | --- | --- |
| Narração | Uma voz lê o post, e o site mostra um player no topo | [Narração dos posts](Narração%20dos%20posts.md) |
| Tradução | Traduz as notas para inglês, espanhol ou português, e o site ganha um seletor de idioma | [Tradução](Tradução.md) |
| Podcast | Nos blogs, as narrações viram episódios com feed para Spotify e Apple Podcasts | [Podcast dos blogs](Podcast%20dos%20blogs.md) |
| Agentes no terminal | Claude Code, Codex, OpenCode, Gemini CLI ou Antigravity trabalhando nas suas notas | [Terminal e agentes de IA](../Blup%20Studio/Terminal%20e%20agentes%20de%20IA.md) |

> [!info] Só no Blup Studio
> Narração, tradução, podcast e o terminal com agentes existem só no Studio. Um site que já tem narrações e podcast continua publicando os dois pela linha de comando.

## Três formas de pagar

Narração e tradução podem rodar de três jeitos. Você escolhe em **Configurações > IA e voz**.

### 1. No seu computador, de graça

- **Voz no computador (Kokoro)**: grátis e sem internet. O Studio baixa o motor, o modelo e as vozes uma vez, confere cada arquivo e deixa tudo pronto. Ainda não está disponível em todos os sistemas: o Studio avisa quando não está.
- **Tradução no computador (Firefox Translations)**: o mesmo tradutor do Firefox, da Mozilla. Grátis, sem internet, e as notas não saem do computador. Baixa cerca de 100 MB uma vez.

### 2. Pelo plano do Blup Cloud

Os planos pagos (Pro, Max e Ultra) já incluem uso de IA para narração e tradução, sem chave de nenhum serviço. O uso é medido por mês:

- Em **Configurações > Blup Cloud**, a barra **IA do mês** mostra quanto foi usado e quando renova.
- Cada semana usa no máximo metade do mês, para o uso não acabar de uma vez.
- O Max tem o dobro do uso do Pro, e o Ultra, cinco vezes.
- Antes de narrar, o Studio mostra quanto do mês aquele texto vai usar.

Veja [Conta e planos](../Blup%20Cloud/Conta%20e%20planos.md).

### 3. Com a sua chave do OpenRouter

O OpenRouter é um serviço que dá acesso a vários modelos de IA, pagos pelo uso.

1. Em **Configurações > IA e voz > IA**, clique em **Criar no OpenRouter**.
2. Na sua conta do OpenRouter, clique em **Create Key**, dê um nome (por exemplo, Blup Studio) e copie a chave.
3. Cole a chave no Studio. Ele confere se funciona e guarda só no seu computador.

Cada uso gasta créditos da sua conta no OpenRouter: uma narração custa cerca de US$ 0,15 por post, e a tradução, centavos de dólar por nota.

## Qual escolher?

| Se você quer | Use |
| --- | --- |
| Não gastar nada e não mandar o texto para fora | O computador |
| Não pensar em chave nem em créditos | O plano do Blup Cloud |
| Escolher entre muitos modelos e vozes | O OpenRouter |

> [!tip] Na nuvem fica um pouco mais natural
> A tradução na nuvem costuma soar mais natural; a do computador é grátis e privada. Em **Quem traduz**, escolha **No computador** ou **IA na nuvem**.
