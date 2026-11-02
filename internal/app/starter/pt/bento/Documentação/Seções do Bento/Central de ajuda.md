---
title: Central de ajuda
description: Coleções de artigos com ícones e busca, como as centrais de suporte.
weight: 4
---
A central de ajuda é uma pasta cuja **_index** tem `blup: help`. A página de entrada tem uma busca e as **coleções** em cartões.

![O Blup com uma boia](../../Anexos/blup-ajuda.jpg)

## Coleções e artigos

- Cada **subpasta** é uma coleção.
- Cada **nota** dentro dela é um artigo.
- A **_index** da coleção dá título, descrição e ícone.

```yaml
---
title: Primeiros passos
description: O básico para começar.
icon: rocket
weight: 1
---
```

## Ícones

**No Blup Studio**: abra a nota **_index** da coleção. Nas propriedades aparece **Ícone da coleção**: clique para abrir a grade de ícones e busque pelo nome em inglês ou por palavras comuns em português (foguete, livro, ajuda...). **Automático** tira o ícone e deixa o tema alternar.

**No Obsidian**: a propriedade **icon** aceita qualquer nome da biblioteca [Lucide](https://lucide.dev/icons), com mais de 2 mil ícones: `rocket`, `pen-line`, `globe`, `life-buoy`, `credit-card`, `shield-check`, `users`... Copie o nome do site da Lucide. Sem icon, o tema escolhe um.

## Título e menu

Na _index da central, **title** é a pergunta grande da entrada ("Como podemos ajudar?") e **linkTitle** é o nome curto no menu ("Central de ajuda").
