---
title: Comentários e reações
description: Votos no roadmap, reações no changelog e comentários nos posts, sem sair do site.
weight: 7
---
Quem visita o site pode votar nos itens do **roadmap**, reagir às versões do **changelog** e comentar nos posts do **blog**. Sem servidor e sem mensalidade: tudo fica em issues de um repositório público no seu GitHub.

## Ligar

Rode `blup personalizar` e escolha **Comentários e reações**. O assistente de site novo também pergunta no final.

1. O Blup cria o repositório público com o nome do site mais `-feedback`. Um site chamado "Meu Produto" fica com `meu-produto-feedback`.
2. Ele abre a página do app **Blup Comments** no GitHub. Escolha **Only select repositories**, marque o repositório `-feedback` e clique em **Install**. É esse app que deixa as pessoas votarem e comentarem sem sair do site.
3. Rode `blup publicar`. A cada publicação o Blup abre uma issue para cada página nova e mantém os títulos em dia.

> [!info] Sem o app
> Se você pular a instalação, tudo continua funcionando, mas votar, reagir e comentar levam a pessoa para a issue no GitHub. Instale depois pelo mesmo caminho em `blup personalizar`.

## Como aparece

- **Roadmap**: cada card mostra os votos e os comentários. Na página do item, o botão **Votar** soma o voto da pessoa; clicar de novo desfaz.
- **Changelog**: as reações aparecem embaixo de cada versão. Clique numa reação para somar a sua ou no botão de carinha para escolher outra.
- **Blog e itens**: a página mostra as reações, os comentários e uma caixa para comentar, que aceita Markdown.

## Quem pode participar

Ler é livre para todos. Para votar, reagir ou comentar, a pessoa clica em **Entrar com o GitHub** e autoriza o app uma vez. Ela continua na mesma página e pode participar à vontade; a conexão fica só no navegador dela, e **Sair** desconecta.

Para responder alguém, clique em **Responder** no comentário: a caixa já vem com um trecho citado e a menção à pessoa, e o GitHub avisa a pessoa (notificação e, se ela quiser, e-mail). Como dono do repositório, você é avisado de todo comentário novo.

Cada pessoa pode apagar os próprios comentários pelo botão **Apagar**. Você, como dono do repositório, vê esse botão em todos os comentários quando entra no site.

Exigir uma conta do GitHub afasta o spam. Para moderar mais, use as ferramentas do próprio GitHub: bloquear usuário ou trancar a conversa.

> [!tip] Os números se atualizam sozinhos
> O site lê o GitHub na hora em que a página abre. Um voto novo aparece sem publicar de novo.

## Desligar

Rode `blup personalizar`, escolha **Comentários e reações** e responda **Não**. Na próxima publicação os blocos saem do site. O repositório e as issues continuam no seu GitHub, para quando você quiser ligar de novo.
