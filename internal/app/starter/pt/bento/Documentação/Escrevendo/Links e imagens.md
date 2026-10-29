---
title: Links e imagens
description: Links entre notas, links para fora, imagens e anexos.
weight: 2
---
## Links entre notas

Digite `[[` e escolha a nota, ou arraste a nota para o texto. Com o ajuste que o Blup fez no Obsidian, o link é gravado no formato padrão, com caminho relativo:

```md
[Formatação e callouts](Formatação%20e%20callouts.md)
```

No site ele vira o link da página: [Formatação e callouts](Formatação%20e%20callouts.md). Links para um título da nota (`Nota.md#titulo`) também funcionam.

> [!tip] Notas antigas com [[wikilinks]]
> Rode `blup converter`. O Blup mostra quantos links vai trocar e, se você confirmar, troca todos os `[[links]]` da pasta do site pelo formato padrão.

## Links para outros sites

Escreva normalmente: `[Obsidian](https://obsidian.md)`. Eles abrem no mesmo lugar, com segurança (`noopener`).

## Imagens

Arraste a imagem para a nota. O Blup:

- gera versões menores e em WebP de cada imagem;
- entrega ao navegador só o tamanho de que ele precisa;
- usa o texto entre colchetes como descrição (bom para acessibilidade e Google).

![O Blup no terminal](../../Anexos/blup-terminal.jpg)

## Vídeos do YouTube

Cole o link do vídeo como imagem: `![Como publicar](https://www.youtube.com/watch?v=...)`. No site vira o player. Com o banner de cookies ligado, o vídeo espera o aceite; veja [Cookies e estatísticas](../Configuração/Cookies%20e%20estatísticas.md).

## Outros anexos

PDFs, áudios, vídeos e arquivos ZIP do cofre também podem ser linkados: `[Manual em PDF](../../Anexos/manual.pdf)`. Eles são publicados junto com o site.
