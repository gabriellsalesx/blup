---
title: Escrevendo posts
date: {{date-3}}
description: Criar um post, as propriedades, links entre notas, imagens e seções. No Studio e no Obsidian.
tags: [Escrita, Studio, Obsidian]
image: "[[blup-escrever.jpg]]"
---
Para criar um post, crie uma nota na pasta do site e escreva. Só isso. O resto é opcional.

No Studio, clique em **Nova nota** no painel Arquivos, dê um nome e aperte Enter. A prévia ao lado mostra o post como ele vai ficar no site, enquanto você escreve.

## Comandos com /

No começo de uma linha (ou depois de um espaço), digite `/` e escolha: título, lista, lista de tarefas, citação, destaque (callout), tabela, bloco de código, linha divisória, link para outra nota, link externo, imagem do computador ou a data de hoje. Enter ou Tab escolhem.

Digite `@` para citar outra nota ou uma data, como hoje, amanhã ou a próxima sexta.

A **barra de formatação**, em cima do texto, tem os mesmos comandos em botões. O botão direito no editor também tem os menus **Formatar** e **Inserir**.

## As propriedades

Data, tags, capa e o resto ficam no painel **Propriedades**, em cima do editor. Clique no nome do painel para abrir ou fechar. O Studio guarda tudo no topo da própria nota.

| No Studio | Na nota | Para que serve |
| --- | --- | --- |
| **Título** | `title` | Título do post. Vazio, vale o nome da nota. |
| **Descrição** | `description` | Resumo nas listas, no Google e ao compartilhar. |
| **Data** | `date` | Data do post. |
| **Tags** | `tags` | Assuntos. Cada tag ganha uma página. |
| **Autores** | `authors` | Quem escreveu. Vazio, vale o autor do site. |
| **Capa** | `image` | A imagem de capa, como `"[[minha-capa.jpg]]"`. |
| **Rascunho** | `draft: true` | Deixa a nota fora do site. Ela aparece só na prévia. |
| **Escondida** | `hidden: true` | Publica, mas tira das listas e do Google. |
| **Senha** | `password` | Pede uma senha para ler. Veja [Posts com senha](Posts%20com%20senha.md). |

No Obsidian, as mesmas propriedades ficam no bloco do topo da nota: digite `---` na primeira linha ou use o comando "Adicionar propriedade".

## Links entre notas

No Studio, digite `[[` e escolha a nota pelo nome (Tab escolhe), ou use **Link para outra nota** no menu `/`. No texto, o link aparece só pelo nome, em azul. Gravado na nota, ele fica assim:

```md
[Formatação que funciona](Formatação%20que%20funciona.md)
```

No site, ele vira o link da página. Exemplo: [Formatação que funciona](Formatação%20que%20funciona.md).

Para um link de outro site, use **Link** (Ctrl+K).

> [!tip] Renomeou uma nota?
> No Studio, pode renomear sem medo: ele atualiza sozinho os links que apontam para ela.

No Obsidian, arraste uma nota para dentro de outra ou digite `[[`. Com o ajuste de links que o Blup faz no Obsidian, o link é gravado no mesmo formato acima. Já tem notas antigas com wikilinks? Rode `blup converter`: o Blup troca os links antigos pelo formato padrão e mostra o que vai mudar antes.

## Imagens

Cole a imagem direto na nota, ou arraste, ou use **Imagem do computador**. O Studio guarda o arquivo na pasta **Anexos** sozinho e a imagem aparece no lugar, dentro do texto. No Obsidian, arraste a imagem para a nota, como sempre.

O Blup gera versões menores e modernas (WebP) de cada imagem, e o navegador baixa só o tamanho que precisa.

![Blup escrevendo num cartão de nota](../Anexos/blup-escrever.jpg)

O texto entre colchetes é a descrição da imagem: ajuda quem usa leitor de tela e o Google.

A tela **Anexos**, no menu ao lado do Studio, mostra todas as imagens e arquivos que as notas usam. O filtro **Sem uso** acha o que sobrou.

## Organizar em seções

Crie uma pasta na pasta do site, como **Viagens**, e coloque posts nela: o site ganha a seção Viagens, com página própria e item no menu. Para dar título e descrição à seção, crie uma nota **_index** dentro dela, como a desta seção.

## Corretor ortográfico

O Studio sublinha as palavras com erro enquanto você escreve. Clique com o botão direito na palavra para ver as sugestões ou adicionar a palavra ao dicionário. O corretor roda no seu computador: o texto não sai dele.
