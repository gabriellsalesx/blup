---
title: Podcast dos blogs
description: Nos temas Mochi e Nori, as narrações dos posts viram episódios com feed para Spotify e Apple Podcasts.
weight: 4
---
O **Podcast** é uma seção dos blogs (temas **Mochi** e **Nori**). Ela junta as narrações dos posts como episódios numa página com player próprio e publica um **feed** para Spotify, Apple Podcasts e outros apps.

> [!info] Dos temas de blog, só no Studio
> O Bento (este tema) não tem a seção Podcast. Nos blogs, ela é adicionada pelo Blup Studio. Um blog que já tem a seção continua gerando a página e o feed também pela linha de comando.

## 1. Adicionar a seção

1. Abra o blog no Studio.
2. Clique em **Seções** e, em **Podcast**, clique em **Adicionar**.
3. O Studio cria a pasta **Podcast** e a tela **Podcast** aparece na faixa lateral, no grupo Público.

## 2. Criar os episódios

Cada **post com narração** é um episódio. Narre os posts que quiser (veja [Narração dos posts](Narração%20dos%20posts.md)) e publique o site. A tela Podcast mostra quantos episódios há.

Ficam de fora: rascunhos, posts escondidos e posts com senha.

## 3. Preencher os dados do programa

Na tela **Podcast**:

| Campo | Se ficar vazio |
| --- | --- |
| Nome do podcast | O nome do site |
| Descrição (Google, compartilhamento e apps de podcast) | O texto da nota da seção |
| Autor | O autor do site |
| E-mail de contato | Sem e-mail. É por ele que o Spotify e a Apple confirmam que o podcast é seu, e ele fica público no feed |
| Categoria | Tecnologia (a lista é a da Apple Podcasts) |
| Conteúdo explícito | Não |
| Capa do podcast | A capa do episódio mais novo, depois o logo do site |

A capa ideal é um quadrado de 1400 a 3000 pixels, em JPG ou PNG. O Studio avisa quando a imagem não serve.

## 4. Enviar ao Spotify e à Apple

O feed precisa do endereço completo do site, então **publique o site** primeiro, pelo Blup Cloud, Cloudflare Workers ou GitHub Pages (veja [Onde publicar](../Publicação/Onde%20publicar.md)). Com domínio próprio no Blup Cloud, o feed usa o seu domínio. Aí a tela mostra o **Feed do podcast** de cada idioma, com **Copiar**.

- **Enviar ao Spotify**: no Spotify for Creators, escolha "já tenho um podcast" e cole o endereço. O código de confirmação chega no e-mail de contato.
- **Enviar à Apple Podcasts**: na Apple Podcasts Connect, entre com o seu Apple ID, crie um programa a partir do feed e espere a aprovação.

Faça isso uma vez só: os episódios novos chegam sozinhos aos apps quando você publica o site.

## A página do podcast no site

- A lista de episódios, do mais novo para o mais antigo, com a capa de cada post.
- Um player fixo que passa sozinho para o próximo episódio, inclusive no celular com a tela apagada.
- Voltar e avançar 15 segundos, velocidade (0,75 a 2), a lista de reprodução e as notas do episódio (o texto do post), sem sair da página.
- O ponto onde a pessoa parou fica guardado no aparelho dela.
- Um botão para copiar o feed.

> [!tip] Em vários idiomas
> Num blog em vários idiomas, cada idioma tem o seu feed, como `/podcast/feed.xml` e `/en/podcast/feed.xml`.
