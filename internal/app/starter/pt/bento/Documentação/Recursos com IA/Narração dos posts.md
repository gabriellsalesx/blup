---
title: Narração dos posts
description: Uma voz lê o post e o site mostra um player no topo. Grátis no computador ou pela nuvem.
weight: 2
---
A narração transforma o texto de um post num áudio MP3. No site, o post ganha um **player** no topo, para quem prefere ouvir.

> [!info] Só no Blup Studio
> A narração é criada no Studio. Depois de pronta, ela é um arquivo comum nos Anexos e é publicada por qualquer caminho.

## Escolher a voz (uma vez)

1. Abra **Configurações > IA e voz > Narração**.
2. Para cada idioma (Português, English, Español), escolha a voz. Elas vêm em grupos:
   - **No computador, grátis**: as vozes Kokoro. Em português: Dora, Alex e Santa. Na primeira vez, baixe a voz em **Configurações > IA e voz > IA**, botão **Baixar voz local**.
   - **Blup Cloud, incluso no plano**: precisa de um plano pago.
   - **OpenRouter, pago**: precisa da sua chave.
3. **Ouvir amostra** toca um trecho da voz.

A voz vale para todos os sites. Veja as formas de pagar em [IA no Blup Studio](IA%20no%20Blup%20Studio.md).

## Narrar um post

1. Abra o post no Studio.
2. Nas **Propriedades**, no bloco **Narração**, clique em **Gerar narração**.
3. O Studio mostra o andamento: preparando o texto, gerando a voz parte por parte, salvando o áudio.
4. Pronto: aparece a duração e o tamanho. Ouça no player da prévia, ao lado.

O áudio é salvo em **Anexos**, e a nota ganha as propriedades `audio`, `audio_hash` e `audio_seconds`.

## O que a voz lê

O título, a descrição e o texto. Ela pula o que não faz sentido ouvir: blocos de código, fórmulas, imagens, tabelas e comentários `%%`. Títulos e itens de lista ganham pausas.

## Quando o texto muda

O Studio compara o texto com o da narração:

- **Narração em dia com o texto**: tudo certo.
- **O texto mudou depois da narração**: clique em **Gerar de novo** para o áudio acompanhar.

## Posts em outros idiomas

A tradução de um post (na pasta `en`, `es` ou `pt`) usa a voz daquele idioma e ganha um áudio próprio, como `Post (en).mp3`. Veja [Sites em vários idiomas](../Configuração%20do%20site/Sites%20em%20vários%20idiomas.md).

> [!warning] Posts com senha não são narrados
> Um áudio público entregaria o texto protegido. Por isso o Studio recusa narrar posts com senha.

> [!tip] Vire podcast
> Nos blogs Mochi e Nori, as narrações podem virar episódios de um podcast. Veja [Podcast dos blogs](Podcast%20dos%20blogs.md).
