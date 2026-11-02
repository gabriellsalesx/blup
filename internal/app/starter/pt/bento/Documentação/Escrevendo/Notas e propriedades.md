---
title: Notas e propriedades
description: Crie uma nota e ela vira página. As propriedades controlam o resto.
weight: 1
---
Para criar uma página, crie uma nota na pasta certa. As **propriedades** (o bloco no topo da nota, entre linhas `---`) são opcionais e controlam como ela aparece.

- **No Blup Studio**: clique em **Nova nota** (ou `Ctrl+N`) e preencha o painel **Propriedades**, acima do texto. Veja [Propriedades da nota](../Blup%20Studio/Propriedades%20da%20nota.md).
- **Direto no arquivo**: escreva as propriedades no topo da nota, entre duas linhas `---`, em qualquer editor de texto.

## Em todas as páginas

| Propriedade | Para que serve |
| --- | --- |
| `title` | Título da página. Sem ela, vale o nome da nota. |
| `linkTitle` | Título curto para menus e cartões. |
| `description` | Resumo nas listas, no Google e nas redes sociais. |
| `weight` | Ordem no menu lateral e nas listas: menor vem primeiro. |
| `draft` | `true` deixa a nota fora do site (aparece só na prévia). |
| `hidden` | `true` publica, mas tira das listas e pede ao Google para não mostrar. |
| `password` | Protege a página com senha. Veja [Senhas e páginas escondidas](../Configuração%20do%20site/Senhas%20e%20páginas%20escondidas.md). |

## Nos posts do blog e no changelog

| Propriedade | Para que serve |
| --- | --- |
| `date` | Data. Sem ela, vale `created`. |
| `updated` | Data da última atualização. |
| `tags` | Assuntos; cada tag ganha uma página. |
| `image` | Capa, como `"[[minha-capa.jpg]]"`. |
| `authors` | Quem escreveu, como `[Ana, Bruno]`. |
| `featured` | `true` escolhe o post de destaque. |

## No roadmap e na central de ajuda

| Propriedade | Para que serve |
| --- | --- |
| `status` | Coluna do roadmap: planejado, em andamento ou concluído. |
| `category` | Etiqueta do card no roadmap. |
| `icon` | Ícone de uma coleção da central de ajuda (na _index dela). |

## Que o Blup preenche sozinho

| Propriedade | Quem preenche |
| --- | --- |
| `audio`, `audio_hash`, `audio_seconds` | A narração. Veja [Narração dos posts](../Recursos%20com%20IA/Narração%20dos%20posts.md). |
| `translationKey`, `translation_hash`, `ai_translated` | A tradução. Veja [Sites em vários idiomas](../Configuração%20do%20site/Sites%20em%20vários%20idiomas.md). |
| `created`, `updated` | O Studio, nas notas dos cofres. |

> [!tip] Modelos prontos
> Sites criados pela linha de comando ganham a pasta **Modelos**, com modelos de página da documentação, item do roadmap e versão do changelog. Copie um modelo para começar uma nota com as propriedades certas.
