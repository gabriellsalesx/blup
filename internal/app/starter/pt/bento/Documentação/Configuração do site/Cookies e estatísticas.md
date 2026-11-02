---
title: Cookies e estatísticas
description: Google Analytics, Meta Pixel, Umami, Plausible e o banner de cookies.
weight: 6
---
O Blup não coloca cookies de rastreamento próprios. Esta parte é para quando você usa ferramentas para **contar visitas** ou **medir anúncios**, e para **vídeos do YouTube** nas notas.

## Pelo Blup Studio

Clique em **Cookies e estatísticas** na faixa lateral.

1. Em **Ferramentas**, clique em **Adicionar ferramenta** e escolha: Google Analytics, Google Tag Manager, Meta Pixel, Umami ou Plausible.
2. Cole só o **ID** de cada uma. Se o formato não parecer certo, o Studio avisa.
3. Usa Umami ou Plausible no seu próprio servidor? Preencha **Servidor próprio (opcional)** com o endereço do script.
4. Ligue ou desligue o **Banner de cookies**.
5. Em **Política de privacidade**, escolha **Página do site (modelo para editar)** ou **Endereço meu**. Pode informar um **E-mail de contato**.
6. **Abrir a política** abre a nota da política no editor.
7. Publique de novo para valer no site.

## Pela linha de comando

Rode `blup personalizar` e escolha **Cookies, estatísticas e anúncios**. As perguntas são as mesmas.

## Onde achar cada ID

| Ferramenta | O que colar | Onde achar |
| --- | --- | --- |
| Google Analytics | ID de medição, `G-...` | Administrador › Fluxos de dados |
| Google Tag Manager | ID do contêiner, `GTM-...` | No topo do espaço de trabalho, ao lado do nome do contêiner |
| Meta Pixel | ID do pixel, só números | Gerenciador de Eventos › Fontes de dados |
| Umami | ID do site | Settings › Websites › Edit |
| Plausible | Domínio do site | O domínio cadastrado no Plausible |

O **Google Tag Manager** pode ter tags de estatística e de anúncios. Com o banner, ele só carrega depois que a pessoa aceita **Estatísticas** ou **Marketing**, e o Blup informa o que foi aceito pelo Consent Mode do Google. Tags de outras empresas dentro do Tag Manager precisam ser configuradas para respeitar o consentimento.

## O banner

Ligado:

- Na primeira visita aparece um aviso com **Aceitar todos**, **Recusar** e **Escolher**.
- Em **Escolher**, a pessoa liga ou desliga cada grupo: **Estatísticas** (Google Analytics, Umami, Plausible), **Marketing** (Meta Pixel), o Google Tag Manager nos dois e **Vídeos e conteúdo externo** (YouTube). Os **Essenciais** (tema claro ou escuro, entrar para comentar) ficam sempre ligados.
- Nada de terceiros carrega antes do aceite. A escolha fica guardada no navegador por um ano.
- O link **Cookies** no rodapé reabre as escolhas a qualquer momento.

Desligado, as ferramentas e os vídeos carregam direto, sem perguntar.

## Política de privacidade

- **Uma página do site** (recomendado): o Blup cria a nota **Política de privacidade**, já escrita a partir das ferramentas que você marcou, dos vídeos, dos comentários e da hospedagem. Revise e edite à vontade: ela é sua e o Blup nunca reescreve. Um comentário no topo da nota lembra o que conferir e não aparece no site.
- **Um endereço seu**, se a política já existe em outro lugar.

A página aparece com link no aviso de cookies e no rodapé, fica fora das listas de posts e não recebe comentários. Se mudar as ferramentas depois, atualize a nota.

## Vídeos do YouTube

Cole o link do vídeo como imagem:

```md
![Como publicar](https://www.youtube.com/watch?v=XXXXXXXXXXX)
```

No site vira o player, pelo endereço `youtube-nocookie.com`. Com o banner ligado, aparece o botão **Carregar vídeo** até a pessoa aceitar vídeos.
