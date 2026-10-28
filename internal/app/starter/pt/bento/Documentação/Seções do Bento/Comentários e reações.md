---
title: Comentários e reações
description: Votos no roadmap, reações no changelog e comentários nos posts, guardados no GitHub.
weight: 7
---
Quem visita o site pode votar nos itens do **roadmap**, reagir às versões do **changelog** e comentar nos posts do **blog**. Sem servidor e sem mensalidade: tudo fica em issues de um repositório público no seu GitHub.

## Ligar

Rode `blup personalizar` e escolha **Comentários e reações**. O assistente de site novo também pergunta no final.

O Blup cria o repositório público com o nome do site mais `-feedback`. Um site chamado "Meu Produto" fica com `meu-produto-feedback`.

Depois, rode `blup publicar`. A cada publicação o Blup abre uma issue para cada página nova e mantém os títulos em dia.

## Como aparece

- **Roadmap**: cada card mostra os votos e os comentários. Na página do item, o botão **Votar** leva à issue.
- **Changelog**: as reações aparecem embaixo de cada versão.
- **Blog e itens**: a página mostra as reações e os comentários, com o botão **Comentar no GitHub**.

Ler é livre para todos. Votar, reagir e comentar acontece no GitHub, com uma conta do GitHub. Isso afasta o spam, e você modera pelas ferramentas do próprio GitHub: apagar comentário, bloquear usuário ou trancar a conversa.

> [!tip] Os números se atualizam sozinhos
> O site lê o GitHub na hora em que a página abre. Um voto novo aparece sem publicar de novo.

## Desligar

Rode `blup personalizar`, escolha **Comentários e reações** e responda **Não**. Na próxima publicação os blocos saem do site. O repositório e as issues continuam no seu GitHub, para quando você quiser ligar de novo.
