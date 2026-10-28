---
title: Notas e propriedades
description: Crie uma nota e ela vira página. As propriedades controlam o resto.
weight: 1
---
Para criar uma página, crie uma nota na pasta certa. As **propriedades** (o bloco no topo da nota) são opcionais e controlam como ela aparece. No Obsidian, digite `---` na primeira linha ou use o comando **Adicionar propriedade**.

## Em todas as páginas

| Propriedade | Para que serve |
| --- | --- |
| `title` | Título da página. Sem ela, vale o nome da nota. |
| `linkTitle` | Título curto para menus e listas. |
| `description` | Resumo nas listas, no Google e nas redes sociais. |
| `weight` | Ordem no menu lateral e nas listas: menor vem primeiro. |
| `draft` | `true` deixa a nota fora do site (aparece só na prévia). |
| `hidden` | `true` publica, mas tira das listas e pede ao Google para não mostrar. |
| `password` | Protege a página com senha. Veja [Senhas e páginas escondidas](../Configuração/Senhas%20e%20páginas%20escondidas.md). |

## Nos posts do blog e no changelog

| Propriedade | Para que serve |
| --- | --- |
| `date` | Data. Sem ela, vale `created`. |
| `updated` | Data da última atualização. |
| `tags` | Assuntos; cada tag ganha uma página. |
| `image` | Capa, como `"[[minha-capa.jpg]]"`. |
| `authors` | Quem escreveu, como `[Ana, Bruno]`. |

## No roadmap e na central de ajuda

| Propriedade | Para que serve |
| --- | --- |
| `status` | Coluna do roadmap: planejado, em andamento ou concluído. |
| `category` | Etiqueta do card no roadmap. |
| `icon` | Ícone de uma coleção da central de ajuda (na _index dela). |

> [!tip] Modelos prontos
> A pasta **Templates** do cofre tem modelos de página da documentação, item do roadmap e versão do changelog. Use o comando **Modelos: Inserir modelo** do Obsidian para começar uma nota com as propriedades certas.
