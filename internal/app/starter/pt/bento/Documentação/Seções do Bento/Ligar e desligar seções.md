---
title: Ligar e desligar seções
description: Todas as seções são opcionais. Adicione, tire e restaure quando quiser.
weight: 9
---
## Pelo Blup Studio

Clique em **Seções** na faixa lateral (só no Bento). A tela mostra cada parte do site de produto, com uma linha sobre ela:

| Seção | O que é |
| --- | --- |
| Documentação | Páginas com menu lateral e sumário |
| Roadmap | Quadro com planejado, em andamento e concluído |
| Changelog | O que mudou em cada versão |
| Central de ajuda | Coleções de artigos com busca |
| Blog | Novidades com capa e tags |
| Feedback | Quadro onde quem visita pede recursos, conta bugs e vota. Só no Blup Studio |

- **Adicionar** cria a pasta da seção com notas de exemplo que ensinam a usar.
- As que já estão no site aparecem como **No site**. Tirar uma seção manda a pasta para a **Lixeira**, e o aviso tem **Desfazer**.
- Uma seção tirada pode ser restaurada pela Lixeira.

A prévia se atualiza sozinha depois de cada mudança.

## Pela linha de comando

Rode `blup personalizar` e escolha **Seções do site**. O Blup mostra as que faltam e cria a pasta de cada uma, com notas de exemplo. Para tirar, apague a pasta. O Feedback não aparece nesta lista.

## À mão

Uma pasta na raiz do site com uma nota **_index** que tenha a propriedade **blup**:

| Seção | Propriedade |
| --- | --- |
| Documentação | `blup: docs` |
| Roadmap | `blup: roadmap` |
| Changelog | `blup: changelog` |
| Central de ajuda | `blup: help` |
| Blog | `blup: blog` |
| Feedback | `blup: feedback` |
| Perfis dos autores | `blup: authors` |

## Renomear e reordenar

- O nome da pasta é livre: "Docs", "Guia", "Manual"... o que vale é a propriedade **blup**.
- O **title** da _index é o nome no menu e no topo da seção.
- O **weight** da _index ordena as seções no menu automático e nos cartões da página inicial.
