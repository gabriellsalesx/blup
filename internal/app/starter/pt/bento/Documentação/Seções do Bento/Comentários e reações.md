---
title: Comentários e reações
description: Votos no roadmap, reações no changelog e comentários nos posts, sem sair do site.
weight: 7
---
Quem visita o site pode votar nos itens do **roadmap**, reagir às versões do **changelog** e comentar nos posts do **blog**. Sem servidor e sem mensalidade: tudo fica em issues de um repositório público no seu GitHub.

## Ligar pelo Blup Studio

Clique em **Comentários** na faixa lateral e siga **Ligar pela primeira vez**:

1. **Conectar o GitHub**: clique em **Abrir formulário** (as permissões já vêm marcadas), crie a chave, cole e clique em **Conectar**.
2. **Criar o espaço dos comentários**: clique em **Criar e ligar**. O Blup cria um repositório público no seu GitHub só para os comentários.
3. **Comentar sem sair do site**: clique em **Instalar o app no GitHub**. Na página que abre, escolha **Only select repositories**, marque o repositório dos comentários e clique em **Install**. Volte e clique em **Já instalei**.
4. Publique de novo para os comentários aparecerem no site.

O interruptor **Mostrar comentários e reações** liga e desliga depois.

## Ligar pela linha de comando

Rode `blup personalizar` e escolha **Comentários e reações** (o assistente de site novo também pergunta no final).

1. O Blup cria o repositório público com o nome do site mais `-feedback`.
2. Ele abre a página do app **Blup Comments** no GitHub. Escolha **Only select repositories**, marque o repositório e clique em **Install**.
3. Rode `blup publicar`. A cada publicação o Blup abre uma issue para cada página nova e mantém os títulos em dia.

> [!info] Sem o app
> Se você pular a instalação, tudo continua funcionando, mas votar, reagir e comentar levam a pessoa para a issue no GitHub.

## Como aparece

- **Roadmap**: cada card mostra os votos e os comentários. Na página do item, o botão **Votar** soma o voto da pessoa; clicar de novo desfaz.
- **Changelog**: as reações aparecem embaixo de cada versão. Clique numa reação para somar a sua ou no botão de carinha para escolher outra.
- **Blog e itens**: a página mostra as reações, os comentários e uma caixa para comentar, que aceita Markdown.

## Quem pode participar

Ler é livre para todos. Para votar, reagir ou comentar, a pessoa clica em **Entrar com o GitHub** e autoriza o app uma vez. A conexão fica só no navegador dela, e **Sair** desconecta.

Para responder alguém, clique em **Responder** no comentário: a caixa já vem com um trecho citado e a menção à pessoa, e o GitHub avisa a pessoa. Como dono do repositório, você é avisado de todo comentário novo.

Cada pessoa pode apagar os próprios comentários pelo botão **Apagar**. Você, como dono do repositório, vê esse botão em todos os comentários quando entra no site.

Exigir uma conta do GitHub afasta o spam. Para moderar mais, use as ferramentas do próprio GitHub: bloquear usuário ou trancar a conversa.

> [!tip] Os números se atualizam sozinhos
> O site lê o GitHub na hora em que a página abre. Um voto novo aparece sem publicar de novo.

## Em vários idiomas

Todas as versões traduzidas de uma página usam a mesma conversa: votos e reações somam os leitores de todos os idiomas.

## Desligar

No Studio, desligue **Mostrar comentários e reações**. Na linha de comando, rode `blup personalizar`, escolha **Comentários e reações** e responda **Não**. Na próxima publicação os blocos saem do site. O repositório e as issues continuam no seu GitHub.
