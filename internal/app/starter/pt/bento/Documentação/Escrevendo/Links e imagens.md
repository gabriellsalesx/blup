---
title: Links e imagens
description: Links entre notas, links para fora, imagens, vídeos e anexos.
weight: 2
---
## Links entre notas

- **No Blup Studio**: digite `[[`, escolha a nota e pronto. Ou use `/` e **Link para outra nota**. Veja [Escrever no editor](../Blup%20Studio/Escrever%20no%20editor.md).
- **Direto no arquivo**: escreva o link Markdown com o caminho da nota.

Nos dois casos, o link fica assim, com caminho relativo:

```md
[Formatação e callouts](Formatação%20e%20callouts.md)
```

No site ele vira o link da página: [Formatação e callouts](Formatação%20e%20callouts.md). Links para um título da nota (`Nota.md#titulo`) também funcionam.

> [!tip] Renomeou ou moveu? Os links acompanham
> No Studio, **Renomear** e **Mover para…** corrigem os links das outras notas sozinhos.

> [!tip] Notas antigas com wikilinks
> Rode `blup converter`. O Blup mostra quantos links vai trocar e, se você confirmar, troca todos os wikilinks da pasta do site pelo formato padrão. Antes, guarda uma cópia das notas afetadas.

## Links para outros sites

Escreva normalmente: `[Blup](https://getblup.io)`. No Studio, o botão de link (ou `Ctrl+K`) já põe o formato e seleciona o endereço para você colar.

## Imagens

- **No Studio**: cole (`Ctrl+V`) ou arraste a imagem para o editor. Ela vai para **Anexos** e aparece no lugar.
- **Direto no arquivo**: copie a imagem para a pasta **Anexos** e escreva o link dela na nota.

Na publicação, o Blup:

- gera versões menores e em WebP de cada imagem;
- entrega ao navegador só o tamanho de que ele precisa;
- usa o texto entre colchetes como descrição (bom para acessibilidade e Google).

![O Blup no terminal](../../Anexos/blup-terminal.jpg)

## Vídeos do YouTube

Cole o link do vídeo como imagem: `![Como publicar](https://www.youtube.com/watch?v=...)`. No site vira o player. Com o banner de cookies ligado, o vídeo espera o aceite; veja [Cookies e estatísticas](../Configuração%20do%20site/Cookies%20e%20estatísticas.md).

## Outros anexos

PDFs, áudios, vídeos e arquivos ZIP também podem ser linkados, como `[Manual em PDF](../../Anexos/manual.pdf)`. Eles são publicados junto com o site. Veja a tela [Anexos](../Blup%20Studio/Anexos.md).
