---
title: Prévia e publicação
date: {{date-2}}
description: Veja o site enquanto escreve e publique de graça no Cloudflare Pages ou no GitHub Pages.
tags: [Começando, CLI]
image: "[[blup-publicar.jpg]]"
---
## A prévia

Rode `blup ver` (ou escolha a prévia no menu). O site abre no navegador e **atualiza sozinho** a cada nota salva no Obsidian. Para parar, aperte Ctrl+C no terminal.

A prévia mostra também os rascunhos, para você conferir antes de publicar.

## Publicar

Rode `blup publicar`. Na primeira vez o Blup pergunta onde publicar. Os dois são gratuitos:

- **Cloudflare Pages**: rápido no mundo todo, com endereço como `seusite.pages.dev`.
- **GitHub Pages**: endereço como `seunome.github.io/seusite`.

O Blup abre a página certa para criar a chave de acesso, já preenchida, e explica o que clicar. Você cola a chave no terminal uma vez; ela fica guardada só no seu computador.

Das próximas vezes, `blup publicar` só envia o que mudou.

> [!tip] Publicar é sempre igual
> Escreveu, conferiu na prévia, rodou `blup publicar`. Não existe servidor para manter nem banco de dados.

## O que vai para o ar

- Todas as notas da pasta do site, menos:
  - notas com `draft: true`;
  - pastas que começam com ponto, a pasta de modelos (Templates) e a pasta de autores, que vira só dados dos perfis.
- As imagens e anexos que as notas usam, otimizados.
- A busca, o feed RSS, o mapa do site e o cartão de compartilhamento de cada página.
