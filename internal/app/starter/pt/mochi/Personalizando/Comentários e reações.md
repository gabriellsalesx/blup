---
title: Comentários e reações
date: {{date-9}}
description: Quem lê reage e comenta nos posts, sem sair do blog. Tudo fica no seu GitHub.
tags: [Personalização, CLI]
---
Quem visita o blog pode reagir aos posts (👍 ❤️ 🎉 🚀 e outras) e deixar comentários. Sem servidor e sem mensalidade: tudo fica em issues de um repositório público no seu GitHub.

## Ligar

Rode `blup personalizar` e escolha **Comentários e reações nos posts**. O assistente de site novo também pergunta no final.

1. O Blup cria o repositório público com o nome do site mais `-feedback`. Um blog chamado "Meu Blog" fica com `meu-blog-feedback`.
2. Ele abre a página do app **Blup Comments** no GitHub. Escolha **Only select repositories**, marque o repositório `-feedback` e clique em **Install**. É esse app que deixa as pessoas reagirem e comentarem sem sair do blog.
3. Rode `blup publicar`. A cada publicação o Blup abre uma issue para cada post novo e mantém os títulos em dia.

> [!info] Sem o app
> Se você pular a instalação, tudo continua funcionando, mas reagir e comentar levam a pessoa para a issue no GitHub. Instale depois pelo mesmo caminho em `blup personalizar`.

## Como aparece

No fim de cada post ficam as reações, os comentários e uma caixa para comentar, que aceita Markdown. Clique numa reação para somar a sua ou no botão de carinha para escolher outra; clicar de novo desfaz.

Para responder alguém, clique em **Responder** no comentário: a caixa já vem com um trecho citado e a menção à pessoa, e o GitHub avisa ela (notificação e, se ela quiser, e-mail). Como dono do repositório, você é avisado de todo comentário novo.

## Quem pode participar

Ler é livre para todos. Para reagir ou comentar, a pessoa clica em **Entrar com o GitHub** e autoriza o app uma vez. Ela continua no mesmo post; a conexão fica só no navegador dela, e **Sair** desconecta.

Cada pessoa pode apagar os próprios comentários pelo botão **Apagar**. Você, como dono do repositório, vê esse botão em todos os comentários quando entra no blog.

Exigir uma conta do GitHub afasta o spam. Para moderar mais, use as ferramentas do próprio GitHub: bloquear usuário ou trancar a conversa.

## Desligar

Rode `blup personalizar`, escolha **Comentários e reações nos posts** e responda **Não**. Na próxima publicação os blocos saem do blog. O repositório e as issues continuam no seu GitHub.
