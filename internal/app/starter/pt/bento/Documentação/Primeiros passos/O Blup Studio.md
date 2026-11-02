---
title: O Blup Studio
description: O app que faz o caminho inteiro, do texto ao site no ar.
weight: 4
---
As notas do Blup são **arquivos Markdown comuns** numa pasta do seu computador. O **Blup Studio** abre essa pasta e junta tudo num app só: escrever, ver o site, configurar e publicar. Para quem está começando, é o caminho mais curto.

## O que o Studio faz

| Recurso | Onde ler |
| --- | --- |
| Editor com prévia ao vivo do site, lado a lado | [Prévia ao vivo](../Blup%20Studio/Prévia%20ao%20vivo.md) |
| Telas Site, Seções, Autores, Idiomas, Comentários, Cookies, Podcast e Publicar | [Configurações do site](../Configuração%20do%20site/Configurações%20do%20site.md) |
| Submenus no menu do topo, arrastando | [Menu, submenus e botão](../Configuração%20do%20site/Menu,%20submenus%20e%20botão.md) |
| Tradução das notas para outros idiomas | [Tradução](../Recursos%20com%20IA/Tradução.md) |
| Narração dos posts em áudio | [Narração dos posts](../Recursos%20com%20IA/Narração%20dos%20posts.md) |
| Quadro de Feedback no Bento | [Feedback](../Seções%20do%20Bento/Feedback.md) |
| Cofres de anotações com nota do dia, mapa de notas, canvas, desenhos e bancos de dados | [O que é um cofre](../Cofres%20de%20notas/O%20que%20é%20um%20cofre.md) |
| Sincronização e publicação pelo Blup Cloud | [Sincronizar as notas](../Blup%20Cloud/Sincronizar%20as%20notas.md) |
| Terminal com agentes de IA e desfazer o que a IA mudou | [Terminal e agentes de IA](../Blup%20Studio/Terminal%20e%20agentes%20de%20IA.md) |
| Lixeira, histórico de versões e cópia diária das notas | [Lixeira e histórico de versões](../Blup%20Studio/Lixeira%20e%20histórico.md) |

> [!tip] As notas continuam suas
> Como as notas são arquivos comuns, dá para abrir e editar em qualquer editor de texto. O Studio percebe quando uma nota muda fora dele e atualiza a tela.

## Também existe a linha de comando

Quem prefere o terminal pode usar o `blup` para ver a prévia (`blup ver`), publicar (`blup publicar`) e mudar as configurações (`blup personalizar`). Ele trabalha com as mesmas notas. Veja [Comandos](../Linha%20de%20comando/Comandos.md).

Pelos dois caminhos dá para:

- escrever em Markdown, com propriedades, callouts, tabelas, fórmulas e comentários `%%`;
- gerar o site, ver a prévia e publicar no Cloudflare Pages ou no GitHub Pages;
- sincronizar as notas com um repositório privado do GitHub;
- usar senhas, páginas escondidas, rascunhos, cookies, comentários e reações.

## Links entre notas

Sites entendem links Markdown com caminho relativo, como `[Instalação](Instalação.md)`. O Studio sempre grava os links assim, mesmo quando você digita `[[`. Notas antigas com wikilinks podem ser convertidas com `blup converter`. Veja [Links e imagens](../Escrevendo/Links%20e%20imagens.md).
