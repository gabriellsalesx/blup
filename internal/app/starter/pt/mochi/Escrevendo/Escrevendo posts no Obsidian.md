---
title: Escrevendo posts no Obsidian
date: {{date-3}}
description: As propriedades que o site entende, links entre notas, imagens e modelos.
tags: [Obsidian]
image: "[[blup-escrever.jpg]]"
---
Para criar um post, crie uma nota na pasta do site. Só isso. O resto é opcional e fica nas **propriedades**, o bloco no topo da nota (no Obsidian, digite `---` na primeira linha ou use o comando "Adicionar propriedade").

## As propriedades

| Propriedade | Para que serve |
| --- | --- |
| `title` | Título do post. Sem ela, vale o nome da nota. |
| `date` | Data do post. Sem ela, vale `created` ou a data do arquivo. |
| `description` | Resumo nos cards, no Google e nas redes sociais. |
| `tags` | Assuntos. Cada tag ganha uma página. |
| `image` | Capa do post, como `"[[minha-capa.jpg]]"`. |
| `authors` | Quem escreveu, como `[Ana, Bruno]`. Sem ela, vale o autor do site. |
| `draft` | `true` deixa a nota fora do site (aparece só na prévia). |
| `hidden` | `true` publica, mas tira das listas e do Google. |
| `password` | Protege o post com senha. Veja [Posts com senha](Posts%20com%20senha.md). |
| `updated` | Data da última atualização, quando quiser mostrar. |

Abra este post no Obsidian para ver as propriedades dele no topo.

## Links entre notas

Arraste uma nota para dentro de outra, ou digite `[[` e escolha a nota. Com o ajuste que o Blup fez no Obsidian, o link é gravado assim:

```md
[Formatação que funciona](Formatação%20que%20funciona.md)
```

No site, ele vira o link da página. Exemplo: [Formatação que funciona](Formatação%20que%20funciona.md).

> [!tip] Já tem notas com [[wikilinks]]?
> Rode `blup converter`: o Blup troca os `[[links]]` antigos pelo formato padrão, em todas as notas da pasta do site, e mostra o que vai mudar antes.

## Imagens

Arraste a imagem para a nota, como sempre. O Blup gera versões menores e modernas (WebP) de cada imagem, e o navegador baixa só o tamanho que precisa.

![Blup escrevendo num cartão de nota](../Anexos/blup-escrever.jpg)

O texto entre colchetes é a descrição da imagem: ajuda quem usa leitor de tela e o Google.

## Modelos

O Blup criou a pasta **Templates** no seu cofre com um modelo de post e configurou o plugin Modelos do Obsidian. Para começar um post com as propriedades certas, use o comando **Modelos: Inserir modelo**. A pasta de modelos nunca vai para o site.

## Organizar em seções

Crie uma subpasta na pasta do site, como **Viagens**, e coloque posts nela: o site ganha a seção Viagens, com página própria e item no menu. Para dar título e descrição à seção, crie uma nota **_index** dentro dela, como a desta seção.
