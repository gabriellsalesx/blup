---
title: Cookies e estatísticas
date: {{date-11}}
description: Google Analytics, Meta Pixel, Umami, Plausible, vídeos do YouTube e o banner de cookies.
tags: [Personalização, Studio]
---
O Blup não coloca cookies de rastreamento próprios. Esta parte é para quando você usa ferramentas para **contar visitas** ou **medir anúncios**, e para **vídeos do YouTube** nas notas.

## Ferramentas prontas

No menu ao lado, clique em **Cookies e estatísticas**. Em **Ferramentas**, clique em **Adicionar ferramenta**, escolha qual e cole só o ID dela:

| Ferramenta | O que colar | Onde achar |
| --- | --- | --- |
| Google Analytics | ID de medição, `G-...` | Administrador › Fluxos de dados |
| Google Tag Manager | ID do contêiner, `GTM-...` | No topo do espaço de trabalho, ao lado do nome do contêiner |
| Meta Pixel | ID do pixel, só números | Gerenciador de Eventos › Fontes de dados |
| Umami | ID do site | Settings › Websites › Edit |
| Plausible | Domínio do site | O domínio cadastrado no Plausible |

Usa Umami ou Plausible no seu próprio servidor? Preencha também **Servidor próprio (opcional)** com o endereço do script.

O **Google Tag Manager** pode ter tags de estatística e de anúncios. Com o banner, ele só carrega depois que a pessoa aceita **Estatísticas** ou **Marketing**, e o Blup informa o que foi aceito pelo Consent Mode do Google: as tags do Google seguem cada escolha. Tags de outras empresas dentro do Tag Manager precisam ser configuradas para respeitar o consentimento.

## O banner

Na mesma tela, ligue o **Banner de cookies**. Ligado:

- Na primeira visita aparece um aviso com **Aceitar todos**, **Recusar** e **Escolher**.
- Em **Escolher**, a pessoa liga ou desliga cada grupo: **Estatísticas** (Google Analytics, Umami, Plausible), **Marketing** (Meta Pixel), o Google Tag Manager nos dois e **Vídeos e conteúdo externo** (YouTube). Os **Essenciais** (tema claro ou escuro, entrar para comentar) ficam sempre ligados.
- Nada de terceiros carrega antes do aceite. A escolha fica guardada no navegador por um ano.
- O link **Cookies** no rodapé reabre as escolhas a qualquer momento.

Desligado, as ferramentas e os vídeos carregam direto, sem perguntar.

## Política de privacidade

Ainda na mesma tela, escolha onde fica a **Política de privacidade**:

- **Página do site (modelo para editar)**: o Blup cria a nota **Política de privacidade** na pasta do site, já escrita a partir das ferramentas, dos vídeos, dos comentários e da hospedagem. Clique em **Abrir a política**, revise e edite à vontade: ela é sua e o Blup nunca reescreve. Um comentário no topo da nota lembra o que conferir e não aparece no site.
- **Endereço meu**, se a política já existe em outro lugar.

Preencha também o **E-mail de contato (opcional)**. A página aparece com link no aviso de cookies e no rodapé, fica fora das listas de posts e não recebe comentários. Se mudar as ferramentas depois, atualize a nota.

## Vídeos do YouTube

Cole o link do vídeo como imagem:

```md
![Como publicar](https://www.youtube.com/watch?v=XXXXXXXXXXX)
```

No site vira o player, pelo endereço `youtube-nocookie.com`. Com o banner ligado, aparece o botão **Carregar vídeo** até a pessoa aceitar vídeos.

Para desligar tudo, remova as ferramentas e desligue o banner na mesma tela.

> [!info] Pela linha de comando
> Rode `blup personalizar` e escolha **Cookies, estatísticas e anúncios**. O terminal pergunta as mesmas coisas: as ferramentas, os IDs, o banner e a política de privacidade.
