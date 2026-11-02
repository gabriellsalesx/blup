---
title: Comentários e reações
date: {{date-9}}
description: Quem lê reage e comenta nos posts, sem sair do blog. Tudo fica no seu GitHub.
tags: [Personalização, Studio]
---
Quem visita o blog pode reagir aos posts (👍 ❤️ 🎉 🚀 e outras) e deixar comentários. Sem servidor e sem mensalidade: tudo fica em issues de um repositório público no seu GitHub.

## Ligar no Studio

No menu ao lado, clique em **Comentários** e ligue **Mostrar comentários e reações**. Na primeira vez, são dois passos:

1. **Conectar o GitHub**: clique em **Abrir formulário**, crie a chave (as permissões já vêm marcadas), cole no Studio e clique em **Conectar**.
2. **Criar o espaço dos comentários**: clique em **Criar e ligar**. O Blup cria no seu GitHub um repositório público só para os comentários, com o nome do site mais `-feedback`.

Depois, falta um detalhe para as pessoas comentarem **sem sair do blog**: instalar o app do Blup no repositório. Clique em **Instalar o app no GitHub** e, na página que abre:

- escolha **Only select repositories**;
- marque o repositório `-feedback`;
- clique em **Install** e volte ao Studio. Clique em **Já instalei**.

Por fim, publique de novo para os comentários aparecerem no site.

> [!info] Pelo Obsidian
> Rode `blup personalizar` e escolha **Comentários e reações nos posts**. O Blup faz os mesmos passos pelo terminal.

> [!info] Sem o app
> Se você pular a instalação, tudo continua funcionando, mas reagir e comentar levam a pessoa para a issue no GitHub. Dá para instalar depois, pelo mesmo caminho.

## Como aparece

No fim de cada post ficam as reações, os comentários e uma caixa para comentar, que aceita Markdown. Clique numa reação para somar a sua ou no botão de carinha para escolher outra; clicar de novo desfaz.

Na página inicial, quando os posts já têm reações e comentários, aparecem as abas **Mais recente**, **Principais** (os posts com mais reações) e **Discussões** (os com mais comentários), logo acima da lista.

Para responder alguém, clique em **Responder** no comentário: a caixa já vem com um trecho citado e a menção à pessoa, e o GitHub avisa ela. Como dono do repositório, você é avisado de todo comentário novo.

Num blog em vários idiomas, todas as versões de um post dividem a mesma conversa.

## Quem pode participar

Ler é livre para todos. Para reagir ou comentar, a pessoa clica em **Entrar com o GitHub** e autoriza o app uma vez. Ela continua no mesmo post; a conexão fica só no navegador dela, e **Sair** desconecta.

Cada pessoa pode apagar os próprios comentários pelo botão **Apagar**. Você, como dono do repositório, vê esse botão em todos os comentários quando entra no blog.

Exigir uma conta do GitHub afasta o spam. Para moderar mais, use as ferramentas do próprio GitHub: bloquear usuário ou trancar a conversa.

## Desligar

Na tela **Comentários**, desligue **Mostrar comentários e reações** e publique de novo: os blocos saem do blog. O repositório e as issues continuam no seu GitHub.
